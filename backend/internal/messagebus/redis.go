package messagebus

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
	"sync"
	"time"

	goredis "github.com/redis/go-redis/v9"
	goredislogging "github.com/redis/go-redis/v9/logging"
	"go.uber.org/zap"
)

// envelopeSeparator separates the serverID prefix from the payload in Redis messages.
// This allows receivers to identify and skip messages they published themselves.
var envelopeSeparator = []byte{0xFF, 0x00}

// RedisMessageBus implements MessageBus using Redis Pub/Sub
type RedisMessageBus struct {
	client        *goredis.Client
	logger        *zap.Logger
	subscriptions map[string]*subscription // roomID -> subscription
	subMu         sync.RWMutex
	serverID      string
}

type subscription struct {
	pubsub  *goredis.PubSub
	channel chan []byte
	cancel  context.CancelFunc
}

// NewRedisMessageBus creates a new Redis-backed message bus
func NewRedisMessageBus(redisURL string, serverID string, logger *zap.Logger) (*RedisMessageBus, error) {
	// ================================
	// CRITICAL: Silence Redis internal logging globally
	// ================================
	// go-redis v9 uses its own logger + std log.
	// Disable go-redis logger and discard std log to remove lock contention.
	goredislogging.Disable()
	log.SetOutput(io.Discard)

	opts, err := goredis.ParseURL(redisURL)
	if err != nil {
		logger.Error("Redis URL failed",
			zap.String("url", redisURL),
			zap.Error(err),
		)
		return nil, err
	}

	// ================================
	// CRITICAL FIX: Prevent Redis connection churn to reduce internal logging
	// ================================
	// Redis client uses Go's standard log package for connection pool events.
	// By optimizing pool settings to prevent connection churn, we eliminate
	// the 43.26s mutex contention (99.50% of total delay) caused by
	// log.(*Logger).output mutex in connection dial operations.

	// ================================
	// Connection Pool Configuration (tuned for worker pool architecture)
	// ================================
	// With 50 publish workers + 10 PubSub subscriptions + awareness ops,
	// we need ~100 concurrent connections max, not 2000.
	// Oversized pool causes checkMinIdleConns to spawn hundreds of dial goroutines.
	opts.PoolSize = 200

	// MinIdleConns: keep a small base ready for the worker pool
	// 50 workers + headroom. Too high = hundreds of maintenance goroutines dialing.
	opts.MinIdleConns = 30

	// PoolTimeout: How long to wait for a connection from the pool
	// - With bounded worker pool, fail fast is better than blocking workers
	opts.PoolTimeout = 5 * time.Second

	// ConnMaxIdleTime: Close idle connections after this duration
	// - Set to 0 to never close idle connections (good for stable load)
	// - Prevents connection churn that causes dialConn overhead
	opts.ConnMaxIdleTime = 0

	// ConnMaxLifetime: Maximum lifetime of any connection
	// - Set high to avoid unnecessary reconnections during stable operation
	// - Redis will handle stale connections via TCP keepalive
	opts.ConnMaxLifetime = 1 * time.Hour

	client := goredis.NewClient(opts)

	// ================================
	// Connection Pool Pre-warming
	// ================================
	// Force the pool to establish MinIdleConns connections BEFORE accepting traffic.
	// This prevents the "thundering herd" problem where all 1000 users dial simultaneously.
	logger.Info("Pre-warming Redis connection pool...", zap.Int("target_conns", opts.MinIdleConns))

	warmupCtx, warmupCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer warmupCancel()

	var wg sync.WaitGroup
	for i := 0; i < opts.MinIdleConns; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = client.Ping(warmupCtx).Err() // Ignore errors, best-effort warmup
		}()
	}
	wg.Wait() // Block until warmup completes

	logger.Info("Connection pool pre-warming completed")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	logger.Info("Trying to reach Redis...", zap.String("addr", opts.Addr))

	if err := client.Ping(ctx).Err(); err != nil {
		logger.Error("Redis connection Ping Failed",
			zap.String("addr", opts.Addr),
			zap.Error(err),
		)
		_ = client.Close()
		return nil, err
	}

	logger.Info("Redis connected successfully", zap.String("addr", opts.Addr))
	return &RedisMessageBus{
		client:        client,
		logger:        logger,
		subscriptions: make(map[string]*subscription),
		serverID:      serverID,
	}, nil
}

// Publish sends a binary message to a room channel, prepending the serverID envelope
func (r *RedisMessageBus) Publish(ctx context.Context, roomID string, data []byte) error {
	channel := fmt.Sprintf("room:%s:messages", roomID)

	// Prepend serverID + separator so receivers can filter self-echoes
	envelope := make([]byte, 0, len(r.serverID)+len(envelopeSeparator)+len(data))
	envelope = append(envelope, []byte(r.serverID)...)
	envelope = append(envelope, envelopeSeparator...)
	envelope = append(envelope, data...)

	err := r.client.Publish(ctx, channel, envelope).Err()

	if err != nil {
		r.logger.Error("failed to publish message",
			zap.String("roomID", roomID),
			zap.Int("data_len", len(data)),
			zap.String("channel", channel),
			zap.Error(err),
		)
		return fmt.Errorf("redis publish failed: %w", err)
	}
	r.logger.Debug("published message successfully",
		zap.String("roomID", roomID),
		zap.Int("data_len", len(data)),
	)
	return nil
}

// Subscribe creates a subscription to a room channel
func (r *RedisMessageBus) Subscribe(ctx context.Context, roomID string) (<-chan []byte, error) {
	r.subMu.Lock()
	defer r.subMu.Unlock()

	if sub, exists := r.subscriptions[roomID]; exists {
		r.logger.Debug("returning existing subscription", zap.String("roomID", roomID))
		return sub.channel, nil
	}
	r.logger.Info("Creating new Redis subscription",
		zap.String("roomID", roomID),
		zap.Int("current_map_size", len(r.subscriptions)),
	)

	subCtx, cancel := context.WithCancel(context.Background())
	msgChan := make(chan []byte, 256)
	sub := &subscription{
		channel: msgChan,
		cancel:  cancel,
	}
	r.subscriptions[roomID] = sub

	go r.readLoop(subCtx, roomID, sub, msgChan)

	r.logger.Info("successfully subscribed to room",
		zap.String("roomID", roomID),
	)
	return msgChan, nil
}

// readLoop uses ReceiveTimeout to avoid the go-redis channel helper and its health-check goroutine.
func (r *RedisMessageBus) readLoop(ctx context.Context, roomID string, sub *subscription, msgChan chan []byte) {
	defer func() {
		close(msgChan)
		r.logger.Info("forwarder stopped", zap.String("roomID", roomID))
	}()

	channel := fmt.Sprintf("room:%s:messages", roomID)
	backoff := 200 * time.Millisecond
	maxBackoff := 5 * time.Second

	for {
		if ctx.Err() != nil {
			r.logger.Info("stopping read loop due to context", zap.String("roomID", roomID))
			return
		}

		pubsub := r.client.Subscribe(ctx, channel)
		if _, err := pubsub.Receive(ctx); err != nil {
			pubsub.Close()
			if ctx.Err() != nil {
				return
			}
			time.Sleep(backoff)
			if backoff < maxBackoff {
				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
			}
			continue
		}

		// attach latest pubsub for Unsubscribe to close
		r.subMu.Lock()
		if cur, ok := r.subscriptions[roomID]; ok && cur == sub {
			sub.pubsub = pubsub
		} else {
			r.subMu.Unlock()
			pubsub.Close()
			return
		}
		r.subMu.Unlock()

		backoff = 200 * time.Millisecond
		if err := r.receiveOnce(ctx, roomID, pubsub, msgChan); err != nil {
			pubsub.Close()
			if ctx.Err() != nil {
				return
			}
			time.Sleep(backoff)
			if backoff < maxBackoff {
				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
			}
		}
	}
}

func (r *RedisMessageBus) receiveOnce(ctx context.Context, roomID string, pubsub *goredis.PubSub, msgChan chan []byte) error {
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		msg, err := pubsub.ReceiveTimeout(ctx, 5*time.Second)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				return err
			}
			if errors.Is(err, goredis.Nil) {
				continue
			}
			r.logger.Warn("pubsub receive error, closing subscription",
				zap.String("roomID", roomID),
				zap.Error(err),
			)
			return err
		}

		switch m := msg.(type) {
		case *goredis.Message:
			raw := []byte(m.Payload)
			sepIdx := bytes.Index(raw, envelopeSeparator)
			if sepIdx == -1 {
				r.logger.Warn("received message without server envelope, skipping",
					zap.String("roomID", roomID))
				continue
			}

			senderID := string(raw[:sepIdx])
			if senderID == r.serverID {
				continue
			}
			payload := raw[sepIdx+len(envelopeSeparator):]

			select {
			case msgChan <- payload:
				r.logger.Debug("message forwarded",
					zap.String("roomID", roomID),
					zap.String("from_server", senderID),
					zap.Int("size", len(payload)))
			default:
				r.logger.Warn("message dropped: consumer too slow",
					zap.String("roomID", roomID))
			}
		case *goredis.Subscription:
			continue
		default:
			continue
		}
	}
}

// Unsubscribe stops listening to a room
func (r *RedisMessageBus) Unsubscribe(ctx context.Context, roomID string) error {
	r.subMu.Lock()
	// Check if subscription exists
	sub, ok := r.subscriptions[roomID]
	if !ok {
		r.subMu.Unlock()
		r.logger.Debug("unsubscribe ignored: room not found", zap.String("roomID", roomID))
		return nil
	}
	delete(r.subscriptions, roomID)
	r.subMu.Unlock()

	// Cancel the context (stops readLoop goroutine)
	sub.cancel()

	// Close the Redis pubsub connection (outside lock to avoid blocking others)
	if sub.pubsub != nil {
		if err := sub.pubsub.Close(); err != nil {
			r.logger.Error("failed to close redis pubsub",
				zap.String("roomID", roomID),
				zap.Error(err),
			)
		}
	}
	r.logger.Info("successfully unsubscribed", zap.String("roomID", roomID))

	return nil
}

// SetAwareness caches awareness data in Redis Hash
func (r *RedisMessageBus) SetAwareness(ctx context.Context, roomID string, clientID uint64, data []byte) error {
	key := fmt.Sprintf("room:%s:awareness", roomID)
	field := fmt.Sprintf("%d", clientID)

	if err := r.client.HSet(ctx, key, field, data).Err(); err != nil {
		r.logger.Error("failed to set awareness data",
			zap.String("roomID", roomID),
			zap.Uint64("clientID", clientID),
			zap.Error(err),
		)
		return fmt.Errorf("hset awareness failed: %w", err)
	}

	// Set expiration on the Hash (30s)
	if err := r.client.Expire(ctx, key, 30*time.Second).Err(); err != nil {
		r.logger.Warn("failed to set expiration on awareness key",
			zap.String("key", key),
			zap.Error(err),
		)
	}
	r.logger.Debug("awareness updated",
		zap.String("roomID", roomID),
		zap.Uint64("clientID", clientID),
		zap.Int("data_len", len(data)),
	)

	return nil
}

// GetAllAwareness retrieves all cached awareness for a room
func (r *RedisMessageBus) GetAllAwareness(ctx context.Context, roomID string) (map[uint64][]byte, error) {
	// 1. 构建 Redis Hash key
	key := fmt.Sprintf("room:%s:awareness", roomID)

	// 2. 从 Redis 获取所有字段
	// HGetAll 会返回该 Hash 下所有的 field 和 value
	result, err := r.client.HGetAll(ctx, key).Result()
	if err != nil {
		r.logger.Error("failed to HGetAll awareness",
			zap.String("roomID", roomID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("redis hgetall failed: %w", err)
	}

	// 3. 转换数据类型：map[string]string -> map[uint64][]byte
	awarenessMap := make(map[uint64][]byte, len(result))

	for field, value := range result {
		// 解析 field (clientID) 为 uint64
		// 虽然提示可以用 Sscanf，但在 Go 中 strconv.ParseUint 通常更高效且稳健
		clientID, err := strconv.ParseUint(field, 10, 64)
		if err != nil {
			r.logger.Warn("invalid clientID format in awareness hash",
				zap.String("roomID", roomID),
				zap.String("field", field),
			)
			continue // 跳过异常字段，保证其他数据正常显示
		}

		// 将 string 转换为 []byte
		awarenessMap[clientID] = []byte(value)
	}

	// 4. 记录日志
	r.logger.Debug("retrieved all awareness data",
		zap.String("roomID", roomID),
		zap.Int("client_count", len(awarenessMap)),
	)

	return awarenessMap, nil
}

// DeleteAwareness removes awareness cache for a client
func (r *RedisMessageBus) DeleteAwareness(ctx context.Context, roomID string, clientID uint64) error {
	key := fmt.Sprintf("room:%s:awareness", roomID)
	field := fmt.Sprintf("%d", clientID)

	if err := r.client.HDel(ctx, key, field).Err(); err != nil {
		r.logger.Error("failed to delete awareness data",
			zap.String("roomID", roomID),
			zap.Uint64("clientID", clientID),
			zap.Error(err),
		)
		return fmt.Errorf("delete awareness failed: %w", err)
	}
	return nil
}

// IsHealthy checks Redis connectivity
func (r *RedisMessageBus) IsHealthy() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// 只有 Ping 成功且没有报错，才认为服务是健康的
	if err := r.client.Ping(ctx).Err(); err != nil {
		r.logger.Warn("Redis health check failed", zap.Error(err))
		return false
	}

	return true
}

// StartHealthMonitoring runs periodic health checks
func (r *RedisMessageBus) StartHealthMonitoring(ctx context.Context, interval time.Duration, onStatusChange func(bool)) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	previouslyHealthy := true

	for {
		select {
		case <-ctx.Done():
			r.logger.Info("stopping health monitoring")
			return
		case <-ticker.C:
			// 检查当前健康状态
			currentlyHealthy := r.IsHealthy()

			// 如果状态发生变化（健康 -> 亚健康，或 亚健康 -> 恢复）
			if currentlyHealthy != previouslyHealthy {
				r.logger.Warn("Redis health status changed",
					zap.Bool("old_status", previouslyHealthy),
					zap.Bool("new_status", currentlyHealthy),
				)

				// 触发外部回调逻辑
				if onStatusChange != nil {
					onStatusChange(currentlyHealthy)
				}

				// 更新历史状态
				previouslyHealthy = currentlyHealthy
			}
		}
	}
}

func (r *RedisMessageBus) Close() error {
	r.subMu.Lock()
	r.logger.Info("gracefully shutting down message bus", zap.Int("active_subs", len(r.subscriptions)))
	subs := r.subscriptions
	r.subscriptions = make(map[string]*subscription)
	r.subMu.Unlock()

	// 1. 关闭所有正在运行的订阅
	for roomID, sub := range subs {
		// 停止对应的 readLoop 协程
		sub.cancel()

		// 关闭物理连接
		if sub.pubsub != nil {
			if err := sub.pubsub.Close(); err != nil {
				r.logger.Error("failed to close pubsub connection",
					zap.String("roomID", roomID),
					zap.Error(err),
				)
			}
		}
	}

	// 2. 关闭主 Redis 客户端连接池
	if err := r.client.Close(); err != nil {
		r.logger.Error("failed to close redis client", zap.Error(err))
		return err
	}

	r.logger.Info("Redis message bus closed successfully")
	return nil
}

// ClearAllAwareness 彻底删除该房间的感知数据 Hash
func (r *RedisMessageBus) ClearAllAwareness(ctx context.Context, roomID string) error {
	key := fmt.Sprintf("room:%s:awareness", roomID)
	// 直接使用 Del 命令删除整个 Key
	return r.client.Del(ctx, key).Err()
}

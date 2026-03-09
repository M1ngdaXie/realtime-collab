package hub

import (
	"context"
	"encoding/base64"
	"strconv"
	"sync"
	"time"

	"github.com/M1ngdaXie/realtime-collab/internal/messagebus"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type Message struct {
	RoomID string
	Data   []byte
	sender *Client
}

type Client struct {
	ID             string
	UserID         *uuid.UUID // Authenticated user ID (nil for public share access)
	UserName       string     // User's display name for presence
	UserAvatar     *string    // User's avatar URL for presence
	Permission     string     // User's permission level: "owner", "edit", "view"
	Conn           *websocket.Conn
	send           chan []byte
	sendMu         sync.Mutex
	sendClosed     bool
	hub            *Hub
	roomID         string
	mutex          sync.Mutex
	unregisterOnce sync.Once
	failureCount   int
	failureMu      sync.Mutex
	observedYjsIDs map[uint64]uint64 // clientID -> maxClock
	idsMu          sync.Mutex
}
type Room struct {
	ID             string
	clients        map[*Client]bool
	mu             sync.RWMutex
	cancel         context.CancelFunc
	reconnectCount int // Track Redis reconnection attempts for debugging
}

type Hub struct {
	rooms      map[string]*Room
	mu         sync.RWMutex
	Register   chan *Client  // Exported
	Unregister chan *Client  // Exported
	Broadcast  chan *Message // Exported

	//redis pub/sub config
	messagebus   messagebus.MessageBus
	logger       *zap.Logger
	serverID     string
	fallbackMode bool

	// P0 fix: bounded worker pool for Redis Publish
	publishQueue chan *Message // buffered queue consumed by fixed workers
	publishDone  chan struct{} // close to signal workers to exit

	subscribeMu sync.Mutex

	// Bounded worker pool for Redis SetAwareness
	awarenessQueue chan awarenessItem

	// Stream persistence worker pool (P1: Redis Streams durability)
	streamQueue chan *Message // buffered queue for XADD operations
	streamDone  chan struct{} // close to signal stream workers to exit
}

const (
	// publishWorkerCount is the number of fixed goroutines consuming from publishQueue.
	// 50 workers can handle ~2000 msg/sec assuming ~25ms avg Redis RTT per publish.
	publishWorkerCount = 50

	// publishQueueSize is the buffer size for the publish queue channel.
	publishQueueSize = 4096

	// awarenessWorkerCount is the number of fixed goroutines consuming from awarenessQueue.
	awarenessWorkerCount = 8

	// awarenessQueueSize is the buffer size for awareness updates.
	awarenessQueueSize = 4096

	// streamWorkerCount is the number of fixed goroutines consuming from streamQueue.
	// 50 workers match publish workers for consistent throughput.
	streamWorkerCount = 50

	// streamQueueSize is the buffer size for the stream persistence queue.
	streamQueueSize = 4096
)

type awarenessItem struct {
	roomID    string
	clientIDs []uint64
	data      []byte
}

func NewHub(messagebus messagebus.MessageBus, serverID string, logger *zap.Logger) *Hub {
	h := &Hub{
		rooms:      make(map[string]*Room),
		Register:   make(chan *Client, 2048),
		Unregister: make(chan *Client, 2048),
		Broadcast:  make(chan *Message, 4096),
		// redis
		messagebus:   messagebus,
		serverID:     serverID,
		logger:       logger,
		fallbackMode: false,
		// P0 fix: bounded publish worker pool
		publishQueue: make(chan *Message, publishQueueSize),
		publishDone:  make(chan struct{}),
		// bounded awareness worker pool
		awarenessQueue: make(chan awarenessItem, awarenessQueueSize),
		// Stream persistence worker pool
		streamQueue: make(chan *Message, streamQueueSize),
		streamDone:  make(chan struct{}),
	}

	// Start the fixed worker pool for Redis publishing
	h.startPublishWorkers(publishWorkerCount)
	h.startAwarenessWorkers(awarenessWorkerCount)
	h.startStreamWorkers(streamWorkerCount)

	return h
}

// startPublishWorkers launches n goroutines that consume from publishQueue
// and publish messages to Redis. Workers exit when publishDone is closed.
func (h *Hub) startPublishWorkers(n int) {
	for i := 0; i < n; i++ {
		go func(workerID int) {
			for {
				select {
				case <-h.publishDone:
					h.logger.Info("Publish worker exiting", zap.Int("worker_id", workerID))
					return
				case msg, ok := <-h.publishQueue:
					if !ok {
						return
					}
					ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)

					err := h.messagebus.Publish(ctx, msg.RoomID, msg.Data)

					cancel()

					if err != nil {
						h.logger.Error("Redis Publish failed", zap.Error(err))
					}
				}
			}
		}(i)
	}
	h.logger.Info("Publish worker pool started", zap.Int("workers", n))
}

func (h *Hub) startAwarenessWorkers(n int) {
	for i := 0; i < n; i++ {
		go func(workerID int) {
			for {
				select {
				case <-h.publishDone:
					h.logger.Info("Awareness worker exiting", zap.Int("worker_id", workerID))
					return
				case item, ok := <-h.awarenessQueue:
					if !ok {
						return
					}
					if h.fallbackMode || h.messagebus == nil {
						continue
					}
					ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
					for _, clientID := range item.clientIDs {
						if err := h.messagebus.SetAwareness(ctx, item.roomID, clientID, item.data); err != nil {
							h.logger.Warn("Failed to cache awareness in Redis",
								zap.Uint64("yjs_id", clientID),
								zap.Error(err))
						}
					}
					cancel()
				}
			}
		}(i)
	}
	h.logger.Info("Awareness worker pool started", zap.Int("workers", n))
}

// startStreamWorkers launches n goroutines that consume from streamQueue
// and add messages to Redis Streams for durability and replay.
func (h *Hub) startStreamWorkers(n int) {
	for i := 0; i < n; i++ {
		go func(workerID int) {
			for {
				select {
				case <-h.streamDone:
					h.logger.Info("Stream worker exiting", zap.Int("worker_id", workerID))
					return
				case msg, ok := <-h.streamQueue:
					if !ok {
						return
					}
					h.addToStream(msg)
				}
			}
		}(i)
	}
	h.logger.Info("Stream worker pool started", zap.Int("workers", n))
}

// encodeBase64 encodes binary data to base64 string for Redis storage
func encodeBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// addToStream adds a message to Redis Streams for durability
func (h *Hub) addToStream(msg *Message) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	streamKey := "stream:" + msg.RoomID

	// Get next sequence number atomically
	seqKey := "seq:" + msg.RoomID
	seq, err := h.messagebus.Incr(ctx, seqKey)
	if err != nil {
		h.logger.Error("Failed to increment sequence",
			zap.String("room_id", msg.RoomID),
			zap.Error(err))
		return
	}

	// Encode payload as base64 (binary-safe storage)
	payload := encodeBase64(msg.Data)

	// Extract Yjs message type from first byte as numeric string
	msgType := "0"
	if len(msg.Data) > 0 {
		msgType = strconv.Itoa(int(msg.Data[0]))
	}

	// Add entry to Stream with MAXLEN trimming
	values := map[string]interface{}{
		"type":        "update",
		"server_id":   h.serverID,
		"yjs_payload": payload,
		"msg_type":    msgType,
		"seq":         seq,
		"timestamp":   time.Now().Format(time.RFC3339),
	}

	_, err = h.messagebus.XAdd(ctx, streamKey, 10000, true, values)
	if err != nil {
		h.logger.Error("Failed to add to Stream",
			zap.String("stream_key", streamKey),
			zap.Int64("seq", seq),
			zap.Error(err))
		return
	}

	// Mark this document as active so the persist worker only processes active streams
	_ = h.messagebus.ZAdd(ctx, "active-streams", float64(time.Now().Unix()), msg.RoomID)
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.registerClient(client)
		case client := <-h.Unregister:
			h.unregisterClient(client)
		case message := <-h.Broadcast:
			h.broadcastMessage(message)
		}
	}
}

func (h *Hub) registerClient(client *Client) {
	var room *Room
	var exists bool
	var needSubscribe bool

	h.mu.Lock()
	room, exists = h.rooms[client.roomID]

	// --- 1. 初始化房间 (仅针对该服务器上的第一个人) ---
	if !exists {
		room = &Room{
			ID:      client.roomID,
			clients: make(map[*Client]bool),
			cancel:  nil,
			// lastAwareness 已被我们从结构体中物理删除或不再使用
		}
		h.rooms[client.roomID] = room
		h.logger.Info("Created new local room instance", zap.String("room_id", client.roomID))
	}
	if room.cancel == nil && !h.fallbackMode && h.messagebus != nil {
		needSubscribe = true
	}
	h.mu.Unlock()

	// 开启跨服订阅（避免在 h.mu 下做网络 I/O）
	if needSubscribe {
		h.subscribeMu.Lock()
		h.mu.RLock()
		room = h.rooms[client.roomID]
		alreadySubscribed := room != nil && room.cancel != nil
		h.mu.RUnlock()
		if !alreadySubscribed {
			ctx, cancel := context.WithCancel(context.Background())
			msgChan, err := h.messagebus.Subscribe(ctx, client.roomID)
			if err != nil {
				h.logger.Error("Redis Subscribe failed", zap.Error(err))
				cancel()
			} else {
				h.mu.Lock()
				room = h.rooms[client.roomID]
				if room == nil {
					h.mu.Unlock()
					cancel()
					_ = h.messagebus.Unsubscribe(context.Background(), client.roomID)
				} else {
					room.cancel = cancel
					h.mu.Unlock()
					go h.startRoomMessageForwarding(ctx, client.roomID, msgChan)
				}
			}
		}
		h.subscribeMu.Unlock()
	}

	// --- 2. 将客户端加入本地房间列表 ---
	room.mu.Lock()
	room.clients[client] = true
	room.mu.Unlock()

	h.logger.Info("Client linked to room", zap.String("client_id", client.ID))

	// --- 3. 核心改进：立刻同步全量状态 ---
	// 无论是不是第一个人，只要有人进来，我们就去 Redis 抓取所有人的状态发给他
	// hub/hub.go 内部的 registerClient 函数

	// ... 之前的代码保持不变 ...

	if !h.fallbackMode && h.messagebus != nil {
		go func(c *Client) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			// 1. 从 Redis 抓取
			awarenessMap, err := h.messagebus.GetAllAwareness(ctx, c.roomID)
			if err != nil {
				h.logger.Error("Redis sync failed in goroutine",
					zap.String("client_id", c.ID),
					zap.Error(err))
				return
			}

			if len(awarenessMap) == 0 {
				h.logger.Debug("No awareness data found in Redis for sync", zap.String("room_id", c.roomID))
				return
			}

			h.logger.Info("Starting state delivery to joiner",
				zap.String("client_id", c.ID),
				zap.Int("items", len(awarenessMap)))

			// 2. 逐条发送，带锁保护
			sentCount := 0
			for clientID, data := range awarenessMap {
				c.sendMu.Lock()
				// 🛑 核心防御：检查通道是否已被 unregisterClient 关闭
				if c.sendClosed {
					c.sendMu.Unlock()
					h.logger.Warn("Sync aborted: client channel closed while sending",
						zap.String("client_id", c.ID),
						zap.Uint64("target_yjs_id", clientID))
					return // 直接退出协程，不发了
				}

				select {
				case c.send <- data:
					sentCount++
				default:
					// 缓冲区满了（通常是因为网络太卡），记录一条警告
					h.logger.Warn("Sync item skipped: client send buffer full",
						zap.String("client_id", c.ID),
						zap.Uint64("target_yjs_id", clientID))
				}
				c.sendMu.Unlock()
			}

			h.logger.Info("State sync completed successfully",
				zap.String("client_id", c.ID),
				zap.Int("delivered", sentCount))
		}(client)
	}
}
func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	// 注意：这里不使用 defer，因为我们需要根据逻辑流手动释放锁，避免阻塞后续的 Redis 操作

	room, exists := h.rooms[client.roomID]
	if !exists {
		h.mu.Unlock()
		return
	}

	// 1. 局部清理：从房间内移除客户端
	room.mu.Lock()
	if _, ok := room.clients[client]; ok {
		delete(room.clients, client)

		// 安全关闭客户端的发送管道
		client.sendMu.Lock()
		if !client.sendClosed {
			close(client.send)
			client.sendClosed = true
		}
		client.sendMu.Unlock()
	}
	remainingClientsCount := len(room.clients)
	room.mu.Unlock()

	h.logger.Info("Unregistered client from room",
		zap.String("client_id", client.ID),
		zap.String("room_id", client.roomID),
		zap.Int("remaining_clients", remainingClientsCount),
	)

	// 2. 分布式清理：删除 Redis 中的感知数据 (Awareness)
	if !h.fallbackMode && h.messagebus != nil {
		go func() {
			// 使用带超时的 Context 执行删除
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			client.idsMu.Lock()
			// 遍历该客户端在本机观察到的所有 Yjs ID
			for clientID := range client.observedYjsIDs {
				err := h.messagebus.DeleteAwareness(ctx, client.roomID, clientID)
				h.logger.Info("DEBUG: IDs to cleanup",
					zap.String("client_id", client.ID),
					zap.Any("ids", client.observedYjsIDs))
				if err != nil {
					h.logger.Warn("Failed to delete awareness from Redis",
						zap.Uint64("yjs_id", clientID),
						zap.Error(err),
					)
				}
			}
			client.idsMu.Unlock()
		}()
	}

	// 3. 协作清理：发送"僵尸删除"消息
	// 注意：无论本地是否有其他客户端，都要发布到 Redis，因为其他服务器可能有客户端
	client.idsMu.Lock()
	clientClocks := make(map[uint64]uint64, len(client.observedYjsIDs))
	for id, clock := range client.observedYjsIDs {
		clientClocks[id] = clock
	}
	client.idsMu.Unlock()

	if len(clientClocks) > 0 {
		// 构造 Yjs 协议格式的删除消息
		deleteMsg := MakeYjsDeleteMessage(clientClocks)

		// 本地广播：只有当本地还有其他客户端时才需要
		if remainingClientsCount > 0 {
			msg := &Message{
				RoomID: client.roomID,
				Data:   deleteMsg,
				sender: nil, // 系统发送
			}
			go func() {
				select {
				case h.Broadcast <- msg:
				case <-time.After(500 * time.Millisecond):
					h.logger.Error("Critical: Failed to broadcast cleanup message (channel blocked)")
				}
			}()
		}

		// 发布到 Redis：无论本地是否有客户端，都要通知其他服务器
		if !h.fallbackMode && h.messagebus != nil {
			go func(roomID string, data []byte) {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()
				if err := h.messagebus.Publish(ctx, roomID, data); err != nil {
					h.logger.Error("Failed to publish delete message to Redis",
						zap.String("room_id", roomID),
						zap.Error(err))
				} else {
					h.logger.Debug("Published delete message to Redis",
						zap.String("room_id", roomID),
						zap.Int("yjs_ids_count", len(clientClocks)))
				}
			}(client.roomID, deleteMsg)
		}
	}

	// 4. 房间清理：如果是本服务器最后一个人，清理本地资源
	// 注意：不要删除整个 Redis Hash，因为其他服务器可能还有客户端
	if remainingClientsCount == 0 {
		h.logger.Info("Room is empty on this server, cleaning up local resources", zap.String("room_id", client.roomID))

		// A. 停止转发协程
		if room.cancel != nil {
			room.cancel()
		}

		// B. 取消 Redis 订阅（但不删除 awareness hash，其他服务器可能还有客户端）
		if !h.fallbackMode && h.messagebus != nil {
			go func(rID string) {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				// 只取消订阅，不清除 awareness（已在步骤2中按 Yjs ID 单独删除）
				if err := h.messagebus.Unsubscribe(ctx, rID); err != nil {
					h.logger.Warn("Failed to unsubscribe", zap.Error(err))
				}
			}(client.roomID)
		}
		// C. 从内存中移除
		delete(h.rooms, client.roomID)
	}

	h.mu.Unlock() // 手动释放 Hub 锁
}

const (
	writeWait       = 10 * time.Second
	pongWait        = 60 * time.Second
	pingPeriod      = (pongWait * 9) / 10 // 54 seconds
	maxSendFailures = 5
)

func (h *Hub) broadcastMessage(message *Message) {
	h.mu.RLock()
	room, exists := h.rooms[message.RoomID]
	h.mu.RUnlock()

	if !exists {
		// 如果房间不存在，没必要继续
		return
	}

	// 1. 处理 Awareness 缓存 (Type 1)
	// if len(message.Data) > 0 && message.Data[0] == 1 {
	// 	room.mu.Lock()
	// 	room.lastAwareness = make([]byte, len(message.Data))
	// 	copy(room.lastAwareness, message.Data)
	// 	room.mu.Unlock()
	// }

	// 2. 本地广播
	h.broadcastToLocalClients(room, message.Data, message.sender)

	// 只有本地客户端发出的消息 (sender != nil) 才推送到 Redis
	// P0 fix: send to bounded worker pool instead of spawning unbounded goroutines
	if message.sender != nil && !h.fallbackMode && h.messagebus != nil {
		// 3a. Publish to Pub/Sub (real-time cross-server broadcast)
		select {
		case h.publishQueue <- message:
			// Successfully queued for async publish by worker pool
		default:
			// Queue full — drop to protect the system (same pattern as broadcastToLocalClients)
			h.logger.Warn("Publish queue full, dropping Redis publish",
				zap.String("room_id", message.RoomID))
		}

		// 3b. Add to Stream for durability (only Type 0 updates, not Type 1 awareness)
		// Type 0 = Yjs sync/update messages (document changes)
		// Type 1 = Yjs awareness messages (cursors, presence) - ephemeral, skip
		if len(message.Data) > 0 && message.Data[0] == 0 {
			select {
			case h.streamQueue <- message:
				// Successfully queued for async Stream add
			default:
				h.logger.Warn("Stream queue full, dropping durability",
					zap.String("room_id", message.RoomID))
			}
		}
	}
}

func (h *Hub) broadcastToLocalClients(room *Room, data []byte, sender *Client) {
	room.mu.RLock()
	defer room.mu.RUnlock()

	// 2. 遍历房间内所有本地连接的客户端
	for client := range room.clients {
		// 3. 排除发送者（如果是从 Redis 来的消息，sender 通常为 nil）
		if client != sender {
			select {
			case client.send <- data:
				// 发送成功：重置该客户端的失败计数
				client.failureMu.Lock()
				client.failureCount = 0
				client.failureMu.Unlock()

			default:
				client.handleSendFailure()
			}
		}
	}
}
func (h *Hub) startRoomMessageForwarding(ctx context.Context, roomID string, msgChan <-chan []byte) {
	// Increment and log reconnection count for debugging
	h.mu.RLock()
	room, exists := h.rooms[roomID]
	h.mu.RUnlock()

	if exists {
		room.mu.Lock()
		room.reconnectCount++
		reconnectCount := room.reconnectCount
		room.mu.Unlock()

		h.logger.Info("Starting message forwarding from Redis to room",
			zap.String("room_id", roomID),
			zap.String("server_id", h.serverID),
			zap.Int("reconnect_count", reconnectCount),
		)
	} else {
		h.logger.Info("Starting message forwarding from Redis to room",
			zap.String("room_id", roomID),
			zap.String("server_id", h.serverID),
		)
	}

	for {
		select {
		case <-ctx.Done():
			h.logger.Info("Stopping Redis message forwarding", zap.String("room_id", roomID))
			return

		case data, ok := <-msgChan:
			if !ok {
				h.logger.Warn("Redis message channel closed for room", zap.String("room_id", roomID))
				return
			}

			// 1. 日志记录：记录消息类型和长度
			msgType := byte(0)
			if len(data) > 0 {
				msgType = data[0]
			}
			h.logger.Debug("Received message from Redis",
				zap.String("room_id", roomID),
				zap.Int("bytes", len(data)),
				zap.Uint8("msg_type", msgType),
			)

			// 2. 获取本地房间对象
			h.mu.RLock()
			room, exists := h.rooms[roomID]
			h.mu.RUnlock()

			if !exists {
				// 这种情况常见于：Redis 消息飞过来时，本地最后一个用户刚好断开删除了房间
				h.logger.Warn("Received Redis message but room does not exist locally",
					zap.String("room_id", roomID),
				)
				continue
			}

			// 3. 广播给本地所有客户端
			// sender=nil 确保每个人（包括原本发送这条消息的人，如果他在本台机器上）都能收到
			h.broadcastToLocalClients(room, data, nil)

			// 4. 如果是感知数据 (Awareness)，更新房间缓存
			// 这样后来加入的用户能立即看到其他人的光标
			// if len(data) > 0 && data[0] == 1 {
			// 	room.mu.Lock()
			// 	room.lastAwareness = make([]byte, len(data))
			// 	copy(room.lastAwareness, data)
			// 	room.mu.Unlock()
			// }
		}
	}
}

// 辅助方法：处理发送失败逻辑，保持代码整洁 (DRY 原则)
func (c *Client) handleSendFailure() {
	c.failureMu.Lock()
	c.failureCount++
	currentFailures := c.failureCount
	c.failureMu.Unlock()

	// 直接通过 c.hub 访问日志，代码非常自然
	c.hub.logger.Warn("Failed to send message to client",
		zap.String("clientID", c.ID),
		zap.String("roomID", c.roomID),
		zap.Int("failures", currentFailures),
		zap.Int("max", maxSendFailures),
	)

	if currentFailures >= maxSendFailures {
		c.hub.logger.Error("Client exceeded max failures, disconnecting",
			zap.String("clientID", c.ID))

		// 这里的异步处理很正确，防止阻塞当前的广播循环
		go func() {
			c.unregister()
			c.Conn.Close()
		}()
	}
}
func (h *Hub) SetFallbackMode(fallback bool) {
	h.mu.Lock()
	// 1. 检查状态是否有变化（幂等性）
	if h.fallbackMode == fallback {
		h.mu.Unlock()
		return
	}

	// 2. 更新状态并记录日志
	h.fallbackMode = fallback
	if fallback {
		h.logger.Warn("Hub entering FALLBACK MODE (local-only communication)")
	} else {
		h.logger.Info("Hub restored to DISTRIBUTED MODE (cross-server sync enabled)")
	}

	// 3. 如果是从备用模式恢复 (fallback=false)，需要重新激活所有房间的 Redis 订阅
	if !fallback && h.messagebus != nil {
		h.logger.Info("Recovering Redis subscriptions for existing rooms", zap.Int("room_count", len(h.rooms)))

		for roomID, room := range h.rooms {
			// 先清理旧的订阅句柄（如果有残留）
			if room.cancel != nil {
				room.cancel()
				room.cancel = nil
			}

			// 为每个房间重新开启订阅
			ctx, cancel := context.WithCancel(context.Background())
			room.cancel = cancel

			// 尝试重新订阅
			msgChan, err := h.messagebus.Subscribe(ctx, roomID)
			if err != nil {
				h.logger.Error("Failed to re-subscribe during recovery",
					zap.String("room_id", roomID),
					zap.Error(err),
				)
				cancel()
				room.cancel = nil
				continue
			}

			// 重新启动转发协程
			go h.startRoomMessageForwarding(ctx, roomID, msgChan)

			h.logger.Debug("Successfully re-synced room with Redis", zap.String("room_id", roomID))
		}
	}
	h.mu.Unlock()
}
func (c *Client) ReadPump() {
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	defer func() {
		c.unregister()
		c.Conn.Close()
	}()

	for {
		messageType, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.hub.logger.Warn("Unexpected WebSocket close",
					zap.String("client_id", c.ID),
					zap.Error(err))
			}
			break
		}

		// 1. Sniff Yjs client IDs from awareness messages (before broadcast)
		if messageType == websocket.BinaryMessage && len(message) > 0 && message[0] == 1 {
			clockMap := SniffYjsClientIDs(message)
			if len(clockMap) > 0 {
				c.idsMu.Lock()
				for id, clock := range clockMap {
					if clock > c.observedYjsIDs[id] {
						c.hub.logger.Debug("Sniffed Yjs client ID",
							zap.String("client_id", c.ID),
							zap.Uint64("yjs_id", id),
							zap.Uint64("clock", clock))
						c.observedYjsIDs[id] = clock
					}
				}
				c.idsMu.Unlock()

				// Cache awareness in Redis for cross-server sync
				// Use a bounded worker pool to avoid blocking ReadPump on Redis I/O.
				if !c.hub.fallbackMode && c.hub.messagebus != nil {
					clientIDs := make([]uint64, 0, len(clockMap))
					for clientID := range clockMap {
						clientIDs = append(clientIDs, clientID)
					}
					select {
					case c.hub.awarenessQueue <- awarenessItem{
						roomID:    c.roomID,
						clientIDs: clientIDs,
						data:      message,
					}:
					default:
						c.hub.logger.Warn("Awareness queue full, dropping update",
							zap.String("room_id", c.roomID),
							zap.Int("clients", len(clientIDs)))
					}
				}
			}
		}

		// 2. Permission check - block write operations from view-only users
		if c.Permission == "view" {
			isSyncMessage := len(message) > 0 && message[0] == 0
			isUpdateOp := len(message) > 1 && message[1] == 2

			if isSyncMessage && isUpdateOp {
				c.hub.logger.Warn("Blocked write from view-only user",
					zap.String("client_id", c.ID))
				continue
			}
			// Allow: Awareness (type=1), SyncStep1/2 (type=0, subtype=0/1)
		}

		// 3. Broadcast to room
		if messageType == websocket.BinaryMessage {
			select {
			case c.hub.Broadcast <- &Message{
				RoomID: c.roomID,
				Data:   message,
				sender: c,
			}:
			default:
				c.hub.logger.Warn("Hub broadcast channel full, dropping message",
					zap.String("client_id", c.ID))
			}
		}
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.unregister()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteMessage(websocket.BinaryMessage, message); err != nil {
				c.hub.logger.Warn("Error writing message to client",
					zap.String("client_id", c.ID),
					zap.Error(err))
				return
			}

			// P2 fix: write coalescing — drain all queued messages in a tight loop
			for {
				select {
				case extra, ok := <-c.send:
					if !ok {
						c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
						return
					}
					c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
					if err := c.Conn.WriteMessage(websocket.BinaryMessage, extra); err != nil {
						return
					}
				default:
					break
				}
				if len(c.send) == 0 {
					break
				}
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.hub.logger.Debug("Ping failed for client",
					zap.String("client_id", c.ID),
					zap.Error(err))
				return
			}
		}
	}
}

func NewClient(id string, userID *uuid.UUID, userName string, userAvatar *string, permission string, conn *websocket.Conn, hub *Hub, roomID string) *Client {
	return &Client{
		ID:             id,
		UserID:         userID,
		UserName:       userName,
		UserAvatar:     userAvatar,
		Permission:     permission,
		Conn:           conn,
		send:           make(chan []byte, 8192),
		hub:            hub,
		roomID:         roomID,
		observedYjsIDs: make(map[uint64]uint64),
	}
}

// Enqueue sends a message to the client send buffer (non-blocking).
// Returns false if the buffer is full.
func (c *Client) Enqueue(message []byte) bool {
	select {
	case c.send <- message:
		return true
	default:
		if c.hub != nil && c.hub.logger != nil {
			c.hub.logger.Warn("Client send buffer full during replay",
				zap.String("client_id", c.ID),
				zap.String("room_id", c.roomID))
		}
		return false
	}
}
func (c *Client) unregister() {
	c.unregisterOnce.Do(func() {
		c.hub.Unregister <- c
	})
}

package messagebus

import (
	"context"
	"time"
)

// MessageBus abstracts message distribution across server instances
type MessageBus interface {
	// Publish sends a message to a specific room channel
	// data must be preserved as-is (binary safe)
	Publish(ctx context.Context, roomID string, data []byte) error

	// Subscribe listens to messages for a specific room
	// Returns a channel that receives binary messages
	Subscribe(ctx context.Context, roomID string) (<-chan []byte, error)

	// Unsubscribe stops listening to a room
	Unsubscribe(ctx context.Context, roomID string) error

	// SetAwareness caches awareness data for a client in a room
	SetAwareness(ctx context.Context, roomID string, clientID uint64, data []byte) error

	// GetAllAwareness retrieves all cached awareness for a room
	GetAllAwareness(ctx context.Context, roomID string) (map[uint64][]byte, error)

	// DeleteAwareness removes awareness cache for a client
	DeleteAwareness(ctx context.Context, roomID string, clientID uint64) error

	ClearAllAwareness(ctx context.Context, roomID string) error

	// IsHealthy returns true if message bus is operational
	IsHealthy() bool

	// Close gracefully shuts down the message bus
	Close() error

	// ========== Redis Streams Operations ==========

	// XAdd adds a new entry to a stream with optional MAXLEN trimming
	XAdd(ctx context.Context, stream string, maxLen int64, approx bool, values map[string]interface{}) (string, error)

	// XReadGroup reads messages from a stream using a consumer group
	XReadGroup(ctx context.Context, group, consumer string, streams []string, count int64, block time.Duration) ([]StreamMessage, error)

	// XAck acknowledges one or more messages from a consumer group
	XAck(ctx context.Context, stream, group string, ids ...string) (int64, error)

	// XGroupCreate creates a new consumer group for a stream
	XGroupCreate(ctx context.Context, stream, group, start string) error

	// XGroupCreateMkStream creates a consumer group and the stream if it doesn't exist
	XGroupCreateMkStream(ctx context.Context, stream, group, start string) error

	// XPending returns pending messages information for a consumer group
	XPending(ctx context.Context, stream, group string) (*PendingInfo, error)

	// XClaim claims pending messages from a consumer group
	XClaim(ctx context.Context, stream, group, consumer string, minIdleTime time.Duration, ids ...string) ([]StreamMessage, error)

	// XAutoClaim claims pending messages automatically (Redis >= 6.2)
	// Returns claimed messages and next start ID.
	XAutoClaim(ctx context.Context, stream, group, consumer string, minIdleTime time.Duration, start string, count int64) ([]StreamMessage, string, error)

	// XRange reads a range of messages from a stream
	XRange(ctx context.Context, stream, start, end string) ([]StreamMessage, error)

	// XTrimMinID trims a stream to a minimum ID (time-based retention)
	XTrimMinID(ctx context.Context, stream, minID string) (int64, error)

	// Incr increments a counter atomically (for sequence numbers)
	Incr(ctx context.Context, key string) (int64, error)

	// ========== Sorted Set (ZSET) Operations ==========

	// ZAdd adds a member with a score to a sorted set (used for active-stream tracking)
	ZAdd(ctx context.Context, key string, score float64, member string) error

	// ZRangeByScore returns members with scores between min and max
	ZRangeByScore(ctx context.Context, key string, min, max float64) ([]string, error)

	// ZRemRangeByScore removes members with scores between min and max
	ZRemRangeByScore(ctx context.Context, key string, min, max float64) (int64, error)

	// Distributed lock helpers (used by background workers)
	AcquireLock(ctx context.Context, key string, ttl time.Duration) (bool, error)
	RefreshLock(ctx context.Context, key string, ttl time.Duration) (bool, error)
	ReleaseLock(ctx context.Context, key string) error
}

// StreamMessage represents a message from a Redis Stream
type StreamMessage struct {
	ID     string
	Values map[string]interface{}
}

// PendingInfo contains information about pending messages in a consumer group
type PendingInfo struct {
	Count     int64
	Lower     string
	Upper     string
	Consumers map[string]int64
}

// LocalMessageBus is a no-op implementation for single-server mode
type LocalMessageBus struct{}

func NewLocalMessageBus() *LocalMessageBus {
	return &LocalMessageBus{}
}

func (l *LocalMessageBus) Publish(ctx context.Context, roomID string, data []byte) error {
	return nil // No-op for local mode
}

func (l *LocalMessageBus) Subscribe(ctx context.Context, roomID string) (<-chan []byte, error) {
	ch := make(chan []byte)
	close(ch) // Immediately closed channel
	return ch, nil
}

func (l *LocalMessageBus) Unsubscribe(ctx context.Context, roomID string) error {
	return nil
}

func (l *LocalMessageBus) SetAwareness(ctx context.Context, roomID string, clientID uint64, data []byte) error {
	return nil
}

func (l *LocalMessageBus) GetAllAwareness(ctx context.Context, roomID string) (map[uint64][]byte, error) {
	return nil, nil
}

func (l *LocalMessageBus) DeleteAwareness(ctx context.Context, roomID string, clientID uint64) error {
	return nil
}
func (l *LocalMessageBus) ClearAllAwareness(ctx context.Context, roomID string) error {
	return nil
}

func (l *LocalMessageBus) IsHealthy() bool {
	return true
}

func (l *LocalMessageBus) Close() error {
	return nil
}

// ========== Redis Streams Operations (No-op for local mode) ==========

func (l *LocalMessageBus) XAdd(ctx context.Context, stream string, maxLen int64, approx bool, values map[string]interface{}) (string, error) {
	return "0-0", nil
}

func (l *LocalMessageBus) XReadGroup(ctx context.Context, group, consumer string, streams []string, count int64, block time.Duration) ([]StreamMessage, error) {
	return nil, nil
}

func (l *LocalMessageBus) XAck(ctx context.Context, stream, group string, ids ...string) (int64, error) {
	return 0, nil
}

func (l *LocalMessageBus) XGroupCreate(ctx context.Context, stream, group, start string) error {
	return nil
}

func (l *LocalMessageBus) XGroupCreateMkStream(ctx context.Context, stream, group, start string) error {
	return nil
}

func (l *LocalMessageBus) XPending(ctx context.Context, stream, group string) (*PendingInfo, error) {
	return &PendingInfo{}, nil
}

func (l *LocalMessageBus) XClaim(ctx context.Context, stream, group, consumer string, minIdleTime time.Duration, ids ...string) ([]StreamMessage, error) {
	return nil, nil
}

func (l *LocalMessageBus) XAutoClaim(ctx context.Context, stream, group, consumer string, minIdleTime time.Duration, start string, count int64) ([]StreamMessage, string, error) {
	return nil, "0-0", nil
}

func (l *LocalMessageBus) XRange(ctx context.Context, stream, start, end string) ([]StreamMessage, error) {
	return nil, nil
}

func (l *LocalMessageBus) XTrimMinID(ctx context.Context, stream, minID string) (int64, error) {
	return 0, nil
}

func (l *LocalMessageBus) Incr(ctx context.Context, key string) (int64, error) {
	return 0, nil
}

func (l *LocalMessageBus) ZAdd(ctx context.Context, key string, score float64, member string) error {
	return nil
}

func (l *LocalMessageBus) ZRangeByScore(ctx context.Context, key string, min, max float64) ([]string, error) {
	return nil, nil
}

func (l *LocalMessageBus) ZRemRangeByScore(ctx context.Context, key string, min, max float64) (int64, error) {
	return 0, nil
}

func (l *LocalMessageBus) AcquireLock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	return true, nil
}

func (l *LocalMessageBus) RefreshLock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	return true, nil
}

func (l *LocalMessageBus) ReleaseLock(ctx context.Context, key string) error {
	return nil
}

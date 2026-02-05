package messagebus

import (
	"context"
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

package models

import (
	"time"

	"github.com/google/uuid"
)

// StreamCheckpoint tracks the last processed Redis Stream entry per document
type StreamCheckpoint struct {
	DocumentID   uuid.UUID `json:"document_id"`
	LastStreamID string    `json:"last_stream_id"`
	LastSeq      int64     `json:"last_seq"`
	UpdatedAt    time.Time `json:"updated_at"`
}

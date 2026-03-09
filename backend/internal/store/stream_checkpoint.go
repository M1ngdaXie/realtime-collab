package store

import (
	"context"
	"fmt"

	"github.com/M1ngdaXie/realtime-collab/internal/models"
	"github.com/google/uuid"
)

// UpsertStreamCheckpoint creates or updates the stream checkpoint for a document
func (s *PostgresStore) UpsertStreamCheckpoint(ctx context.Context, documentID uuid.UUID, streamID string, seq int64) error {
	query := `
		INSERT INTO stream_checkpoints (document_id, last_stream_id, last_seq, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (document_id)
		DO UPDATE SET last_stream_id = EXCLUDED.last_stream_id,
		             last_seq = EXCLUDED.last_seq,
		             updated_at = NOW()
	`

	if _, err := s.db.ExecContext(ctx, query, documentID, streamID, seq); err != nil {
		return fmt.Errorf("failed to upsert stream checkpoint: %w", err)
	}
	return nil
}

// GetStreamCheckpoint retrieves the stream checkpoint for a document
func (s *PostgresStore) GetStreamCheckpoint(ctx context.Context, documentID uuid.UUID) (*models.StreamCheckpoint, error) {
	query := `
		SELECT document_id, last_stream_id, last_seq, updated_at
		FROM stream_checkpoints
		WHERE document_id = $1
	`

	var checkpoint models.StreamCheckpoint
	if err := s.db.QueryRowContext(ctx, query, documentID).Scan(
		&checkpoint.DocumentID,
		&checkpoint.LastStreamID,
		&checkpoint.LastSeq,
		&checkpoint.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("failed to get stream checkpoint: %w", err)
	}
	return &checkpoint, nil
}

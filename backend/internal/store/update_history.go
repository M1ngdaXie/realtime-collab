package store

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

// UpdateHistoryEntry represents a persisted update from Redis Streams
// used for recovery and replay.
type UpdateHistoryEntry struct {
	DocumentID uuid.UUID
	StreamID   string
	Seq        int64
	Payload    []byte
	MsgType    string
	ServerID   string
	CreatedAt  time.Time
}

// InsertUpdateHistoryBatch inserts update history entries in a single batch.
// Uses ON CONFLICT DO NOTHING to make inserts idempotent.
func (s *PostgresStore) InsertUpdateHistoryBatch(ctx context.Context, entries []UpdateHistoryEntry) error {
	if len(entries) == 0 {
		return nil
	}

	var sb strings.Builder
	sb.WriteString("INSERT INTO document_update_history (document_id, stream_id, seq, payload, msg_type, server_id, created_at) VALUES ")

	args := make([]interface{}, 0, len(entries)*7)
	for i, e := range entries {
		if i > 0 {
			sb.WriteString(",")
		}
		base := i*7 + 1
		sb.WriteString(fmt.Sprintf("($%d,$%d,$%d,$%d,$%d,$%d,$%d)", base, base+1, base+2, base+3, base+4, base+5, base+6))
		msgType := sanitizeTextForDB(e.MsgType)
		serverID := sanitizeTextForDB(e.ServerID)
		args = append(args, e.DocumentID, e.StreamID, e.Seq, e.Payload, nullIfEmpty(msgType), nullIfEmpty(serverID), e.CreatedAt)
	}
	// Idempotent insert
	sb.WriteString(" ON CONFLICT (document_id, stream_id) DO NOTHING")

	if _, err := s.db.ExecContext(ctx, sb.String(), args...); err != nil {
		return fmt.Errorf("failed to insert update history batch: %w", err)
	}
	return nil
}

// ListUpdateHistoryAfterSeq returns updates with seq greater than afterSeq, ordered by seq.
func (s *PostgresStore) ListUpdateHistoryAfterSeq(ctx context.Context, documentID uuid.UUID, afterSeq int64, limit int) ([]UpdateHistoryEntry, error) {
	if limit <= 0 {
		limit = 1000
	}
	query := `
		SELECT document_id, stream_id, seq, payload, COALESCE(msg_type, ''), COALESCE(server_id, ''), created_at
		FROM document_update_history
		WHERE document_id = $1 AND seq > $2
		ORDER BY seq ASC
		LIMIT $3
	`

	rows, err := s.db.QueryContext(ctx, query, documentID, afterSeq, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list update history: %w", err)
	}
	defer rows.Close()

	var results []UpdateHistoryEntry
	for rows.Next() {
		var e UpdateHistoryEntry
		if err := rows.Scan(&e.DocumentID, &e.StreamID, &e.Seq, &e.Payload, &e.MsgType, &e.ServerID, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan update history: %w", err)
		}
		results = append(results, e)
	}
	return results, nil
}

// DeleteUpdateHistoryUpToSeq deletes updates with seq <= maxSeq for a document.
func (s *PostgresStore) DeleteUpdateHistoryUpToSeq(ctx context.Context, documentID uuid.UUID, maxSeq int64) error {
	query := `
		DELETE FROM document_update_history
		WHERE document_id = $1 AND seq <= $2
	`
	if _, err := s.db.ExecContext(ctx, query, documentID, maxSeq); err != nil {
		return fmt.Errorf("failed to delete update history: %w", err)
	}
	return nil
}

func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func sanitizeTextForDB(s string) string {
	if s == "" {
		return ""
	}
	if strings.IndexByte(s, 0) >= 0 {
		return ""
	}
	if !utf8.ValidString(s) {
		return ""
	}
	return s
}

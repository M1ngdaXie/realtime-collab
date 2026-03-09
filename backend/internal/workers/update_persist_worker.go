package workers

import (
	"context"
	"encoding/base64"
	"fmt"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/M1ngdaXie/realtime-collab/internal/messagebus"
	"github.com/M1ngdaXie/realtime-collab/internal/store"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	updatePersistGroupName = "update-persist-worker"
	updatePersistLockKey   = "lock:update-persist-worker"
	updatePersistLockTTL   = 30 * time.Second
	updatePersistTick      = 2 * time.Second
	updateReadCount        = 200
	updateReadBlock        = -1 // negative → go-redis omits BLOCK clause → non-blocking
	updateBatchSize        = 500
	updateSafeSeqLag       = int64(1000)
	updateAutoClaimIdle    = 30 * time.Second
	updateHeartbeatEvery   = 30 * time.Second
)

// StartUpdatePersistWorker persists Redis Stream updates into Postgres for recovery.
func StartUpdatePersistWorker(ctx context.Context, msgBus messagebus.MessageBus, dbStore *store.PostgresStore, logger *zap.Logger, serverID string) {
	if msgBus == nil || dbStore == nil {
		return
	}

	for {
		func() {
			defer func() {
				if r := recover(); r != nil {
					logWorker(logger, "Update persist worker panic",
						zap.Any("panic", r),
						zap.ByteString("stack", debug.Stack()))
				}
			}()

			select {
			case <-ctx.Done():
				return
			default:
			}

			acquired, err := msgBus.AcquireLock(ctx, updatePersistLockKey, updatePersistLockTTL)
			if err != nil {
				logWorker(logger, "Failed to acquire update persist worker lock", zap.Error(err))
				time.Sleep(updatePersistTick)
				return
			}
			if !acquired {
				time.Sleep(updatePersistTick)
				return
			}

			logWorker(logger, "Update persist worker lock acquired", zap.String("server_id", serverID))
			runUpdatePersistWorker(ctx, msgBus, dbStore, logger, serverID)
		}()

		select {
		case <-ctx.Done():
			return
		default:
		}
		// If the worker exited (including panic), pause briefly before retry.
		time.Sleep(updatePersistTick)
	}
}

func runUpdatePersistWorker(ctx context.Context, msgBus messagebus.MessageBus, dbStore *store.PostgresStore, logger *zap.Logger, serverID string) {
	ticker := time.NewTicker(updatePersistTick)
	defer ticker.Stop()

	refreshTicker := time.NewTicker(updatePersistLockTTL / 2)
	defer refreshTicker.Stop()

	heartbeatTicker := time.NewTicker(updateHeartbeatEvery)
	defer heartbeatTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			_ = msgBus.ReleaseLock(ctx, updatePersistLockKey)
			return
		case <-refreshTicker.C:
			ok, err := msgBus.RefreshLock(ctx, updatePersistLockKey, updatePersistLockTTL)
			if err != nil || !ok {
				logWorker(logger, "Update persist worker lock lost", zap.Error(err))
				_ = msgBus.ReleaseLock(ctx, updatePersistLockKey)
				return
			}
		case <-heartbeatTicker.C:
			logWorker(logger, "Update persist worker heartbeat", zap.String("server_id", serverID))
		case <-ticker.C:
			if err := processUpdatePersistence(ctx, msgBus, dbStore, logger, serverID); err != nil {
				logWorker(logger, "Update persist worker tick failed", zap.Error(err))
			}
		}
	}
}

func processUpdatePersistence(ctx context.Context, msgBus messagebus.MessageBus, dbStore *store.PostgresStore, logger *zap.Logger, serverID string) error {
	// Only process documents with recent stream activity (active in the last 60 seconds)
	cutoff := float64(time.Now().Add(-60 * time.Second).Unix())
	activeDocIDs, err := msgBus.ZRangeByScore(ctx, "active-streams", cutoff, float64(time.Now().Unix()))
	if err != nil {
		return fmt.Errorf("failed to get active streams: %w", err)
	}

	// Prune stale entries older than 5 minutes (best-effort cleanup)
	stale := float64(time.Now().Add(-5 * time.Minute).Unix())
	if _, err := msgBus.ZRemRangeByScore(ctx, "active-streams", 0, stale); err != nil {
		logWorker(logger, "Failed to prune stale active-streams entries", zap.Error(err))
	}

	for _, docIDStr := range activeDocIDs {
		docID, err := uuid.Parse(docIDStr)
		if err != nil {
			logWorker(logger, "Invalid document ID in active-streams", zap.String("doc_id", docIDStr))
			continue
		}

		streamKey := "stream:" + docIDStr
		if err := ensureConsumerGroup(ctx, msgBus, streamKey, updatePersistGroupName); err != nil {
			logWorker(logger, "Failed to ensure update persist consumer group", zap.String("stream", streamKey), zap.Error(err))
			continue
		}

		var ackIDs []string
		docEntries := make([]store.UpdateHistoryEntry, 0, updateBatchSize)

		// First, try to claim idle pending messages (e.g., from previous crashes)
		claimed, _, err := msgBus.XAutoClaim(ctx, streamKey, updatePersistGroupName, serverID, updateAutoClaimIdle, "0-0", updateReadCount)
		if err != nil {
			logWorker(logger, "XAutoClaim failed", zap.String("stream", streamKey), zap.Error(err))
		} else if len(claimed) > 0 {
			collectStreamMessages(ctx, msgBus, dbStore, logger, docID, streamKey, claimed, &docEntries, &ackIDs)
		}

		messages, err := msgBus.XReadGroup(ctx, updatePersistGroupName, serverID, []string{streamKey, ">"}, updateReadCount, updateReadBlock)
		if err != nil {
			logWorker(logger, "XReadGroup failed", zap.String("stream", streamKey), zap.Error(err))
			continue
		}
		if len(messages) > 0 {
			collectStreamMessages(ctx, msgBus, dbStore, logger, docID, streamKey, messages, &docEntries, &ackIDs)
		}

		if len(docEntries) > 0 {
			if err := dbStore.InsertUpdateHistoryBatch(ctx, docEntries); err != nil {
				logWorker(logger, "Failed to insert update history batch", zap.Error(err))
				// Skip ACK to retry on next tick
				continue
			}
		}

		if len(ackIDs) > 0 {
			if _, err := msgBus.XAck(ctx, streamKey, updatePersistGroupName, ackIDs...); err != nil {
				logWorker(logger, "XAck failed", zap.String("stream", streamKey), zap.Error(err))
			}
		}
	}

	return nil
}

func collectStreamMessages(ctx context.Context, msgBus messagebus.MessageBus, dbStore *store.PostgresStore, logger *zap.Logger, documentID uuid.UUID, streamKey string, messages []messagebus.StreamMessage, docEntries *[]store.UpdateHistoryEntry, ackIDs *[]string) {
	for _, msg := range messages {
		msgType := getString(msg.Values["type"])
		switch msgType {
		case "update":
			payloadB64 := getString(msg.Values["yjs_payload"])
			payload, err := base64.StdEncoding.DecodeString(payloadB64)
			if err != nil {
				logWorker(logger, "Failed to decode update payload",
					zap.String("stream", streamKey),
					zap.String("stream_id", msg.ID),
					zap.Error(err))
				continue
			}
			seq := parseInt64(msg.Values["seq"])
			msgType := normalizeMsgType(msg.Values["msg_type"])
			serverID := sanitizeText(getString(msg.Values["server_id"]))
			entry := store.UpdateHistoryEntry{
				DocumentID: documentID,
				StreamID:   msg.ID,
				Seq:        seq,
				Payload:    payload,
				MsgType:    msgType,
				ServerID:   serverID,
				CreatedAt:  time.Now().UTC(),
			}
			*docEntries = append(*docEntries, entry)
		case "snapshot":
			seq := parseInt64(msg.Values["seq"])
			if seq > 0 {
				if err := dbStore.UpsertStreamCheckpoint(ctx, documentID, msg.ID, seq); err != nil {
					logWorker(logger, "Failed to upsert stream checkpoint from snapshot marker",
						zap.String("document_id", documentID.String()),
						zap.Error(err))
				}
				// Retention: prune DB history based on checkpoint (best-effort)
				maxSeq := seq - updateSafeSeqLag
				if maxSeq > 0 {
					if err := dbStore.DeleteUpdateHistoryUpToSeq(ctx, documentID, maxSeq); err != nil {
						logWorker(logger, "Failed to prune update history",
							zap.String("document_id", documentID.String()),
							zap.Error(err))
					}
				}
				// Trim Redis stream to avoid unbounded growth (best-effort)
				if _, err := msgBus.XTrimMinID(ctx, streamKey, msg.ID); err != nil {
					logWorker(logger, "Failed to trim Redis stream",
						zap.String("stream", streamKey),
						zap.Error(err))
				}
			}
		}
		*ackIDs = append(*ackIDs, msg.ID)
	}
}

func ensureConsumerGroup(ctx context.Context, msgBus messagebus.MessageBus, streamKey, group string) error {
	if err := msgBus.XGroupCreateMkStream(ctx, streamKey, group, "0-0"); err != nil {
		if !isBusyGroup(err) {
			return err
		}
	}
	return nil
}

func isBusyGroup(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "BUSYGROUP")
}

func getString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		return fmt.Sprint(v)
	}
}

func parseInt64(value interface{}) int64 {
	switch v := value.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case uint64:
		return int64(v)
	case string:
		if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
			return parsed
		}
	case []byte:
		if parsed, err := strconv.ParseInt(string(v), 10, 64); err == nil {
			return parsed
		}
	}
	return 0
}

func sanitizeText(s string) string {
	if s == "" {
		return s
	}
	if strings.IndexByte(s, 0) >= 0 {
		return ""
	}
	if !utf8.ValidString(s) {
		return ""
	}
	return s
}

func normalizeMsgType(value interface{}) string {
	switch v := value.(type) {
	case string:
		if v == "" {
			return ""
		}
		if len(v) == 1 {
			return strconv.Itoa(int(v[0]))
		}
		return sanitizeText(v)
	case []byte:
		if len(v) == 0 {
			return ""
		}
		if len(v) == 1 {
			return strconv.Itoa(int(v[0]))
		}
		return sanitizeText(string(v))
	default:
		return sanitizeText(fmt.Sprint(v))
	}
}

func logWorker(logger *zap.Logger, msg string, fields ...zap.Field) {
	if logger == nil {
		return
	}
	logger.Info(msg, fields...)
}

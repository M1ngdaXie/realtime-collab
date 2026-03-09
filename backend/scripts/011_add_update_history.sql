-- Migration: Add update history table for Redis Stream WAL
-- This table stores per-update payloads for recovery and replay

CREATE TABLE IF NOT EXISTS document_update_history (
    id BIGSERIAL PRIMARY KEY,
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    stream_id TEXT NOT NULL,
    seq BIGINT NOT NULL,
    payload BYTEA NOT NULL,
    msg_type TEXT,
    server_id TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uniq_update_history_document_stream_id
    ON document_update_history(document_id, stream_id);

CREATE UNIQUE INDEX IF NOT EXISTS uniq_update_history_document_seq
    ON document_update_history(document_id, seq);

CREATE INDEX IF NOT EXISTS idx_update_history_document_seq
    ON document_update_history(document_id, seq);

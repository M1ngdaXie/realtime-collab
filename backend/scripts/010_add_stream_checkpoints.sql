-- Migration: Add stream checkpoints table for Redis Streams durability
-- This table tracks last processed stream position per document

CREATE TABLE IF NOT EXISTS stream_checkpoints (
    document_id UUID PRIMARY KEY REFERENCES documents(id) ON DELETE CASCADE,
    last_stream_id TEXT NOT NULL,
    last_seq BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_stream_checkpoints_updated_at
    ON stream_checkpoints(updated_at DESC);

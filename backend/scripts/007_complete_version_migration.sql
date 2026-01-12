-- Migration: Add document version history support
-- This migration creates the version history table, adds tracking columns,
-- and provides a helper function for version numbering

-- Create document versions table for storing version snapshots
CREATE TABLE IF NOT EXISTS document_versions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    yjs_snapshot BYTEA NOT NULL,
    text_preview TEXT,
    version_number INTEGER NOT NULL,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    version_label TEXT,
    is_auto_generated BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),

    CONSTRAINT unique_document_version UNIQUE(document_id, version_number)
);

CREATE INDEX idx_document_versions_document_id ON document_versions(document_id, created_at DESC);
CREATE INDEX idx_document_versions_created_by ON document_versions(created_by);

-- Add version tracking columns to documents table
ALTER TABLE documents ADD COLUMN IF NOT EXISTS version_count INTEGER DEFAULT 0;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS last_snapshot_at TIMESTAMPTZ;

-- Function to get the next version number for a document
-- This ensures version numbers are sequential and unique per document
CREATE OR REPLACE FUNCTION get_next_version_number(p_document_id UUID)
RETURNS INTEGER AS $$
DECLARE
    next_version INTEGER;
BEGIN
    SELECT COALESCE(MAX(version_number), 0) + 1
    INTO next_version
    FROM document_versions
    WHERE document_id = p_document_id;

    RETURN next_version;
END;
$$ LANGUAGE plpgsql;

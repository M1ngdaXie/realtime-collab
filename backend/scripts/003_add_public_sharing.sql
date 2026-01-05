-- Migration: Add public sharing support via share tokens
-- Dependencies: Run after 002_add_document_shares.sql
-- Purpose: Add share_token and is_public columns used by share link feature

-- Add columns for public sharing
ALTER TABLE documents ADD COLUMN IF NOT EXISTS share_token VARCHAR(255);
ALTER TABLE documents ADD COLUMN IF NOT EXISTS is_public BOOLEAN DEFAULT false NOT NULL;

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_documents_share_token ON documents(share_token) WHERE share_token IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_documents_is_public ON documents(is_public) WHERE is_public = true;

-- Constraint: public documents must have a token
-- This ensures data integrity - a document can't be public without a share token
ALTER TABLE documents ADD CONSTRAINT check_public_has_token
    CHECK (is_public = false OR (is_public = true AND share_token IS NOT NULL));

-- Documentation
COMMENT ON COLUMN documents.share_token IS 'Public share token for link-based access (base64-encoded random string, 32 bytes)';
COMMENT ON COLUMN documents.is_public IS 'Whether document is publicly accessible via share link';

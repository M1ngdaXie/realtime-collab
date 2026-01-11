-- Migration: Add permission column for public share links
-- Dependencies: Run after 003_add_public_sharing.sql
-- Purpose: Store permission level (view/edit) for public share links

-- Add permission column to documents table
ALTER TABLE documents ADD COLUMN IF NOT EXISTS share_permission VARCHAR(20) DEFAULT 'edit' CHECK (share_permission IN ('view', 'edit'));

-- Create index for performance
CREATE INDEX IF NOT EXISTS idx_documents_share_permission ON documents(share_permission) WHERE is_public = true;

-- Documentation
COMMENT ON COLUMN documents.share_permission IS 'Permission level for public share link: view (read-only) or edit (read-write). Defaults to edit for backward compatibility.';

-- Migration: Add document sharing with permissions
-- Run against existing database

CREATE TABLE IF NOT EXISTS document_shares (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    permission VARCHAR(20) NOT NULL CHECK (permission IN ('view', 'edit')),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    UNIQUE(document_id, user_id)
);

CREATE INDEX idx_shares_document_id ON document_shares(document_id);
CREATE INDEX idx_shares_user_id ON document_shares(user_id);
CREATE INDEX idx_shares_permission ON document_shares(document_id, permission);

COMMENT ON TABLE document_shares IS 'Stores per-user document access permissions';
COMMENT ON COLUMN document_shares.permission IS 'Access level: view (read-only) or edit (read-write)';

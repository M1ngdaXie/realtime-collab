-- Migration: Create required PostgreSQL extensions
-- Extensions must be created before other migrations can use them

-- uuid-ossp: Provides functions for generating UUIDs (uuid_generate_v4())
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- pgcrypto: Provides cryptographic functions (used for token hashing)
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
-- Initialize database schema for realtime collaboration
-- This is the base schema that creates core tables for documents and updates

CREATE TABLE IF NOT EXISTS documents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL CHECK (type IN ('editor', 'kanban')),
    yjs_state BYTEA,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_documents_type ON documents(type);
CREATE INDEX idx_documents_created_at ON documents(created_at DESC);

-- Table for storing incremental updates (for history tracking)
CREATE TABLE IF NOT EXISTS document_updates (
    id SERIAL PRIMARY KEY,
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    update BYTEA NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_updates_document_id ON document_updates(document_id);
CREATE INDEX idx_updates_created_at ON document_updates(created_at DESC);
-- Migration: Add users and sessions tables for authentication
-- Run this before 002_add_document_shares.sql

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    avatar_url TEXT,
    provider VARCHAR(50) NOT NULL CHECK (provider IN ('google', 'github')),
    provider_user_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    last_login_at TIMESTAMPTZ,
    UNIQUE(provider, provider_user_id)
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_provider ON users(provider, provider_user_id);

COMMENT ON TABLE users IS 'Stores user accounts from OAuth providers';
COMMENT ON COLUMN users.provider IS 'OAuth provider: google or github';
COMMENT ON COLUMN users.provider_user_id IS 'User ID from OAuth provider';

-- Sessions table
CREATE TABLE IF NOT EXISTS sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    user_agent TEXT,
    ip_address VARCHAR(45),
    UNIQUE(token_hash)
);

CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_token_hash ON sessions(token_hash);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);

COMMENT ON TABLE sessions IS 'Stores active JWT sessions for revocation support';
COMMENT ON COLUMN sessions.token_hash IS 'SHA-256 hash of JWT token';
COMMENT ON COLUMN sessions.user_agent IS 'User agent string for device tracking';

-- Add owner_id to documents table if it doesn't exist
ALTER TABLE documents ADD COLUMN IF NOT EXISTS owner_id UUID REFERENCES users(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_documents_owner_id ON documents(owner_id);

COMMENT ON COLUMN documents.owner_id IS 'User who created the document';
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
-- Migration: Add permission column for public share links
-- Dependencies: Run after 003_add_public_sharing.sql
-- Purpose: Store permission level (view/edit) for public share links

-- Add permission column to documents table
ALTER TABLE documents ADD COLUMN IF NOT EXISTS share_permission VARCHAR(20) DEFAULT 'edit' CHECK (share_permission IN ('view', 'edit'));

-- Create index for performance
CREATE INDEX IF NOT EXISTS idx_documents_share_permission ON documents(share_permission) WHERE is_public = true;

-- Documentation
COMMENT ON COLUMN documents.share_permission IS 'Permission level for public share link: view (read-only) or edit (read-write). Defaults to edit for backward compatibility.';
-- Migration: Add OAuth token storage
-- This table stores OAuth2 access tokens and refresh tokens from external providers
-- Used for refreshing user sessions without re-authentication

CREATE TABLE IF NOT EXISTS oauth_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL,
    access_token TEXT NOT NULL,
    refresh_token TEXT,
    token_type VARCHAR(50) DEFAULT 'Bearer',
    expires_at TIMESTAMPTZ NOT NULL,
    scope TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    CONSTRAINT oauth_tokens_user_id_provider_key UNIQUE (user_id, provider)
);

CREATE INDEX idx_oauth_tokens_user_id ON oauth_tokens(user_id);
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
-- Migration: Enable Row Level Security (RLS) on all tables
-- This enables RLS but uses permissive policies to allow all operations
-- Authorization is still handled by the Go backend middleware

-- Enable RLS on all tables
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE oauth_tokens ENABLE ROW LEVEL SECURITY;
ALTER TABLE documents ENABLE ROW LEVEL SECURITY;
ALTER TABLE document_updates ENABLE ROW LEVEL SECURITY;
ALTER TABLE document_shares ENABLE ROW LEVEL SECURITY;
ALTER TABLE document_versions ENABLE ROW LEVEL SECURITY;

-- Create permissive policies that allow all operations
-- This maintains current behavior where backend handles authorization

-- Users table
CREATE POLICY "Allow all operations on users" ON users FOR ALL USING (true);

-- Sessions table
CREATE POLICY "Allow all operations on sessions" ON sessions FOR ALL USING (true);

-- OAuth tokens table
CREATE POLICY "Allow all operations on oauth_tokens" ON oauth_tokens FOR ALL USING (true);

-- Documents table
CREATE POLICY "Allow all operations on documents" ON documents FOR ALL USING (true);

-- Document updates table
CREATE POLICY "Allow all operations on document_updates" ON document_updates FOR ALL USING (true);

-- Document shares table
CREATE POLICY "Allow all operations on document_shares" ON document_shares FOR ALL USING (true);

-- Document versions table
CREATE POLICY "Allow all operations on document_versions" ON document_versions FOR ALL USING (true);
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

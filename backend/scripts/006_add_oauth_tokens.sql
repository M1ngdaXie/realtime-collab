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

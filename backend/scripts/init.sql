-- Initialize database schema for realtime collaboration

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

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

-- Optional: Table for storing incremental updates (for history)
CREATE TABLE IF NOT EXISTS document_updates (
    id SERIAL PRIMARY KEY,
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    update BYTEA NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_updates_document_id ON document_updates(document_id);
CREATE INDEX idx_updates_created_at ON document_updates(created_at DESC);

-- Insert some sample documents for testing
INSERT INTO documents (id, name, type) VALUES
    ('00000000-0000-0000-0000-000000000001', 'Welcome Document', 'editor'),
    ('00000000-0000-0000-0000-000000000002', 'Project Kanban', 'kanban')
ON CONFLICT DO NOTHING;

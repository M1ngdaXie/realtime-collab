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

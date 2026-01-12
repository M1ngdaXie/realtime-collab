-- Migration: Revoke PostgREST access to public schema
-- This prevents Supabase's auto-generated REST API from exposing tables
-- Use this if you ONLY connect via your Go backend, not via Supabase client libraries

-- Revoke access from anon and authenticated roles (used by PostgREST)
REVOKE ALL ON ALL TABLES IN SCHEMA public FROM anon, authenticated;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA public FROM anon, authenticated;
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA public FROM anon, authenticated;

-- Grant access only to postgres role (your backend connection)
GRANT ALL ON ALL TABLES IN SCHEMA public TO postgres;
GRANT ALL ON ALL SEQUENCES IN SCHEMA public TO postgres;
GRANT ALL ON ALL FUNCTIONS IN SCHEMA public TO postgres;

-- Note: Run this AFTER all other migrations
-- If you need PostgREST access later, you can re-grant permissions selectively

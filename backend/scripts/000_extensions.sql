-- Migration: Create required PostgreSQL extensions
-- Extensions must be created before other migrations can use them

-- uuid-ossp: Provides functions for generating UUIDs (uuid_generate_v4())
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- pgcrypto: Provides cryptographic functions (used for token hashing)
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

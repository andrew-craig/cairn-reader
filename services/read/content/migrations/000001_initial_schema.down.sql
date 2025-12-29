-- Content Service - Rollback Initial Schema

-- Drop triggers
DROP TRIGGER IF EXISTS trg_clear_orphaned ON user_contents;
DROP TRIGGER IF EXISTS trg_mark_orphaned ON user_contents;

-- Drop trigger functions
DROP FUNCTION IF EXISTS clear_orphaned_status();
DROP FUNCTION IF EXISTS mark_content_orphaned();

-- Drop tables (in reverse order of creation due to foreign key constraints)
DROP TABLE IF EXISTS user_contents;
DROP TABLE IF EXISTS contents;

-- Drop extension (optional - comment out if other services use it)
-- DROP EXTENSION IF EXISTS "pgcrypto";

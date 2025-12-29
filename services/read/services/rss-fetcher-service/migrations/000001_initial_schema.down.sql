-- RSS Fetcher Service Initial Schema Rollback
-- Drops all tables, triggers, and functions created in the up migration

-- Drop triggers first
DROP TRIGGER IF EXISTS trg_check_feed_limit ON feed_subscriptions;

-- Drop trigger functions
DROP FUNCTION IF EXISTS check_feed_limit();

-- Drop tables (cascade to remove foreign key constraints)
DROP TABLE IF EXISTS content_outbox CASCADE;
DROP TABLE IF EXISTS feed_items CASCADE;
DROP TABLE IF EXISTS feed_subscriptions CASCADE;
DROP TABLE IF EXISTS feeds CASCADE;

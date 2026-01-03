-- Drop indexes
DROP INDEX IF EXISTS idx_fetch_history_created_at;
DROP INDEX IF EXISTS idx_fetch_history_feed_id;
DROP INDEX IF EXISTS idx_feeds_enabled;
DROP INDEX IF EXISTS idx_feeds_last_fetched;

-- Drop tables
DROP TABLE IF EXISTS fetch_history;
DROP TABLE IF EXISTS feeds;

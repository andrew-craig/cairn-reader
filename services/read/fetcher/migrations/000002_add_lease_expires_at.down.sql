DROP INDEX IF EXISTS idx_feed_items_processing_lease;

ALTER TABLE feeds DROP COLUMN IF EXISTS lease_expires_at;
ALTER TABLE content_outbox DROP COLUMN IF EXISTS lease_expires_at;
ALTER TABLE feed_items DROP COLUMN IF EXISTS lease_expires_at;

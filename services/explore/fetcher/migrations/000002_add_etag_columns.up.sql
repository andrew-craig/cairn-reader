-- Add ETag and Last-Modified columns to support conditional GET requests.
-- These values are stored from HTTP responses and sent on the next fetch to
-- avoid re-downloading feeds that have not changed (RFC 7232).
ALTER TABLE feeds
    ADD COLUMN IF NOT EXISTS etag TEXT,
    ADD COLUMN IF NOT EXISTS last_modified TEXT;

-- Article IDs changed from SHA256(link)[:16] to SHA256(cleaned_content).
-- Reset last_fetched_at so all feeds are re-processed with the new ID scheme,
-- ensuring the recommender receives articles with consistent content-hash IDs.
UPDATE feeds SET last_fetched_at = NULL;

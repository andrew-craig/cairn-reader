-- Reverse the unique dedup indexes. The duplicate-row backfill performed by
-- the up migration is not reversible (the removed duplicate rows and their
-- user_contents links are gone), matching this repo's existing convention
-- for lossy backfills (see 000002/000003).

DROP INDEX idx_contents_nonrss_dedup;

DROP INDEX idx_contents_rss_dedup;
CREATE INDEX idx_contents_rss_dedup ON contents(content_hash, source_feed_id)
    WHERE source_type = 'rss';

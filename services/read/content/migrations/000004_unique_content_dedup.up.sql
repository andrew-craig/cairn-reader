-- idx_contents_rss_dedup was a plain (non-unique) index, so the
-- check-then-insert dedup in the service layer had no DB backstop: two
-- concurrent deliveries of the same RSS item both pass the "not found" check
-- and both insert, producing duplicate content rows. Email/manual content had
-- no dedup key at all (source_feed_id is always NULL for those), so an
-- outbox retry after a partial delivery failure created a fresh duplicate row
-- every time. This migration makes both paths UNIQUE at the DB level.

-- Backfill: collapse pre-existing RSS duplicates sharing (content_hash,
-- source_feed_id) before the unique index is added, or the CREATE UNIQUE
-- INDEX below will fail against live data. Keep the earliest-created row per
-- group as the survivor.
CREATE TEMP TABLE content_dedup_map AS
SELECT id AS loser_id, first_value(id) OVER w AS survivor_id
FROM contents
WHERE source_type = 'rss'
WINDOW w AS (PARTITION BY content_hash, source_feed_id ORDER BY created_at ASC, id ASC)
UNION ALL
SELECT id AS loser_id, first_value(id) OVER w AS survivor_id
FROM contents
WHERE source_type != 'rss'
WINDOW w AS (PARTITION BY content_hash, original_url ORDER BY created_at ASC, id ASC);

DELETE FROM content_dedup_map WHERE loser_id = survivor_id;

-- Re-point user_contents from losing rows to the survivor so users keep
-- their saved item instead of losing it when the loser is deleted below.
-- A user can own more than one loser mapping to the same survivor (e.g.
-- two of the concurrent duplicates this migration is cleaning up), so this
-- is an INSERT ... ON CONFLICT DO NOTHING of the survivor link followed by
-- an unconditional DELETE of every loser link, rather than an UPDATE: an
-- UPDATE's NOT EXISTS guard is evaluated once against the pre-statement
-- snapshot, so two loser rows for the same user in the same statement would
-- both pass the guard and then collide with each other on
-- unique_user_content, aborting the migration.
INSERT INTO user_contents (user_id, content_id, status, scroll_position, is_favorite, added_at, updated_at)
SELECT uc.user_id, m.survivor_id, uc.status, uc.scroll_position, uc.is_favorite, uc.added_at, uc.updated_at
FROM user_contents uc
JOIN content_dedup_map m ON uc.content_id = m.loser_id
ON CONFLICT (user_id, content_id) DO NOTHING;

DELETE FROM user_contents
WHERE content_id IN (SELECT loser_id FROM content_dedup_map);

DELETE FROM contents WHERE id IN (SELECT loser_id FROM content_dedup_map);

DROP TABLE content_dedup_map;

-- Replace the plain RSS dedup index with a UNIQUE one.
DROP INDEX idx_contents_rss_dedup;
CREATE UNIQUE INDEX idx_contents_rss_dedup ON contents(content_hash, source_feed_id)
    WHERE source_type = 'rss';

-- New dedup key for email/manual content, which has no feed to key off —
-- (content_hash, original_url) is stable across outbox retries (the URL is
-- fixed per source item: the RSS item link, or email://<raw_email_id>).
CREATE UNIQUE INDEX idx_contents_nonrss_dedup ON contents(content_hash, original_url)
    WHERE source_type != 'rss';

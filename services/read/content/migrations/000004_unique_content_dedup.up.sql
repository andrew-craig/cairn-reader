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
-- Skip a reassignment that would collide with a row the user already has
-- for the survivor (unique_user_content) — that duplicate link is simply
-- dropped along with the loser via ON DELETE CASCADE.
UPDATE user_contents uc
SET content_id = m.survivor_id
FROM content_dedup_map m
WHERE uc.content_id = m.loser_id
  AND NOT EXISTS (
      SELECT 1 FROM user_contents uc2
      WHERE uc2.user_id = uc.user_id AND uc2.content_id = m.survivor_id
  );

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

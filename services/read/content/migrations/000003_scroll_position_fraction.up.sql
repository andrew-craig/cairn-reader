-- Reading progress is persisted by the mobile/web readers as a fraction of the
-- article scrolled (offsetY / contentHeight) in the range [0, 1] — see commit
-- c67a17d ("Store reading progress as fraction instead of absolute pixels").
-- The column was still INTEGER (a legacy absolute character/pixel offset), so
-- fractional PATCH values like 0.5 were rejected by the API and progress never
-- persisted. Migrate to NUMERIC(5,4) so the fraction can be stored.

-- Clamp legacy absolute offsets (any value > 1) back to the top before the type
-- change: they are meaningless as fractions, would overflow NUMERIC(5,4), and
-- clients already ignore values > 1 on restore.
UPDATE user_contents SET scroll_position = 0 WHERE scroll_position > 1;

ALTER TABLE user_contents DROP CONSTRAINT chk_scroll_position;

ALTER TABLE user_contents
    ALTER COLUMN scroll_position TYPE NUMERIC(5,4) USING scroll_position::NUMERIC(5,4),
    ALTER COLUMN scroll_position SET DEFAULT 0.0;

ALTER TABLE user_contents
    ADD CONSTRAINT chk_scroll_position CHECK (scroll_position >= 0 AND scroll_position <= 1);

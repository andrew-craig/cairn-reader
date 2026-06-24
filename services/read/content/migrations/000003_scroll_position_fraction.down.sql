-- Revert scroll_position from a [0, 1] fraction back to an INTEGER offset.
-- Fractional values truncate toward zero; this is a lossy reversal.
ALTER TABLE user_contents DROP CONSTRAINT chk_scroll_position;

ALTER TABLE user_contents
    ALTER COLUMN scroll_position TYPE INTEGER USING FLOOR(scroll_position)::INTEGER,
    ALTER COLUMN scroll_position SET DEFAULT 0;

ALTER TABLE user_contents
    ADD CONSTRAINT chk_scroll_position CHECK (scroll_position >= 0);

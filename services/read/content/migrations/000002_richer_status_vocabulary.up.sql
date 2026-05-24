-- Expand user_contents.status vocabulary from ('unread', 'read', 'archived')
-- to ('unread', 'reading', 'completed', 'archived') to support the staged
-- reading-progress model used by the mobile reader.

-- Backfill existing rows: 'read' becomes 'completed' (terminal read state).
UPDATE user_contents SET status = 'completed' WHERE status = 'read';

-- Replace the CHECK constraint with the new allowed set.
ALTER TABLE user_contents DROP CONSTRAINT chk_status;
ALTER TABLE user_contents
    ADD CONSTRAINT chk_status CHECK (status IN ('unread', 'reading', 'completed', 'archived'));

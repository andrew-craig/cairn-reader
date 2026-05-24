-- Reverse the richer status vocabulary migration. 'reading' collapses back to
-- 'unread' (closest pre-existing state) and 'completed' collapses back to 'read'.

UPDATE user_contents SET status = 'unread' WHERE status = 'reading';
UPDATE user_contents SET status = 'read' WHERE status = 'completed';

ALTER TABLE user_contents DROP CONSTRAINT chk_status;
ALTER TABLE user_contents
    ADD CONSTRAINT chk_status CHECK (status IN ('unread', 'read', 'archived'));

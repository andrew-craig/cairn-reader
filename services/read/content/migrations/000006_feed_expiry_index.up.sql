-- Supports the cleanup job's feed expiry delete (list = 'feed', not favorited, oldest first).
CREATE INDEX idx_user_contents_feed_expiry
    ON user_contents(added_at)
    WHERE list = 'feed' AND is_favorite = false;

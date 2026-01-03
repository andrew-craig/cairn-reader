-- Add UNIQUE constraint to recommendations table to prevent duplicate tracking
-- This constraint is required for ON CONFLICT DO NOTHING to work properly
-- and ensures the same article isn't recommended to the same user multiple times

ALTER TABLE recommendations
ADD CONSTRAINT recommendations_user_article_unique
UNIQUE (user_id, article_id);

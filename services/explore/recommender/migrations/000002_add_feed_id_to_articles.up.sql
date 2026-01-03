-- Add feed_id column to articles table
-- NOTE: feed_id references feeds table in the FETCHER database (separate service)
-- Therefore, no foreign key constraint is used here
ALTER TABLE articles
ADD COLUMN IF NOT EXISTS feed_id INT;

-- Create index for feed_id lookups
CREATE INDEX IF NOT EXISTS idx_articles_feed_id ON articles(feed_id);

-- Phase 1.1: Create tables for voting and recommendation tracking
-- This migration adds the core tables needed for the recommendation engine

-- 1. Drop the existing view that depends on the users table structure
DROP VIEW IF EXISTS user_unread_counts CASCADE;

-- 2. Update users table structure to match plan requirements
-- The existing users table uses VARCHAR(255) for id, but the plan specifies:
-- - id SERIAL PRIMARY KEY (internal ID)
-- - user_id TEXT UNIQUE NOT NULL (external user identifier)
-- We'll create a new users table structure and migrate existing data

-- First, rename the existing users table
ALTER TABLE users RENAME TO users_old;

-- Create new users table with correct structure
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    user_id TEXT UNIQUE NOT NULL,  -- External user identifier
    created_at TIMESTAMP DEFAULT NOW()
);

-- Migrate existing user data (use the old id as the user_id)
INSERT INTO users (user_id, created_at)
SELECT id, created_at FROM users_old;

-- 3. Update user_articles to reference the new users table structure
-- First, add a new user_id_int column for the new user reference
ALTER TABLE user_articles ADD COLUMN user_id_int INT;

-- Populate the new column with the new user IDs
UPDATE user_articles ua
SET user_id_int = u.id
FROM users u
WHERE ua.user_id = u.user_id;

-- Drop the old user_id column and rename user_id_int to user_id
ALTER TABLE user_articles DROP COLUMN user_id CASCADE;
ALTER TABLE user_articles RENAME COLUMN user_id_int TO user_id;

-- Set NOT NULL constraint and add foreign key
ALTER TABLE user_articles ALTER COLUMN user_id SET NOT NULL;
ALTER TABLE user_articles ADD CONSTRAINT user_articles_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- Recreate the primary key on user_articles
ALTER TABLE user_articles DROP CONSTRAINT IF EXISTS user_articles_pkey;
ALTER TABLE user_articles ADD PRIMARY KEY (user_id, article_id);

-- Recreate indexes
DROP INDEX IF EXISTS idx_user_articles_user_id;
CREATE INDEX idx_user_articles_user_id ON user_articles(user_id);

-- Drop the old users table
DROP TABLE users_old CASCADE;

-- 4. Create article_categories table
CREATE TABLE IF NOT EXISTS article_categories (
    article_id TEXT REFERENCES articles(id) ON DELETE CASCADE,
    category TEXT NOT NULL,
    PRIMARY KEY (article_id, category)
);

-- Create index for category lookups
CREATE INDEX IF NOT EXISTS idx_article_categories_category ON article_categories(category);

-- 5. Create votes table: Track individual votes by user
CREATE TABLE IF NOT EXISTS votes (
    id SERIAL PRIMARY KEY,
    user_id INT REFERENCES users(id) ON DELETE CASCADE,
    article_id TEXT REFERENCES articles(id) ON DELETE CASCADE,
    vote_type TEXT CHECK (vote_type IN ('upvote', 'downvote')),
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_id, article_id)
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_votes_article ON votes(article_id);
CREATE INDEX IF NOT EXISTS idx_votes_user ON votes(user_id);
CREATE INDEX IF NOT EXISTS idx_votes_vote_type ON votes(vote_type);

-- 6. Create recommendations table: Track which articles have been recommended to which users
CREATE TABLE IF NOT EXISTS recommendations (
    id SERIAL PRIMARY KEY,
    user_id INT REFERENCES users(id) ON DELETE CASCADE,
    article_id TEXT REFERENCES articles(id) ON DELETE CASCADE,
    recommended_at TIMESTAMP DEFAULT NOW()
);

-- Create composite index for efficient lookups
CREATE INDEX IF NOT EXISTS idx_recommendations_user_article ON recommendations(user_id, article_id);
CREATE INDEX IF NOT EXISTS idx_recommendations_article ON recommendations(article_id);

-- 7. Add additional indexes for recommendation algorithm performance
-- Index for quality score calculation (upvotes, downvotes, recommends)
CREATE INDEX IF NOT EXISTS idx_articles_recommends ON articles(recommends);
CREATE INDEX IF NOT EXISTS idx_articles_quality_score ON articles(upvotes, downvotes, recommends)
    WHERE deleted = false;

-- 8. Recreate the view with the new users table structure
CREATE OR REPLACE VIEW user_unread_counts AS
SELECT
    u.id as user_id,
    COUNT(a.id) - COALESCE(SUM(CASE WHEN ua.read THEN 1 ELSE 0 END), 0) as unread_count
FROM users u
CROSS JOIN articles a
LEFT JOIN user_articles ua ON u.id = ua.user_id AND a.id = ua.article_id
GROUP BY u.id;

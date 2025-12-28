-- Migration: Use external user IDs directly as primary keys
-- This removes the internal user ID mapping pattern and simplifies the schema
-- External user IDs (UUIDs from user service) are used directly in all tables

-- 1. Drop dependent views
DROP VIEW IF EXISTS user_unread_counts CASCADE;

-- 2. Update users table to use external user_id as primary key
-- First, create a new users table with the correct structure
CREATE TABLE users_new (
    id TEXT PRIMARY KEY,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Migrate existing data (map internal ID back to external user_id)
INSERT INTO users_new (id, created_at)
SELECT user_id, created_at FROM users;

-- 3. Update user_articles table to reference external user IDs
-- Create new table with updated schema
CREATE TABLE user_articles_new (
    user_id TEXT NOT NULL,
    article_id TEXT NOT NULL,
    read BOOLEAN DEFAULT FALSE,
    read_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, article_id)
);

-- Migrate data (join to get external user_id)
INSERT INTO user_articles_new (user_id, article_id, read, read_at, created_at)
SELECT u.user_id, ua.article_id, ua.read, ua.read_at, ua.created_at
FROM user_articles ua
JOIN users u ON ua.user_id = u.id;

-- 4. Update votes table
CREATE TABLE votes_new (
    id SERIAL PRIMARY KEY,
    user_id TEXT NOT NULL,
    article_id TEXT NOT NULL,
    vote_type TEXT CHECK (vote_type IN ('upvote', 'downvote')),
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_id, article_id)
);

-- Migrate data
INSERT INTO votes_new (user_id, article_id, vote_type, created_at)
SELECT u.user_id, v.article_id, v.vote_type, v.created_at
FROM votes v
JOIN users u ON v.user_id = u.id;

-- 5. Update recommendations table
CREATE TABLE recommendations_new (
    id SERIAL PRIMARY KEY,
    user_id TEXT NOT NULL,
    article_id TEXT NOT NULL,
    recommended_at TIMESTAMP DEFAULT NOW()
);

-- Migrate data
INSERT INTO recommendations_new (user_id, article_id, recommended_at)
SELECT u.user_id, r.article_id, r.recommended_at
FROM recommendations r
JOIN users u ON r.user_id = u.id;

-- 6. Drop old tables and rename new ones
DROP TABLE user_articles CASCADE;
DROP TABLE votes CASCADE;
DROP TABLE recommendations CASCADE;
DROP TABLE users CASCADE;

ALTER TABLE users_new RENAME TO users;
ALTER TABLE user_articles_new RENAME TO user_articles;
ALTER TABLE votes_new RENAME TO votes;
ALTER TABLE recommendations_new RENAME TO recommendations;

-- 7. Add foreign key constraints
ALTER TABLE user_articles ADD CONSTRAINT user_articles_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE user_articles ADD CONSTRAINT user_articles_article_id_fkey
    FOREIGN KEY (article_id) REFERENCES articles(id) ON DELETE CASCADE;

ALTER TABLE votes ADD CONSTRAINT votes_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE votes ADD CONSTRAINT votes_article_id_fkey
    FOREIGN KEY (article_id) REFERENCES articles(id) ON DELETE CASCADE;

ALTER TABLE recommendations ADD CONSTRAINT recommendations_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE recommendations ADD CONSTRAINT recommendations_article_id_fkey
    FOREIGN KEY (article_id) REFERENCES articles(id) ON DELETE CASCADE;

-- 8. Recreate indexes for performance
CREATE INDEX idx_user_articles_user_id ON user_articles(user_id);
CREATE INDEX idx_user_articles_read ON user_articles(read);

CREATE INDEX idx_votes_article ON votes(article_id);
CREATE INDEX idx_votes_user ON votes(user_id);
CREATE INDEX idx_votes_vote_type ON votes(vote_type);

CREATE INDEX idx_recommendations_user_article ON recommendations(user_id, article_id);
CREATE INDEX idx_recommendations_article ON recommendations(article_id);

-- 9. Add unique constraint on recommendations (user_id, article_id)
-- This was added in migration 005 but needs to be recreated
ALTER TABLE recommendations ADD CONSTRAINT recommendations_user_article_unique
    UNIQUE (user_id, article_id);

-- 10. Recreate the user_unread_counts view
CREATE OR REPLACE VIEW user_unread_counts AS
SELECT
    u.id as user_id,
    COUNT(a.id) - COALESCE(SUM(CASE WHEN ua.read THEN 1 ELSE 0 END), 0) as unread_count
FROM users u
CROSS JOIN articles a
LEFT JOIN user_articles ua ON u.id = ua.user_id AND a.id = ua.article_id
GROUP BY u.id;

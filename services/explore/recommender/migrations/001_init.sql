-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(255) PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP
);

-- Create articles table
CREATE TABLE IF NOT EXISTS articles (
    id VARCHAR(255) PRIMARY KEY,
    title TEXT NOT NULL,
    link TEXT NOT NULL,
    description TEXT,
    content TEXT,
    author VARCHAR(255),
    published TIMESTAMP NOT NULL,
    feed_url TEXT NOT NULL,
    feed_title VARCHAR(255),
    categories TEXT[],
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP
);

-- Create user_articles table (tracks read status)
CREATE TABLE IF NOT EXISTS user_articles (
    user_id VARCHAR(255) REFERENCES users(id) ON DELETE CASCADE,
    article_id VARCHAR(255) REFERENCES articles(id) ON DELETE CASCADE,
    read BOOLEAN DEFAULT FALSE,
    read_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, article_id)
);

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_articles_published ON articles(published DESC);
CREATE INDEX IF NOT EXISTS idx_articles_feed_url ON articles(feed_url);
CREATE INDEX IF NOT EXISTS idx_user_articles_user_id ON user_articles(user_id);
CREATE INDEX IF NOT EXISTS idx_user_articles_read ON user_articles(read);
CREATE INDEX IF NOT EXISTS idx_articles_categories ON articles USING GIN(categories);

-- Create a view for quick access to unread article counts
CREATE OR REPLACE VIEW user_unread_counts AS
SELECT
    u.id as user_id,
    COUNT(a.id) - COALESCE(SUM(CASE WHEN ua.read THEN 1 ELSE 0 END), 0) as unread_count
FROM users u
CROSS JOIN articles a
LEFT JOIN user_articles ua ON u.id = ua.user_id AND a.id = ua.article_id
GROUP BY u.id;

-- Feeds table: RSS feed sources managed by fetcher
CREATE TABLE IF NOT EXISTS feeds (
    id SERIAL PRIMARY KEY,
    url TEXT UNIQUE NOT NULL,
    title TEXT,
    description TEXT,
    last_fetched_at TIMESTAMP,
    consecutive_failures INT DEFAULT 0,
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Fetch history: Track each fetch attempt for monitoring
CREATE TABLE IF NOT EXISTS fetch_history (
    id SERIAL PRIMARY KEY,
    feed_id INT REFERENCES feeds(id) ON DELETE CASCADE,
    fetch_started_at TIMESTAMP NOT NULL,
    fetch_completed_at TIMESTAMP,
    success BOOLEAN,
    articles_found INT DEFAULT 0,
    articles_sent INT DEFAULT 0,
    error_message TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_feeds_last_fetched ON feeds(last_fetched_at NULLS FIRST) WHERE enabled = true;
CREATE INDEX IF NOT EXISTS idx_feeds_enabled ON feeds(enabled);
CREATE INDEX IF NOT EXISTS idx_fetch_history_feed_id ON fetch_history(feed_id);
CREATE INDEX IF NOT EXISTS idx_fetch_history_created_at ON fetch_history(created_at DESC);

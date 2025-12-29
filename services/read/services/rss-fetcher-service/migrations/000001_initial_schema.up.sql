-- RSS Fetcher Service Initial Schema Migration
-- Creates the feeds, feed_subscriptions, feed_items, and content_outbox tables

-- Enable UUID extension if not already enabled
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- 1. Create feeds table
CREATE TABLE feeds (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    feed_url TEXT NOT NULL UNIQUE,

    -- Feed metadata
    title TEXT,
    description TEXT,
    site_url TEXT,

    -- Polling management
    polling_tier VARCHAR(20) NOT NULL DEFAULT 'active', -- 'active', 'moderate', 'quiet'
    last_fetched_at TIMESTAMP WITH TIME ZONE,
    last_published_at TIMESTAMP WITH TIME ZONE,
    next_poll_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- Status and error tracking
    status VARCHAR(20) NOT NULL DEFAULT 'active', -- 'active', 'disabled'
    consecutive_error_days INTEGER NOT NULL DEFAULT 0,
    last_error_at TIMESTAMP WITH TIME ZONE,
    last_error_message TEXT,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- Constraints
    CONSTRAINT chk_polling_tier CHECK (polling_tier IN ('active', 'moderate', 'quiet')),
    CONSTRAINT chk_status CHECK (status IN ('active', 'disabled'))
);

-- Indexes
CREATE INDEX idx_feeds_next_poll ON feeds(next_poll_at) WHERE status = 'active';
CREATE INDEX idx_feeds_polling_tier ON feeds(polling_tier);
CREATE INDEX idx_feeds_last_published ON feeds(last_published_at);

-- 2. Create feed_subscriptions table
CREATE TABLE feed_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL, -- External user ID
    feed_id UUID NOT NULL REFERENCES feeds(id) ON DELETE CASCADE,

    subscribed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    CONSTRAINT unique_user_feed UNIQUE(user_id, feed_id)
);

-- Indexes
CREATE INDEX idx_feed_subs_user ON feed_subscriptions(user_id);
CREATE INDEX idx_feed_subs_feed ON feed_subscriptions(feed_id);

-- Function to enforce 100 feed limit per user
CREATE OR REPLACE FUNCTION check_feed_limit()
RETURNS TRIGGER AS $$
DECLARE
    feed_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO feed_count
    FROM feed_subscriptions
    WHERE user_id = NEW.user_id;

    IF feed_count >= 100 THEN
        RAISE EXCEPTION 'User has reached maximum feed limit of 100';
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_check_feed_limit
BEFORE INSERT ON feed_subscriptions
FOR EACH ROW
EXECUTE FUNCTION check_feed_limit();

-- 3. Create feed_items table
CREATE TABLE feed_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    feed_id UUID NOT NULL REFERENCES feeds(id) ON DELETE CASCADE,

    -- Item identifiers
    item_url TEXT NOT NULL,
    item_guid TEXT, -- RSS GUID if available

    -- Processing status
    processing_status VARCHAR(20) NOT NULL DEFAULT 'pending', -- 'pending', 'processing', 'completed', 'failed'
    content_hash VARCHAR(64), -- SHA-256 hash after processing
    content_service_id UUID, -- ID returned from Content Service

    -- Item metadata (raw from RSS)
    title TEXT,
    author TEXT,
    published_at TIMESTAMP WITH TIME ZONE,
    description TEXT,

    -- Content update tracking
    http_last_modified TEXT, -- Last-Modified header from article fetch
    http_etag TEXT, -- ETag header from article fetch
    last_checked_at TIMESTAMP WITH TIME ZONE, -- When we last checked for updates

    -- Error tracking
    retry_count INTEGER NOT NULL DEFAULT 0,
    last_error TEXT,

    -- Timestamps
    discovered_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    processed_at TIMESTAMP WITH TIME ZONE,

    -- Constraints
    CONSTRAINT chk_processing_status CHECK (
        processing_status IN ('pending', 'processing', 'completed', 'failed')
    ),
    CONSTRAINT unique_feed_item UNIQUE(feed_id, item_guid)
);

-- Indexes
CREATE INDEX idx_feed_items_status ON feed_items(processing_status, discovered_at);
CREATE INDEX idx_feed_items_feed ON feed_items(feed_id, discovered_at DESC);
CREATE INDEX idx_feed_items_hash ON feed_items(feed_id, content_hash)
    WHERE processing_status = 'completed';

-- 4. Create content_outbox table
CREATE TABLE content_outbox (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    feed_item_id UUID NOT NULL REFERENCES feed_items(id) ON DELETE CASCADE,

    -- Processed content ready for delivery
    content_payload JSONB NOT NULL, -- Full content payload for Content Service API
    user_ids UUID[] NOT NULL, -- Array of user IDs to deliver to

    -- Delivery status tracking
    delivery_status VARCHAR(20) NOT NULL DEFAULT 'pending', -- 'pending', 'sending', 'delivered', 'failed'
    retry_count INTEGER NOT NULL DEFAULT 0,
    max_retries INTEGER NOT NULL DEFAULT 6,
    next_retry_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_error TEXT,

    -- Response tracking
    content_service_id UUID, -- ID returned from Content Service on success

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    delivered_at TIMESTAMP WITH TIME ZONE,

    -- Constraints
    CONSTRAINT chk_delivery_status CHECK (
        delivery_status IN ('pending', 'sending', 'delivered', 'failed')
    ),
    CONSTRAINT chk_retry_count CHECK (retry_count >= 0)
);

-- Indexes for efficient delivery worker queries
CREATE INDEX idx_outbox_pending ON content_outbox(next_retry_at)
    WHERE delivery_status IN ('pending', 'sending');
CREATE INDEX idx_outbox_status ON content_outbox(delivery_status, created_at);
CREATE INDEX idx_outbox_feed_item ON content_outbox(feed_item_id);

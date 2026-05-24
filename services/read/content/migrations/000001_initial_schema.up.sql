-- Content Service - Initial Schema
-- This migration creates the core tables for the Content Service

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Contents table: Stores unique content items (shared across users)
CREATE TABLE contents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    content_hash VARCHAR(64) NOT NULL, -- SHA-256 hash
    cleaned_html TEXT NOT NULL, -- Max 5MB enforced at application level
    original_url TEXT NOT NULL,
    canonical_url TEXT, -- Normalized URL for deduplication (future use)
    title TEXT NOT NULL,
    author TEXT,
    published_at TIMESTAMP WITH TIME ZONE,
    description TEXT,
    image_urls TEXT[], -- Array of image URLs
    source_type VARCHAR(50) NOT NULL, -- 'rss', 'web', etc.

    -- Source-specific identifiers (nullable, depends on source_type)
    source_feed_id UUID, -- Feed ID for RSS content (more efficient than JSONB extraction)

    -- Content-type specific metadata stored as JSONB for flexibility
    -- Use for additional metadata that doesn't need indexing
    metadata JSONB,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    orphaned_at TIMESTAMP WITH TIME ZONE, -- When last user-content relationship was deleted

    -- Constraints
    CONSTRAINT chk_html_size CHECK (octet_length(cleaned_html) <= 5242880) -- 5MB
);

-- Composite index for RSS deduplication (using dedicated column for efficiency)
CREATE INDEX idx_contents_rss_dedup ON contents(content_hash, source_feed_id)
    WHERE source_type = 'rss';
CREATE INDEX idx_contents_orphaned ON contents(orphaned_at)
    WHERE orphaned_at IS NOT NULL;
CREATE INDEX idx_contents_url ON contents(original_url);
CREATE INDEX idx_contents_canonical_url ON contents(canonical_url)
    WHERE canonical_url IS NOT NULL;

-- Full-text search index for title and author (supports basic search feature)
CREATE INDEX idx_contents_search ON contents
    USING gin(to_tsvector('english', coalesce(title, '') || ' ' || coalesce(author, '')));

-- User-Contents table: Junction table mapping users to content with user-specific metadata
CREATE TABLE user_contents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL, -- External user ID, not validated initially
    content_id UUID NOT NULL REFERENCES contents(id) ON DELETE CASCADE,

    -- User-specific metadata
    status VARCHAR(20) NOT NULL DEFAULT 'unread', -- 'unread', 'read', 'archived' (see 000002 for richer vocabulary)
    scroll_position INTEGER NOT NULL DEFAULT 0, -- Character offset
    is_favorite BOOLEAN NOT NULL DEFAULT false,

    -- Timestamps
    added_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- Constraints
    CONSTRAINT chk_status CHECK (status IN ('unread', 'read', 'archived')),
    CONSTRAINT chk_scroll_position CHECK (scroll_position >= 0),
    CONSTRAINT unique_user_content UNIQUE(user_id, content_id)
);

-- Indexes for efficient querying
CREATE INDEX idx_user_contents_user ON user_contents(user_id, added_at DESC);
CREATE INDEX idx_user_contents_status ON user_contents(user_id, status);
CREATE INDEX idx_user_contents_favorite ON user_contents(user_id, is_favorite)
    WHERE is_favorite = true;
CREATE INDEX idx_user_contents_content ON user_contents(content_id);

-- Trigger function to mark content as orphaned when last user removes it
CREATE OR REPLACE FUNCTION mark_content_orphaned()
RETURNS TRIGGER AS $$
BEGIN
    -- Check if this was the last user-content relationship
    IF NOT EXISTS (
        SELECT 1 FROM user_contents WHERE content_id = OLD.content_id
    ) THEN
        UPDATE contents
        SET orphaned_at = NOW()
        WHERE id = OLD.content_id;
    END IF;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

-- Trigger to mark content as orphaned after user-content deletion
CREATE TRIGGER trg_mark_orphaned
AFTER DELETE ON user_contents
FOR EACH ROW
EXECUTE FUNCTION mark_content_orphaned();

-- Trigger function to clear orphaned status when content is re-saved
CREATE OR REPLACE FUNCTION clear_orphaned_status()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE contents
    SET orphaned_at = NULL
    WHERE id = NEW.content_id AND orphaned_at IS NOT NULL;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to clear orphaned status after user-content insertion
CREATE TRIGGER trg_clear_orphaned
AFTER INSERT ON user_contents
FOR EACH ROW
EXECUTE FUNCTION clear_orphaned_status();

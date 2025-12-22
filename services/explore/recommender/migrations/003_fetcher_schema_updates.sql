-- Phase 1: Database Schema Updates for Recommender Service
-- This migration adds voting and recommendation tracking columns

-- 1. Add missing columns to articles table for voting and recommendation tracking
ALTER TABLE articles ADD COLUMN IF NOT EXISTS upvotes INT DEFAULT 0;
ALTER TABLE articles ADD COLUMN IF NOT EXISTS downvotes INT DEFAULT 0;
ALTER TABLE articles ADD COLUMN IF NOT EXISTS recommends INT DEFAULT 0;
ALTER TABLE articles ADD COLUMN IF NOT EXISTS deleted BOOLEAN DEFAULT false;

-- 3. Add UNIQUE constraint on link column for deduplication
-- First, ensure there are no duplicate links (keep the oldest one)
DELETE FROM articles a1
WHERE EXISTS (
    SELECT 1 FROM articles a2
    WHERE a2.link = a1.link
    AND a2.created_at < a1.created_at
);

-- Now add the unique constraint
ALTER TABLE articles ADD CONSTRAINT articles_link_unique UNIQUE (link);

-- 4. Add indexes for performance
-- Adding index for articles published_at if not already exists (should exist from 001)
CREATE INDEX IF NOT EXISTS idx_articles_published_desc ON articles(published DESC);

-- 5. Add index for deleted articles (for filtering)
CREATE INDEX IF NOT EXISTS idx_articles_deleted ON articles(deleted) WHERE deleted = false;

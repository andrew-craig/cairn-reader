-- Raw input of each extraction, kept so a bad extraction can be reproduced.
-- 1:1 with contents; expired by the cleanup job (RAW_SOURCE_RETENTION).
CREATE TABLE content_sources (
    content_id UUID PRIMARY KEY REFERENCES contents(id) ON DELETE CASCADE,
    raw_html TEXT NOT NULL,
    readability_failed BOOLEAN NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_source_html_size CHECK (octet_length(raw_html) <= 5242880) -- 5MB
);

-- Supports the cleanup job's source expiry delete (oldest first).
CREATE INDEX idx_content_sources_created_at ON content_sources(created_at);

-- User reports of bad extraction. No FK to contents on purpose: a report
-- carries its own snapshot so it survives content rewrites and deletion.
CREATE TABLE content_feedback (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    content_id UUID NOT NULL,
    categories TEXT[] NOT NULL,
    comment TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'open',

    -- Snapshot taken when the report is submitted. raw_html and
    -- readability_failed are NULL when the content_sources row had expired.
    original_url TEXT NOT NULL,
    title TEXT NOT NULL,
    author TEXT,
    description TEXT,
    image_url TEXT,
    cleaned_html TEXT,
    raw_html TEXT,
    readability_failed BOOLEAN,

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_content_feedback_user_content UNIQUE (user_id, content_id),
    CONSTRAINT chk_feedback_categories CHECK (
        cardinality(categories) >= 1
        AND categories <@ ARRAY['body_missing_content', 'body_extra_content', 'images', 'formatting', 'links', 'title', 'author', 'description', 'lead_image', 'other']::TEXT[]
    ),
    CONSTRAINT chk_feedback_comment_length CHECK (char_length(comment) <= 2000),
    CONSTRAINT chk_feedback_status CHECK (status IN ('open', 'resolved'))
);

-- Supports triage: open reports, oldest first.
CREATE INDEX idx_content_feedback_status_created ON content_feedback(status, created_at);

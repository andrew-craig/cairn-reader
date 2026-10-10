-- Feed vs Reads: every user_contents row lives in exactly one of two lists,
-- chosen per source by the user (see epic_6e4d). Existing rows predate the
-- split and stay in Reads; direct saves are always Reads.
ALTER TABLE user_contents
    ADD COLUMN list VARCHAR(10) NOT NULL DEFAULT 'reads'
        CONSTRAINT user_contents_list_check CHECK (list IN ('feed', 'reads'));

CREATE INDEX idx_user_contents_user_list_added
    ON user_contents(user_id, list, added_at DESC);

-- The email sender that produced the content, so ingest can look up the
-- sender's route. NULL for non-email content.
ALTER TABLE contents ADD COLUMN source_sender_id UUID;

-- Per-user destination for a source: source_key is the feed ID (rss) or the
-- sender ID (email). No row means Reads. Changing a route only affects items
-- delivered afterwards -- existing user_contents rows are never moved.
CREATE TABLE source_routes (
    user_id     UUID        NOT NULL,
    source_type VARCHAR(50) NOT NULL CHECK (source_type IN ('rss', 'email')),
    source_key  UUID        NOT NULL,
    list        VARCHAR(10) NOT NULL CHECK (list IN ('feed', 'reads')),
    created_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, source_type, source_key)
);

-- Add revoked_at so rotated/logged-out tokens are retained (not deleted) until
-- their natural expiry, so reuse of an already-rotated token can be detected
-- instead of silently falling through as "not found".
ALTER TABLE refresh_tokens ADD COLUMN revoked_at TIMESTAMP WITH TIME ZONE;

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_revoked_at ON refresh_tokens(revoked_at) WHERE revoked_at IS NOT NULL;

COMMENT ON COLUMN refresh_tokens.revoked_at IS 'When this token was revoked (rotated away or logged out); NULL means still active. Revoked rows are retained until expires_at so replay of a revoked token can be detected as reuse.';

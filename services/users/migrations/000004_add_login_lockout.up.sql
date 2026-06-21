-- Add per-account failed login tracking and lockout support
ALTER TABLE users
    ADD COLUMN failed_login_attempts INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN locked_until TIMESTAMP WITH TIME ZONE,
    ADD COLUMN last_failed_login_at TIMESTAMP WITH TIME ZONE;

COMMENT ON COLUMN users.failed_login_attempts IS 'Number of consecutive failed login attempts since last success';
COMMENT ON COLUMN users.locked_until IS 'Account is locked until this timestamp (NULL means not locked)';
COMMENT ON COLUMN users.last_failed_login_at IS 'Timestamp of the most recent failed login attempt';

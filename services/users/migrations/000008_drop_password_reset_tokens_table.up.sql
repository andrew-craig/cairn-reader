-- Password reset was never wired up to an email delivery mechanism (tokens were
-- generated and stored but never sent to the user) and the self-host build never
-- initialized the token repository, causing a nil-pointer panic on
-- /forgot-password and /reset-password. The routes and repository have been
-- removed rather than leaving a broken/misleading feature in place.
DROP TABLE IF EXISTS password_reset_tokens;

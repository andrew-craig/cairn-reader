-- Drop the broken trigger that references non-existent updated_at column
-- The refresh_tokens table has last_used_at, not updated_at
-- The code in refresh_token_repository.go already explicitly sets last_used_at
DROP TRIGGER IF EXISTS update_refresh_tokens_last_used_at ON refresh_tokens;

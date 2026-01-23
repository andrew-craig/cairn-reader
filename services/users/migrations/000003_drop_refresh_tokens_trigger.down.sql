-- Recreate the trigger (though it was broken - references wrong column)
-- This is here for rollback completeness only
CREATE TRIGGER update_refresh_tokens_last_used_at
    BEFORE UPDATE ON refresh_tokens
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

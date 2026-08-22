ALTER TABLE content_outbox DROP COLUMN IF EXISTS lease_expires_at;
ALTER TABLE raw_emails DROP COLUMN IF EXISTS lease_expires_at;

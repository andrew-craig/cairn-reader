-- Email Ingest Service - Rollback Initial Schema
-- This migration removes all tables and functions created in 000001_initial_schema.up.sql

-- Drop triggers first
DROP TRIGGER IF EXISTS trg_email_senders_updated_at ON email_senders;

-- Drop trigger functions
DROP FUNCTION IF EXISTS update_email_sender_timestamp();

-- Drop tables in reverse order (accounting for foreign key dependencies)
DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS content_outbox;
DROP TABLE IF EXISTS raw_emails;
DROP TABLE IF EXISTS email_senders;
DROP TABLE IF EXISTS email_addresses;

-- Note: We don't drop the pgcrypto extension as other services may depend on it

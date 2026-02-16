-- Email Ingest Service - Initial Schema
-- This migration creates the core tables for the Email Ingest Service

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- 1. email_addresses table: Maps users to their unique email addresses
CREATE TABLE email_addresses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE,
    local_part VARCHAR(8) NOT NULL UNIQUE,  -- e.g. "k7m2x9pq"

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    CONSTRAINT chk_local_part_format CHECK (local_part ~ '^[a-z0-9]{8}$')
);

CREATE INDEX idx_email_addresses_local_part ON email_addresses(local_part);
CREATE INDEX idx_email_addresses_user ON email_addresses(user_id);

-- 2. email_senders table: Tracks distinct senders for grouping
CREATE TABLE email_senders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    sender_email TEXT NOT NULL,
    sender_name TEXT,

    email_count INTEGER NOT NULL DEFAULT 0,
    last_received_at TIMESTAMP WITH TIME ZONE,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    CONSTRAINT unique_user_sender UNIQUE(user_id, sender_email),
    CONSTRAINT chk_email_count CHECK (email_count >= 0)
);

CREATE INDEX idx_email_senders_user ON email_senders(user_id, last_received_at DESC);
CREATE INDEX idx_email_senders_email ON email_senders(sender_email);

-- 3. raw_emails table: Stores incoming emails before processing
CREATE TABLE raw_emails (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    sender_id UUID REFERENCES email_senders(id),

    -- Raw email data from Cloudflare worker
    recipient TEXT NOT NULL,
    sender_email TEXT NOT NULL,
    sender_name TEXT,
    subject TEXT,
    html_body TEXT,
    text_body TEXT,
    received_at TIMESTAMP WITH TIME ZONE NOT NULL,

    -- Processing status
    processing_status VARCHAR(20) NOT NULL DEFAULT 'pending',
    content_hash VARCHAR(64),
    retry_count INTEGER NOT NULL DEFAULT 0,
    last_error TEXT,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    processed_at TIMESTAMP WITH TIME ZONE,

    CONSTRAINT chk_processing_status CHECK (
        processing_status IN ('pending', 'processing', 'completed', 'failed')
    ),
    CONSTRAINT chk_has_body CHECK (html_body IS NOT NULL OR text_body IS NOT NULL),
    CONSTRAINT chk_retry_count CHECK (retry_count >= 0)
);

CREATE INDEX idx_raw_emails_status ON raw_emails(processing_status, created_at)
    WHERE processing_status IN ('pending', 'processing');
CREATE INDEX idx_raw_emails_user ON raw_emails(user_id, received_at DESC);
CREATE INDEX idx_raw_emails_sender ON raw_emails(sender_id);
CREATE INDEX idx_raw_emails_content_hash ON raw_emails(content_hash)
    WHERE content_hash IS NOT NULL;

-- 4. content_outbox table: Outbox pattern for reliable delivery to Content Service
CREATE TABLE content_outbox (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    raw_email_id UUID NOT NULL REFERENCES raw_emails(id) ON DELETE CASCADE,

    content_payload JSONB NOT NULL,
    user_id UUID NOT NULL,

    delivery_status VARCHAR(20) NOT NULL DEFAULT 'pending',
    retry_count INTEGER NOT NULL DEFAULT 0,
    max_retries INTEGER NOT NULL DEFAULT 6,
    next_retry_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_error TEXT,

    content_service_id UUID,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    delivered_at TIMESTAMP WITH TIME ZONE,

    CONSTRAINT chk_delivery_status CHECK (
        delivery_status IN ('pending', 'sending', 'delivered', 'failed')
    ),
    CONSTRAINT chk_retry_count CHECK (retry_count >= 0),
    CONSTRAINT chk_max_retries CHECK (max_retries > 0)
);

CREATE INDEX idx_content_outbox_pending ON content_outbox(next_retry_at)
    WHERE delivery_status IN ('pending', 'sending');
CREATE INDEX idx_content_outbox_status ON content_outbox(delivery_status, created_at);
CREATE INDEX idx_content_outbox_user ON content_outbox(user_id, created_at DESC);
CREATE INDEX idx_content_outbox_raw_email ON content_outbox(raw_email_id);

-- 5. api_keys table: API key authentication with rotation support
CREATE TABLE api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    key_name VARCHAR(255) NOT NULL UNIQUE,  -- Human-readable identifier (e.g., "cloudflare-worker-prod")
    key_hash VARCHAR(128) NOT NULL UNIQUE,  -- SHA-256 hash of the API key

    status VARCHAR(20) NOT NULL DEFAULT 'active',

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE,
    last_used_at TIMESTAMP WITH TIME ZONE,
    revoked_at TIMESTAMP WITH TIME ZONE,

    created_by VARCHAR(255),  -- Who created this key
    notes TEXT,  -- Optional notes about key purpose

    CONSTRAINT chk_key_status CHECK (status IN ('active', 'expired', 'revoked'))
);

CREATE INDEX idx_api_keys_status ON api_keys(status)
    WHERE status = 'active';
CREATE INDEX idx_api_keys_hash ON api_keys(key_hash)
    WHERE status = 'active';
CREATE INDEX idx_api_keys_expires ON api_keys(expires_at)
    WHERE expires_at IS NOT NULL AND status = 'active';

-- Trigger function to update email_senders.updated_at
CREATE OR REPLACE FUNCTION update_email_sender_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to automatically update email_senders.updated_at
CREATE TRIGGER trg_email_senders_updated_at
BEFORE UPDATE ON email_senders
FOR EACH ROW
EXECUTE FUNCTION update_email_sender_timestamp();

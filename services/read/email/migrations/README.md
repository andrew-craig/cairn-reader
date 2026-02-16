# Database Migrations - Email Ingest Service

This directory contains database migration files for the Cairn Email Ingest Service.

## Migration Files

Migrations are managed using [golang-migrate](https://github.com/golang-migrate/migrate) and follow a sequential numbering scheme:

- `000001_initial_schema` - Creates the core tables for email ingestion and processing

## Database Schema

### email_addresses table

Maps users to their unique email addresses for receiving newsletter emails.

Fields:
- `id` - UUID primary key
- `user_id` - UUID (unique, foreign reference to user service)
- `local_part` - VARCHAR(8), unique 8-character alphanumeric identifier (e.g., "k7m2x9pq")
- `created_at` - Timestamp of address creation

Constraints:
- `chk_local_part_format` - Ensures local_part matches `^[a-z0-9]{8}$`

### email_senders table

Tracks distinct email senders for grouping and analytics.

Fields:
- `id` - UUID primary key
- `user_id` - UUID (which user received emails from this sender)
- `sender_email` - TEXT, sender's email address
- `sender_name` - TEXT, sender's display name (nullable)
- `email_count` - INTEGER, number of emails received from this sender
- `last_received_at` - Timestamp of most recent email
- `created_at` - Timestamp of first email
- `updated_at` - Timestamp of last update (auto-updated via trigger)

Constraints:
- `unique_user_sender` - Unique constraint on (user_id, sender_email)
- `chk_email_count` - Ensures email_count >= 0

### raw_emails table

Stores incoming emails before processing.

Fields:
- `id` - UUID primary key
- `user_id` - UUID (recipient user)
- `sender_id` - UUID (references email_senders)
- `recipient` - TEXT, full recipient email address
- `sender_email` - TEXT, sender's email address
- `sender_name` - TEXT, sender's display name (nullable)
- `subject` - TEXT, email subject (nullable)
- `html_body` - TEXT, HTML email body (nullable)
- `text_body` - TEXT, plain text email body (nullable)
- `received_at` - Timestamp when email was received
- `processing_status` - VARCHAR(20), status: pending, processing, completed, failed
- `content_hash` - VARCHAR(64), SHA-256 hash of content (nullable)
- `retry_count` - INTEGER, number of processing retry attempts
- `last_error` - TEXT, last error message (nullable)
- `created_at` - Timestamp of database insertion
- `processed_at` - Timestamp of successful processing (nullable)

Constraints:
- `chk_processing_status` - Status must be one of: pending, processing, completed, failed
- `chk_has_body` - At least one of html_body or text_body must be present
- `chk_retry_count` - Ensures retry_count >= 0

### content_outbox table

Outbox pattern for reliable delivery to Content Service.

Fields:
- `id` - UUID primary key
- `raw_email_id` - UUID (references raw_emails, cascade delete)
- `content_payload` - JSONB, payload to send to Content Service
- `user_id` - UUID (recipient user)
- `delivery_status` - VARCHAR(20), status: pending, sending, delivered, failed
- `retry_count` - INTEGER, number of delivery retry attempts
- `max_retries` - INTEGER, maximum retry attempts (default 6)
- `next_retry_at` - Timestamp for next retry attempt
- `last_error` - TEXT, last error message (nullable)
- `content_service_id` - UUID, ID returned by Content Service (nullable)
- `created_at` - Timestamp of outbox entry creation
- `delivered_at` - Timestamp of successful delivery (nullable)

Constraints:
- `chk_delivery_status` - Status must be one of: pending, sending, delivered, failed
- `chk_retry_count` - Ensures retry_count >= 0
- `chk_max_retries` - Ensures max_retries > 0

### api_keys table

API key authentication with rotation support.

Fields:
- `id` - UUID primary key
- `key_name` - VARCHAR(255), human-readable identifier (unique)
- `key_hash` - VARCHAR(128), SHA-256 hash of the API key (unique)
- `status` - VARCHAR(20), status: active, expired, revoked
- `created_at` - Timestamp of key creation
- `expires_at` - Timestamp of key expiration (nullable)
- `last_used_at` - Timestamp of last use (nullable)
- `revoked_at` - Timestamp of revocation (nullable)
- `created_by` - VARCHAR(255), who created this key (nullable)
- `notes` - TEXT, optional notes about key purpose (nullable)

Constraints:
- `chk_key_status` - Status must be one of: active, expired, revoked

## Indexes

Indexes are created for query performance:

**email_addresses**:
- `idx_email_addresses_local_part` - Fast lookup by email local part
- `idx_email_addresses_user` - Fast lookup by user_id

**email_senders**:
- `idx_email_senders_user` - Fast lookup by user_id, sorted by last_received_at
- `idx_email_senders_email` - Fast lookup by sender_email

**raw_emails**:
- `idx_raw_emails_status` - Partial index for pending/processing emails
- `idx_raw_emails_user` - User's emails sorted by received_at
- `idx_raw_emails_sender` - Emails from specific sender
- `idx_raw_emails_content_hash` - Duplicate detection

**content_outbox**:
- `idx_content_outbox_pending` - Partial index for pending/sending entries
- `idx_content_outbox_status` - Entries by delivery status
- `idx_content_outbox_user` - User's outbox entries
- `idx_content_outbox_raw_email` - Outbox entries for specific email

**api_keys**:
- `idx_api_keys_status` - Partial index for active keys
- `idx_api_keys_hash` - Fast lookup for authentication
- `idx_api_keys_expires` - Partial index for expiring keys

## Running Migrations

### Using Docker Compose

Migrations run automatically on service startup when using Docker Compose.

```bash
# Start all services (migrations run automatically)
cd infrastructure/docker/dev
docker compose up -d

# Check service logs to verify migrations
docker compose logs email-ingest
```

### Using golang-migrate CLI

```bash
# Apply all pending migrations
migrate -path services/read/email/migrations \
  -database "postgres://cairn_user_email:password@localhost:5432/ingest_email?sslmode=disable" \
  up

# Rollback the last migration
migrate -path services/read/email/migrations \
  -database "postgres://cairn_user_email:password@localhost:5432/ingest_email?sslmode=disable" \
  down 1

# Check current migration version
migrate -path services/read/email/migrations \
  -database "postgres://cairn_user_email:password@localhost:5432/ingest_email?sslmode=disable" \
  version
```

## Database Setup

### PostgreSQL Requirements

- PostgreSQL 12 or higher
- `pgcrypto` extension (for UUID generation)

### Creating the Database

```sql
-- Create database and user
CREATE USER cairn_user_email WITH PASSWORD 'your_secure_password_here';
CREATE DATABASE ingest_email OWNER cairn_user_email;
GRANT ALL PRIVILEGES ON DATABASE ingest_email TO cairn_user_email;
```

### Configuration

Database configuration is loaded from environment variables:

```bash
DB_HOST=localhost
DB_PORT=5432
DB_USER=cairn_user_email
DB_PASSWORD=your_secure_password_here
DB_NAME=ingest_email
DB_SSL_MODE=disable  # Use "require" or "verify-full" in production
```

## Migration Best Practices

1. **Never modify existing migrations** - Once a migration is committed and deployed, create a new migration for changes
2. **Always test rollbacks** - Ensure down migrations work correctly before deploying
3. **Use transactions** - Migrations run in transactions by default for safety
4. **Index carefully** - Use partial indexes where appropriate to optimize query performance
5. **Document constraints** - Use CHECK constraints to enforce business rules at the database level

## Troubleshooting

### Connection Issues

Ensure your database is running and connection parameters are correct:

```bash
# Test database connection
psql -h localhost -U cairn_user_email -d ingest_email

# Check PostgreSQL is running
pg_isready -h localhost -p 5432
```

### Viewing Table Structure

```sql
-- Connect to database
psql -h localhost -U cairn_user_email -d ingest_email

-- View table structure
\d email_addresses
\d email_senders
\d raw_emails
\d content_outbox
\d api_keys

-- View all tables
\dt

-- View indexes
\di
```

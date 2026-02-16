# Implement Email Source

[DRAFT] THIS ISSUE IS NOT READY FOR IMPLEMENTATION

## Overview

The Email Source adds email newsletter ingestion as a new content source for Cairn. Users receive a unique email address they can subscribe to newsletters with. Incoming emails are forwarded by a Cloudflare Email Worker, processed by a new Email Ingest Service, and delivered to the Content Service for reading.

## Architecture

### System Flow

```
Newsletter Provider
        │
        ▼
  Cloudflare Email Worker (minimal transform)
        │  POST /api/v1/source/email/ingest
        ▼
  Email Ingest Service (port 8087)
   ├── API: receives raw email, stores to DB
   ├── Worker: processes raw emails
   │    ├── Email-specific cleaning (tracking pixels, footers)
   │    └── Readability extraction
   └── Outbox Worker: delivers to Content Service
        │
        ▼
  Content Service (port 8083)
   └── Stores content, creates user_contents entries
```

### New Components

| Component | Location | Port | Database |
|-----------|----------|------|----------|
| Email Ingest Service (API) | `services/read/email/` | 8087 | `ingest_email` |
| Email Ingest Worker | `services/read/email/` | 8088 (health) | `ingest_email` |
| Cloudflare Email Worker | `infrastructure/cloudflare/email-worker/` | — | — |

### Service Boundaries

- **Email Ingest Service** owns: email addresses, sender tracking, raw email storage, content extraction, outbox delivery
- **Content Service** owns: cleaned content storage, user-content relationships (unchanged)
- **User Service**: no changes required — user_id is passed through from the mobile app

## User Flow

### 1. Request Email Address

User taps "Add email source" in the mobile app.

```
Mobile App → POST /api/v1/source/email/user/{user_id}/address
             Authorization: Bearer <jwt>

Response:
{
  "data": {
    "email_address": "k7m2x9pq@read.cairnapp.com",
    "created_at": "2026-02-16T10:00:00Z"
  }
}
```

- Generates an 8-character lowercase alphanumeric address
- Domain is configurable via `EMAIL_DOMAIN` environment variable
- One address per user (subsequent calls return the existing address)
- Address is idempotent — if user already has one, return it

### 2. Subscribe to Newsletters (Outside App)

User copies their email address and subscribes to newsletters using it. This happens entirely outside the app.

### 3. Receive and Process Email

When an email arrives:

1. **Cloudflare Email Worker** extracts minimal data and forwards:
   ```
   POST /api/v1/source/email/ingest
   X-API-Key: <shared_api_key>

   {
     "recipient": "k7m2x9pq@read.cairnapp.com",
     "sender": "newsletter@example.com",
     "sender_name": "Example Newsletter",
     "subject": "Weekly Digest #42",
     "html_body": "<html>...",
     "text_body": "...",
     "received_at": "2026-02-16T10:30:00Z"
   }
   ```

2. **Email Ingest Service** stores the raw email and returns `202 Accepted`

3. **Email Processing Worker** picks up the raw email:
   - Resolves recipient address → user_id
   - Resolves or creates sender record
   - Email-specific pre-cleaning:
     - Strip tracking pixels (1x1 images, known tracker domains)
     - Remove unsubscribe links/footers
     - Remove email client artifacts (View in browser, etc.)
     - Strip inline styles specific to email rendering
   - Readability extraction (reuse existing processor patterns)
   - HTML sanitization (same policy as RSS content)
   - Write to content outbox

4. **Outbox Worker** delivers to Content Service:
   ```
   POST /api/v1/internal/content/user/bulk
   {
     "user_id": "...",
     "html": "<cleaned content>",
     "source_type": "email",
     "title": "Weekly Digest #42",
     "author": "Example Newsletter",
     "url": "",
     "metadata": {
       "sender_email": "newsletter@example.com",
       "sender_id": "<uuid>",
       "subject": "Weekly Digest #42"
     }
   }
   ```

## API Endpoints

### Email Ingest Service (port 8087)

**Health Checks** (unprotected):
```
GET /health/live    → Liveness probe
GET /health/ready   → Readiness + database check
```

**Email Address Management** (JWT protected):
```
POST   /api/v1/source/email/user/{user_id}/address    → Create/get email address
GET    /api/v1/source/email/user/{user_id}/address     → Get email address
DELETE /api/v1/source/email/user/{user_id}/address     → Delete email address (future)
```

**Email Ingestion** (API key protected):
```
POST /api/v1/source/email/ingest    → Receive email from Cloudflare worker
```

**Sender Management** (JWT protected):
```
GET /api/v1/source/email/user/{user_id}/senders    → List senders (grouped)
```

## Database Schema

### `ingest_email` Database

#### 1. `email_addresses` table

Maps users to their unique email addresses.

```sql
CREATE TABLE email_addresses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE,
    local_part VARCHAR(8) NOT NULL UNIQUE,  -- e.g. "k7m2x9pq"

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    CONSTRAINT chk_local_part_format CHECK (local_part ~ '^[a-z0-9]{8}$')
);

CREATE INDEX idx_email_addresses_local_part ON email_addresses(local_part);
CREATE INDEX idx_email_addresses_user ON email_addresses(user_id);
```

#### 2. `email_senders` table

Tracks distinct senders for grouping.

```sql
CREATE TABLE email_senders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    sender_email TEXT NOT NULL,
    sender_name TEXT,

    email_count INTEGER NOT NULL DEFAULT 0,
    last_received_at TIMESTAMP WITH TIME ZONE,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    CONSTRAINT unique_user_sender UNIQUE(user_id, sender_email)
);

CREATE INDEX idx_email_senders_user ON email_senders(user_id, last_received_at DESC);
```

#### 3. `raw_emails` table

Stores incoming emails before processing.

```sql
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
    CONSTRAINT chk_has_body CHECK (html_body IS NOT NULL OR text_body IS NOT NULL)
);

CREATE INDEX idx_raw_emails_status ON raw_emails(processing_status, created_at)
    WHERE processing_status IN ('pending', 'processing');
CREATE INDEX idx_raw_emails_user ON raw_emails(user_id, received_at DESC);
CREATE INDEX idx_raw_emails_sender ON raw_emails(sender_id);
```

#### 4. `content_outbox` table

Same outbox pattern as RSS, adapted for email.

```sql
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
    CONSTRAINT chk_retry_count CHECK (retry_count >= 0)
);

CREATE INDEX idx_email_outbox_pending ON content_outbox(next_retry_at)
    WHERE delivery_status IN ('pending', 'sending');
CREATE INDEX idx_email_outbox_status ON content_outbox(delivery_status, created_at);
```

## Content Service Changes

### New Source Type

Add `source_type = 'email'` as a recognized value. The content service already supports arbitrary source types via the `source_type VARCHAR(50)` column, so no schema changes are needed.

In `models.go`, add:
```go
const SourceTypeEmail = "email"
```

### Metadata Convention

Email content stored in the content service will use the `metadata` JSONB field:
```json
{
  "sender_email": "newsletter@example.com",
  "sender_id": "uuid",
  "subject": "Weekly Digest #42"
}
```

## Cloudflare Email Worker

### Location

`infrastructure/cloudflare/email-worker/`

### Responsibilities

Minimal — extract and forward:
1. Parse incoming email (sender, recipient, subject, HTML body, text body)
2. POST to Email Ingest Service API
3. Authenticate with shared API key (`X-API-Key` header)

### Configuration

```toml
# wrangler.toml
[vars]
EMAIL_INGEST_URL = "https://api.cairnapp.com/api/v1/source/email/ingest"

# Secrets (set via wrangler secret put)
# EMAIL_INGEST_API_KEY
```

### Error Handling

- If the Email Ingest Service returns non-2xx, the worker should return a 4xx/5xx to Cloudflare so the email can be retried per Cloudflare's retry policy
- Log errors for monitoring

## Email Ingest Service Structure

Follows the same clean architecture as Ingest RSS:

```
services/read/email/
├── cmd/
│   ├── email_ingest/main.go           # API server entry point
│   └── email_ingest_worker/main.go    # Background worker entry point
├── internal/
│   ├── api/
│   │   ├── handlers/
│   │   │   ├── address_handler.go     # Email address CRUD
│   │   │   ├── ingest_handler.go      # Receive emails from CF worker
│   │   │   └── sender_handler.go      # List senders
│   │   ├── middleware/
│   │   │   ├── apikey.go              # API key auth for ingest endpoint
│   │   │   └── auth.go                # JWT auth for user endpoints
│   │   ├── dto/
│   │   │   ├── address_dto.go
│   │   │   ├── ingest_dto.go
│   │   │   └── sender_dto.go
│   │   └── router.go
│   ├── service/
│   │   ├── address_service.go         # Address generation & lookup
│   │   ├── email_service.go           # Email processing orchestration
│   │   └── sender_service.go          # Sender tracking
│   ├── repository/
│   │   ├── address.go
│   │   ├── raw_email.go
│   │   ├── sender.go
│   │   └── outbox.go
│   ├── processor/
│   │   ├── email_cleaner.go           # Email-specific cleaning
│   │   └── content_extractor.go       # Readability extraction
│   ├── worker/
│   │   ├── email_processor_worker.go  # Processes raw emails
│   │   └── outbox_worker.go           # Delivers to content service
│   ├── client/
│   │   └── content_service_client.go  # HTTP client for content service
│   ├── jobs/
│   │   ├── raw_email_cleanup.go       # Clean processed raw emails
│   │   └── outbox_cleanup.go          # Clean delivered outbox entries
│   ├── models/
│   │   └── models.go
│   ├── config/
│   │   └── config.go
│   └── database/
│       └── database.go
├── migrations/
│   └── 000001_initial_schema.up.sql
│   └── 000001_initial_schema.down.sql
├── Dockerfile
├── Dockerfile.worker
├── go.mod
├── go.sum
└── Makefile
```

## Configuration

### Environment Variables

```bash
# Database
DB_HOST=cairn-db
DB_PORT=5432
DB_USER=cairn_user_email
DB_PASSWORD=<secret>
DB_NAME=ingest_email
DB_SSL_MODE=disable

# Server
PORT=8087
HEALTH_PORT=8088

# Email
EMAIL_DOMAIN=read.cairnapp.com

# Authentication
INGEST_API_KEY=<shared_secret>       # For Cloudflare worker auth
VAULT_ADDR=http://vault:8200         # For JWT validation
VAULT_TOKEN=<token>
JWT_PUBLIC_KEY_PATH=secret/data/jwt

# Content Service
CONTENT_SERVICE_URL=http://content-service:8083

# Worker
EMAIL_PROCESS_WORKERS=3
EMAIL_PROCESS_POLL_INTERVAL=5s
OUTBOX_WORKER_COUNT=3
OUTBOX_POLL_INTERVAL=10s
OUTBOX_MAX_RETRIES=6

# Cleanup
RAW_EMAIL_CLEANUP_CRON="0 5 * * *"   # Daily at 5 AM
RAW_EMAIL_RETENTION_DAYS=7            # Keep processed emails 7 days
OUTBOX_CLEANUP_CRON="0 6 * * *"       # Daily at 6 AM
```

## Docker Compose Integration

Add to `infrastructure/docker/dev/docker-compose.yml`:

```yaml
email-ingest:
  build:
    context: ../../../services/read/email
    dockerfile: Dockerfile
  ports:
    - "8087:8087"
  environment:
    - DB_HOST=cairn-db
    - DB_PORT=5432
    - DB_USER=${POSTGRES_USER_EMAIL}
    - DB_PASSWORD=${POSTGRES_PASSWORD_EMAIL}
    - DB_NAME=${POSTGRES_DB_EMAIL}
    - CONTENT_SERVICE_URL=http://content-service:8083
    - EMAIL_DOMAIN=${EMAIL_DOMAIN}
    - INGEST_API_KEY=${EMAIL_INGEST_API_KEY}
    - VAULT_ADDR=http://vault:8200
    - VAULT_TOKEN=${VAULT_DEV_ROOT_TOKEN_ID}
  depends_on:
    cairn-db:
      condition: service_healthy
    vault-init:
      condition: service_completed_successfully

email-ingest-worker:
  build:
    context: ../../../services/read/email
    dockerfile: Dockerfile.worker
  ports:
    - "8088:8088"
  environment:
    # Same DB and service config as email-ingest
    - CONTENT_SERVICE_URL=http://content-service:8083
  depends_on:
    cairn-db:
      condition: service_healthy
```

Add new database to PostgreSQL init:
```sql
CREATE USER cairn_user_email WITH PASSWORD '...';
CREATE DATABASE ingest_email OWNER cairn_user_email;
```

## Background Workers

### Email Processor Worker

- Polls `raw_emails` with `processing_status = 'pending'`
- Worker pool: 3 concurrent workers (configurable)
- Poll interval: 5 seconds
- Processing steps:
  1. Set status to `processing`
  2. Look up user_id from `email_addresses` by recipient local_part
  3. Upsert `email_senders` record, increment `email_count`
  4. Email-specific cleaning (tracking pixels, footers, etc.)
  5. Readability extraction
  6. HTML sanitization
  7. Create `content_outbox` entry
  8. Set status to `completed`
- On failure: increment `retry_count`, set `last_error`, leave as `pending`
- After 5 retries: set status to `failed`

### Outbox Worker

- Same pattern as RSS outbox worker
- Polls `content_outbox` with `delivery_status = 'pending'`
- Worker pool: 3 concurrent workers
- Retry schedule: 1m → 5m → 15m → 1h → 4h → 12h
- Circuit breaker on content service client
- Delivers via `POST /api/v1/internal/content/user/bulk`

### Cleanup Jobs

- **Raw email cleanup** (daily 5 AM): Delete `completed` raw emails older than 7 days
- **Outbox cleanup** (daily 6 AM): Delete `delivered` outbox entries older than 7 days

## Sender Grouping

Senders are automatically tracked in `email_senders`. The API exposes sender listing for the mobile app to display grouped views:

```
GET /api/v1/source/email/user/{user_id}/senders

Response:
{
  "data": [
    {
      "id": "uuid",
      "sender_email": "newsletter@example.com",
      "sender_name": "Example Newsletter",
      "email_count": 42,
      "last_received_at": "2026-02-16T10:30:00Z"
    }
  ]
}
```

The content service already supports filtering by metadata, so the mobile app can filter content by sender using:
```
GET /api/v1/content/user/{user_id}?source_type=email&metadata.sender_id=<uuid>
```

If this query pattern is insufficient, a dedicated endpoint can be added to the content service later.

## Future Enhancements (Out of Scope for Initial Release)

- **Email address rotation**: Allow users to regenerate their address (invalidates old one)
- **Unsubscribe management**: Track unsubscribe links from emails, allow one-tap unsubscribe from within the app
- **Sender blocking**: Allow users to block specific senders
- **Spam filtering**: Basic sender reputation or user-reported spam
- **Attachment handling**: Extract and store PDF/image attachments
- **DKIM/SPF validation**: Verify email authenticity at the Cloudflare worker level
- **Multiple addresses per user**: For organizing different newsletter categories

## Security Considerations

- **API key rotation**: The shared API key between Cloudflare worker and email service should be rotatable without downtime (support checking against current + previous key)
- **Rate limiting on ingest**: Limit inbound emails per address to prevent abuse (e.g., 100/hour)
- **Email size limit**: Reject emails larger than 5MB (matches content service limit)
- **HTML sanitization**: Same bluemonday policy as RSS content — strip scripts, event handlers, iframes, forms
- **No PII storage beyond email addresses**: Raw email bodies are cleaned up after processing
- **Address enumeration prevention**: The ingest endpoint should return 202 regardless of whether the recipient address exists (prevents probing for valid addresses)

## Success Criteria

The email source implementation is considered successful when:

1. Users can request a unique email address via the API
2. Cloudflare worker receives emails and forwards to the email service
3. Email service processes and cleans newsletter content
4. Processed content appears in the user's reading list via the content service
5. Senders are tracked and grouped for user display
6. Invalid/unknown recipient addresses are silently accepted (no information leakage)
7. Processing failures are retried with the outbox pattern
8. Raw emails are cleaned up after processing

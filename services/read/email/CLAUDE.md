# CLAUDE.md - Email Ingest Service

This file provides guidance to Claude Code (claude.ai/code) when working with the Email Ingest service.

## Service Overview

The Email Ingest service accepts forwarded emails (via a Cloudflare Email Worker), processes them, and delivers sanitised content to the Content Service. It runs as two binaries: an HTTP API server and a background worker.

**Location**: `/services/read/email/`

**Components**:
- **API Server** (`cmd/email_ingest/`): HTTP endpoints for ingestion, address management, and sender listing (port 8087)
- **Worker** (`cmd/email_ingest_worker/`): Background email processing, outbox delivery, and cleanup jobs

## Quick Start

```bash
# Run via the project-wide Docker Compose (recommended)
cd /home/user/cairn-reader/infrastructure/docker/dev
docker compose up -d email-ingest email-ingest-worker

# Verify
curl http://localhost:8089/health/ready

# Run tests
cd /home/user/cairn-reader/services/read/email
go test ./...
```

## Architecture

```
Cloudflare Email Worker
        │
        ▼  POST /api/v1/source/email/ingest (API key)
┌─────────────────────────┐
│  Email Ingest API       │
│  (port 8087)            │
│                         │
│  - Ingest endpoint      │
│  - Address management   │
│  - Sender listing       │
└──────────┬──────────────┘
           │  raw_emails table
           ▼
┌─────────────────────────┐         ┌──────────────────────┐
│  Email Ingest Worker    │────────▶│  Content Service     │
│                         │  HTTP   │  (port 8083)         │
│  - EmailProcessorWorker │         │                      │
│  - OutboxWorker         │         └──────────────────────┘
│  - RawEmailCleanup      │
│  - OutboxCleanup        │
└──────────┬──────────────┘
           │
     ┌─────▼─────┐
     │ PostgreSQL │
     │ (email_ingest) │
     └───────────┘
```

### Key Principles

1. **Sanitise only** — no readability extraction; emails are cleaned and sanitised with bluemonday
2. **One address per user** — 8-char alphanumeric local part, no regeneration
3. **Open by default** — all senders accepted, no allowlist/blocklist
4. **Outbox pattern** — reliable delivery to Content Service with exponential backoff
5. **Circuit breaker** — protects Content Service from cascading failures

## Directory Structure

```
services/read/email/
├── cmd/
│   ├── email_ingest/          # API server entry point
│   └── email_ingest_worker/   # Worker entry point
├── internal/
│   ├── api/
│   │   ├── dto/               # Request/response DTOs
│   │   ├── handlers/          # HTTP handlers
│   │   ├── middleware/        # API key and JWT auth
│   │   └── router.go          # Route definitions
│   ├── client/                # Content Service HTTP client
│   ├── config/                # Configuration from env vars
│   ├── database/              # Connection and migrations
│   ├── jobs/                  # Cleanup cron jobs
│   ├── models/                # Domain models
│   ├── processor/             # EmailCleaner, ContentExtractor
│   ├── repository/            # Database repositories
│   ├── service/               # Business logic
│   └── worker/                # Background workers
├── migrations/                # SQL migrations
├── api/                       # OpenAPI spec
├── Dockerfile                 # API server image
├── Dockerfile.worker          # Worker image
├── go.mod
└── go.sum
```

## Data Models

### email_addresses

| Column     | Type         | Notes                          |
|------------|-------------|--------------------------------|
| id         | UUID PK     |                                |
| user_id    | UUID UNIQUE | One address per user           |
| local_part | VARCHAR(8)  | UNIQUE, alphanumeric lowercase |
| created_at | TIMESTAMPTZ |                                |

### email_senders

| Column           | Type         | Notes                              |
|-----------------|-------------|-------------------------------------|
| id              | UUID PK     |                                     |
| user_id         | UUID        | UNIQUE(user_id, sender_email)       |
| sender_email    | VARCHAR     |                                     |
| sender_name     | VARCHAR     | nullable                            |
| email_count     | INTEGER     | auto-incremented via upsert         |
| last_received_at| TIMESTAMPTZ | nullable                            |
| created_at      | TIMESTAMPTZ |                                     |
| updated_at      | TIMESTAMPTZ | auto-updated via trigger            |

### raw_emails

| Column            | Type         | Notes                                  |
|-------------------|-------------|----------------------------------------|
| id                | UUID PK     |                                        |
| user_id           | UUID        |                                        |
| sender_id         | UUID FK     | nullable, references email_senders     |
| recipient         | VARCHAR     |                                        |
| sender_email      | VARCHAR     |                                        |
| sender_name       | VARCHAR     | nullable                               |
| subject           | VARCHAR     | nullable                               |
| html_body         | TEXT        | at least one of html/text required     |
| text_body         | TEXT        |                                        |
| received_at       | TIMESTAMPTZ |                                        |
| processing_status | VARCHAR     | pending/processing/completed/failed    |
| content_hash      | CHAR(64)    | nullable                               |
| retry_count       | INTEGER     | auto-fails at 5                        |
| last_error        | TEXT        | nullable                               |
| created_at        | TIMESTAMPTZ |                                        |
| processed_at      | TIMESTAMPTZ | nullable                               |

### content_outbox

| Column             | Type         | Notes                                |
|--------------------|-------------|--------------------------------------|
| id                 | UUID PK     |                                      |
| raw_email_id       | UUID FK     | ON DELETE CASCADE                    |
| content_payload    | JSONB       |                                      |
| user_id            | UUID        |                                      |
| delivery_status    | VARCHAR     | pending/sending/delivered/failed     |
| retry_count        | INTEGER     |                                      |
| max_retries        | INTEGER     |                                      |
| next_retry_at      | TIMESTAMPTZ |                                      |
| last_error         | TEXT        | nullable                             |
| content_service_id | UUID        | nullable, set on delivery            |
| created_at         | TIMESTAMPTZ |                                      |
| delivered_at       | TIMESTAMPTZ | nullable                             |

### api_keys

| Column       | Type         | Notes                          |
|-------------|-------------|--------------------------------|
| id          | UUID PK     |                                |
| name        | VARCHAR     |                                |
| key_hash    | CHAR(64)    | SHA-256 of the raw key         |
| status      | VARCHAR     | active/expired/revoked         |
| expires_at  | TIMESTAMPTZ | nullable                       |
| last_used_at| TIMESTAMPTZ | nullable                       |
| created_at  | TIMESTAMPTZ |                                |

## API Endpoints

### Ingest (API key protected)

```
POST /api/v1/source/email/ingest
  Header: X-API-Key: <key>
  Body: {"recipient": "...", "sender": "...", "received_at": "...", ...}
  Response 202: {"data": {"accepted": true}, "meta": {...}}
```

### Address Management (JWT protected)

```
POST /api/v1/source/email/user/{user_id}/address   → Get or create address
GET  /api/v1/source/email/user/{user_id}/address    → Get existing address (404 if none)
  Response 200: {"data": {"email_address": "abc@read.cairnapp.com", "created_at": "..."}, "meta": {...}}
```

### Sender Listing (JWT protected)

```
GET /api/v1/source/email/user/{user_id}/senders?limit=20&offset=0
  Response 200: {"data": [{"id": "...", "sender_email": "...", "email_count": 5, ...}], "meta": {...}}
```

### Health Checks

```
GET /health/live    → 200 OK (plain text)
GET /health/ready   → 200 OK or 503 if DB unreachable
```

## Background Workers & Jobs

| Component              | Purpose                                     | Schedule         |
|------------------------|---------------------------------------------|------------------|
| EmailProcessorWorker   | Clean, sanitise, and create outbox entries   | Poll every 5s    |
| OutboxWorker           | Deliver to Content Service with retries      | Poll every 10s   |
| RawEmailCleanupJob     | Delete completed raw_emails > 7 days old     | Daily at 5 AM    |
| OutboxCleanupJob       | Delete delivered outbox entries > 7 days old  | Daily at 6 AM    |

### Email Processing Pipeline

1. Fetch pending `raw_emails` (batch of 20, semaphore-limited to 3 concurrent)
2. Upsert sender record
3. **EmailCleaner**: Remove tracking pixels, unsubscribe footers, preheader text
4. **ContentExtractor**: Sanitise HTML with bluemonday, generate SHA-256 content hash
5. Create `content_outbox` entry with payload `{url: "email://<uuid>", html, title, author, source_type: "email"}`
6. Mark raw email as completed

### Outbox Delivery

- Calls `POST /api/v1/internal/content/user/bulk` on Content Service
- Exponential backoff: 1m, 2m, 4m, 8m, 16m, 32m (max 6 retries)
- Non-retryable on 4xx (400, 401, 403, 404, 422)
- Circuit breaker: opens after 5 consecutive failures, half-open after 30s

## Configuration

All values loaded from environment variables:

```bash
# Server
PORT=8087

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=cairn
DB_PASSWORD=<required>
DB_NAME=email_ingest
DB_SSL_MODE=disable

# Email
EMAIL_DOMAIN=read.cairnapp.com

# Auth
INGEST_API_KEY=<required>
VAULT_ADDR=http://localhost:8200
VAULT_TOKEN=dev-root-token
JWT_PUBLIC_KEY_PATH=secret/data/jwt/public-key

# Worker
EMAIL_PROCESSORS=3
EMAIL_POLL_INTERVAL=5s
OUTBOX_POLL_INTERVAL=10s
MAX_RETRIES=5
RAW_EMAIL_CLEANUP_CRON=0 5 * * *
OUTBOX_CLEANUP_CRON=0 6 * * *
RETENTION_DAYS=7

# Content Service
CONTENT_SERVICE_URL=http://content-service:8083
CONTENT_SERVICE_TIMEOUT=30s

# Logging
LOG_LEVEL=info
LOG_FORMAT=json
```

## Testing

```bash
# All tests
go test ./...

# Specific packages
go test ./internal/service/...
go test ./internal/api/handlers/...
go test ./internal/worker/...
go test ./internal/processor/...
```

Tests use mocked dependencies (no database required for unit tests).

## Technology Stack

- **Routing**: go-chi/chi v5
- **Database**: lib/pq (PostgreSQL), golang-migrate
- **Sanitisation**: bluemonday
- **Circuit breaker**: sony/gobreaker
- **JWT**: golang-jwt/jwt v5
- **Shared packages**: pkg/logging, pkg/auth, pkg/middleware, pkg/api

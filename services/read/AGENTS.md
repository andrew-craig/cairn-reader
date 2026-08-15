# CLAUDE.md - Read Service

This file provides guidance to Claude Code (claude.ai/code) when working with the Read service in this directory.

## Service Overview

The Read service is a microservices-based backend system that provides article storage, RSS feed management, and content delivery functionality for the Cairn read-it-later application.

**Location**: `/services/read/`

**Components**:
- **Content Service** (`content/`): Stores and serves article content with user-specific metadata (port 8083)
- **Ingest RSS Service** (`fetcher/`): Manages RSS feed subscriptions and content delivery (port 8085)
- **Email Ingest Service** (`email/`): Email-to-article ingestion pipeline, delivers into Content Service the same way as Ingest RSS. See [email/CLAUDE.md](/services/read/email/CLAUDE.md) for details.

## Quick Start

### Start All Services

There is no `docker-compose.yml` under `services/read/`. The dev topology lives in
`infrastructure/docker/dev/docker-compose.yml`, which starts every backend service
(User, Explore, Read, Email Ingest) against one consolidated Postgres container.

```bash
cd /home/user/cairn-reader/infrastructure/docker/dev

# Start all services (Read Service + everything else)
docker compose up -d
```

This starts, among other services:
- Content Service (http://localhost:8083, container port 8080)
- Ingest RSS Service (http://localhost:8085, container port 8081)
- Email Ingest Service (http://localhost:8089, container port 8087)
- `cairn-db`: one Postgres container hosting separate logical databases per service
  (`content_service` for Content Service, `rss_fetcher_service` for Ingest RSS)
- Background workers for content cleanup, feed polling, and outbox delivery

See [infrastructure/docker/README.md](/infrastructure/docker/README.md) for the full topology.

### Verify Services

```bash
# Check Content Service
curl http://localhost:8083/health/ready

# Check Ingest RSS Service
curl http://localhost:8085/health/ready
```

### Common Commands

These are the targets actually defined in `services/read/Makefile` (run `make help` for the full list):

```bash
# Build services
make build-all                # Build content, content-worker, fetcher, fetcher-worker, email, email-worker
make build                    # Alias for build-all
make build-content            # Build content service only
make build-fetcher             # Build ingest_rss service only
make build-email               # Build email ingest service only

# Testing
make test-all                 # Run tests for content, fetcher, and email
make test                     # Alias for test-all
make test-content             # Run content service tests only
make test-fetcher              # Run fetcher service tests only
# Integration tests use the `integration` build tag and TEST_DB_* env vars
# (see internal/testutil/database.go); run them directly with go test, e.g.:
cd content && go test -tags=integration ./...

# Database migrations
make migrate-up               # Apply pending migrations for content + fetcher
make migrate-down             # Rollback last migration for content + fetcher
make migrate-up-content       # Apply pending migrations for content only
make migrate-create-content name=...  # Create new migration for content
make migrate-status-content   # Show migration status for content
# (equivalent *-fetcher targets also exist)

# Docker operations (from infrastructure/docker/dev/)
docker compose up -d          # Start services
docker compose down           # Stop services
docker compose logs -f        # Show logs

# Code quality
make fmt                      # Format Go code (content, fetcher, email)
make vet                      # Run go vet
make lint                     # Run fmt + vet
```

## Architecture

### System Architecture

```
┌─────────────────────┐         ┌──────────────────────┐
│  Content Service    │         │   Ingest RSS Service │
│   (port 8083)       │         │    (port 8085)       │
│                     │         │                      │
│  - Content Storage  │◄────────│  - Feed Polling      │
│  - User Lists       │  REST   │  - Content Extraction│
│  - Search           │   API   │  - Subscription Mgmt │
│  - Metadata         │         │  - Outbox Delivery   │
└──────────┬──────────┘         └──────────┬───────────┘
           │                               │
           │                               │
     ┌─────▼─────┐                   ┌──────────────────┐
     │ PostgreSQL│                   │    PostgreSQL     │
     │ (content_ │                   │(rss_fetcher_      │
     │  service) │                   │      service)     │
     └───────────┘                   └──────────────────┘
```

Both logical databases live in the same consolidated `cairn-db` Postgres container in dev
(`infrastructure/docker/dev/docker-compose.yml`) — this diagram shows logical isolation, not
separate containers.

**Key Principles**:
1. **Service Isolation**: Each service has its own database
2. **REST-Only Communication**: No direct database access between services
3. **Reliability**: Outbox pattern ensures content delivery survives failures
4. **Efficiency**: Tiered polling reduces load on inactive feeds
5. **Resilience**: Circuit breaker protects against cascading failures

### Directory Structure

```
services/read/
├── content/                    # Content Service
│   ├── api/                   # OpenAPI specs
│   ├── cmd/
│   │   ├── content/          # HTTP server entry point (binary: content)
│   │   └── worker/           # Background worker entry point (binary: content-worker)
│   ├── internal/
│   │   ├── api/              # HTTP handlers, middleware, DTOs
│   │   ├── repository/       # Database layer
│   │   ├── service/          # Business logic
│   │   ├── processor/        # Content processing (readability, sanitization)
│   │   ├── jobs/             # Background jobs (orphaned content cleanup)
│   │   └── config/           # Configuration
│   ├── migrations/           # Database migrations
│   ├── Dockerfile            # API server image
│   ├── Dockerfile.worker     # Background worker image
│   └── integration_test.go   # Integration tests
│
├── fetcher/                   # Ingest RSS Service
│   ├── api/                   # OpenAPI specs
│   ├── cmd/
│   │   ├── ingest_rss/       # HTTP server entry point (binary: ingest_rss)
│   │   └── ingest_rss_worker/ # Background worker entry point (binary: ingest_rss_worker)
│   ├── internal/
│   │   ├── api/              # HTTP handlers, middleware, DTOs
│   │   ├── repository/       # Database layer
│   │   ├── service/          # Business logic
│   │   ├── fetcher/          # Feed fetching and parsing
│   │   ├── processor/        # Content extraction, update detection
│   │   ├── worker/           # Background workers (outbox, per-feed fetch pool)
│   │   ├── scheduler/        # Poll scheduling and tier management
│   │   ├── jobs/             # Scheduled jobs (outbox cleanup, feed items cleanup, content extraction)
│   │   ├── client/           # Content Service HTTP client
│   │   └── config/           # Configuration
│   ├── migrations/           # Database migrations
│   ├── Dockerfile            # API server image
│   ├── Dockerfile.worker     # Background worker image
│   └── integration_test.go   # Integration tests
│
├── email/                      # Email Ingest Service (see email/CLAUDE.md)
├── api/                       # Shared OpenAPI documentation
├── scripts/                   # Utility scripts
├── Makefile                   # Build and development commands
└── README.md                  # Main documentation
```

Note: there is no `docker-compose.yml` under `services/read/` — see [Quick Start](#quick-start) above.

## Data Models

### Content Service Database (`content_service`)

**contents** table:
```sql
id                UUID PRIMARY KEY DEFAULT gen_random_uuid()
content_hash      VARCHAR(64) NOT NULL          -- SHA-256 hash
cleaned_html      TEXT NOT NULL                 -- Max 5MB, enforced by CHECK constraint
original_url      TEXT NOT NULL
canonical_url     TEXT                          -- Normalized URL (future use)
title             TEXT NOT NULL
author            TEXT
published_at      TIMESTAMP WITH TIME ZONE
description       TEXT
image_urls        TEXT[]                        -- Array of image URLs
source_type       VARCHAR(50) NOT NULL          -- 'rss', 'web', 'email'
source_feed_id    UUID                          -- Set for RSS-sourced content
metadata          JSONB                         -- Free-form, source-specific metadata
created_at        TIMESTAMP WITH TIME ZONE DEFAULT NOW()
updated_at        TIMESTAMP WITH TIME ZONE DEFAULT NOW()
orphaned_at       TIMESTAMP WITH TIME ZONE      -- Set when last user_contents row is deleted

-- Constraints & Indexes
CHECK (octet_length(cleaned_html) <= 5242880)                       -- 5MB, hardcoded (not env-configurable)
UNIQUE INDEX(content_hash, source_feed_id) WHERE source_type = 'rss'      -- RSS deduplication
UNIQUE INDEX(content_hash, original_url) WHERE source_type != 'rss'      -- email/manual deduplication
INDEX(orphaned_at) WHERE orphaned_at IS NOT NULL
INDEX(original_url)
INDEX(canonical_url) WHERE canonical_url IS NOT NULL

-- Full-text search
GIN INDEX on (to_tsvector('english', coalesce(title, '') || ' ' || coalesce(author, '')))
```

**user_contents** table:
```sql
id                UUID PRIMARY KEY DEFAULT gen_random_uuid()
user_id           UUID NOT NULL                 -- External user ID, not validated here
content_id        UUID NOT NULL REFERENCES contents(id) ON DELETE CASCADE
status            VARCHAR(20) NOT NULL DEFAULT 'unread'   -- 'unread'|'reading'|'completed'|'archived'
scroll_position   NUMERIC(5,4) NOT NULL DEFAULT 0.0       -- Fraction [0,1] of article scrolled
is_favorite       BOOLEAN NOT NULL DEFAULT false
added_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW()
updated_at        TIMESTAMP WITH TIME ZONE DEFAULT NOW()

-- Constraints & Indexes
UNIQUE(user_id, content_id)
CHECK (status IN ('unread', 'reading', 'completed', 'archived'))
CHECK (scroll_position >= 0 AND scroll_position <= 1)
INDEX(user_id, added_at DESC)
INDEX(user_id, status)
INDEX(user_id, is_favorite) WHERE is_favorite = true
INDEX(content_id)  -- For CASCADE DELETE performance
```

Note: there is no `notes`, `read_at`, or `archived_at` column — only `status`/`added_at`/`updated_at` exist.
`scroll_position` was originally an `INTEGER` pixel/character offset; migration `000003` converted it to a
`NUMERIC(5,4)` fraction in `[0,1]` to match what the mobile/web readers actually persist.

**Triggers**:
- **trg_mark_orphaned**: Sets `contents.orphaned_at` when the last `user_contents` row for that content is deleted
- **trg_clear_orphaned**: Clears `contents.orphaned_at` when a new `user_contents` row is inserted

### Ingest RSS Service Database (`rss_fetcher_service`)

Note: the actual database name is `rss_fetcher_service` (see `POSTGRES_DB_RSS` in
`infrastructure/docker/dev/.env.example`), not `ingest_rss`.

**feeds** table:
```sql
id                     UUID PRIMARY KEY DEFAULT gen_random_uuid()
feed_url               TEXT NOT NULL UNIQUE
title                  TEXT
description            TEXT
site_url               TEXT
polling_tier           VARCHAR(20) NOT NULL DEFAULT 'active'   -- 'active'|'moderate'|'quiet'
status                 VARCHAR(20) NOT NULL DEFAULT 'active'   -- 'active'|'disabled'
last_fetched_at        TIMESTAMP WITH TIME ZONE
last_published_at      TIMESTAMP WITH TIME ZONE
next_poll_at           TIMESTAMP WITH TIME ZONE DEFAULT NOW()
consecutive_error_days INTEGER NOT NULL DEFAULT 0
last_error_at          TIMESTAMP WITH TIME ZONE
last_error_message     TEXT
created_at             TIMESTAMP WITH TIME ZONE DEFAULT NOW()
updated_at             TIMESTAMP WITH TIME ZONE DEFAULT NOW()

-- Indexes
INDEX(next_poll_at) WHERE status = 'active'
INDEX(polling_tier)
INDEX(last_published_at)
```

**feed_subscriptions** table (this is the actual user↔feed table name — not `user_feeds`):
```sql
id             UUID PRIMARY KEY DEFAULT gen_random_uuid()
user_id        UUID NOT NULL                    -- External user ID
feed_id        UUID NOT NULL REFERENCES feeds(id) ON DELETE CASCADE
subscribed_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW()

-- Constraints & Indexes
UNIQUE(user_id, feed_id)
INDEX(user_id)
INDEX(feed_id)

-- Trigger: Enforce 100 feed limit per user
```

**feed_items** table:
```sql
id                  UUID PRIMARY KEY DEFAULT gen_random_uuid()
feed_id             UUID NOT NULL REFERENCES feeds(id) ON DELETE CASCADE
item_url            TEXT NOT NULL
item_guid           TEXT                         -- RSS GUID if available (nullable)
processing_status   VARCHAR(20) NOT NULL DEFAULT 'pending'  -- 'pending'|'processing'|'completed'|'failed'
content_hash        VARCHAR(64)                  -- SHA-256 hash after processing
content_service_id  UUID                         -- ID returned from Content Service
title               TEXT
author              TEXT
published_at        TIMESTAMP WITH TIME ZONE
description         TEXT
http_last_modified  TEXT
http_etag           TEXT
last_checked_at     TIMESTAMP WITH TIME ZONE
retry_count         INTEGER NOT NULL DEFAULT 0
last_error          TEXT
discovered_at       TIMESTAMP WITH TIME ZONE DEFAULT NOW()
processed_at        TIMESTAMP WITH TIME ZONE

-- Constraints & Indexes
UNIQUE(feed_id, item_guid)
INDEX(processing_status, discovered_at)
INDEX(feed_id, discovered_at DESC)
INDEX(feed_id, content_hash) WHERE processing_status = 'completed'
```

**content_outbox** table:
```sql
id                UUID PRIMARY KEY DEFAULT gen_random_uuid()
feed_item_id      UUID NOT NULL REFERENCES feed_items(id) ON DELETE CASCADE
content_payload   JSONB NOT NULL                 -- Full content payload for Content Service API
user_ids          UUID[] NOT NULL                -- Array of user IDs to deliver to
delivery_status   VARCHAR(20) NOT NULL DEFAULT 'pending'  -- 'pending'|'sending'|'delivered'|'failed'
retry_count       INTEGER NOT NULL DEFAULT 0
max_retries       INTEGER NOT NULL DEFAULT 6
next_retry_at     TIMESTAMP WITH TIME ZONE DEFAULT NOW()
last_error        TEXT
content_service_id UUID                          -- ID returned from Content Service on success
created_at        TIMESTAMP WITH TIME ZONE DEFAULT NOW()
delivered_at      TIMESTAMP WITH TIME ZONE

-- Indexes
INDEX(next_retry_at) WHERE delivery_status IN ('pending', 'sending')
INDEX(delivery_status, created_at)
INDEX(feed_item_id)
```

## API Endpoints

### Content Service (port 8083)

**Health Checks**:
```
GET  /health/live                                    → Liveness check
GET  /health/ready                                   → Readiness check (includes DB)
```

**URL Detection (Smart Submission)** (⚠️ Requires JWT authentication):
```
POST /api/v1/content/detect                          → Detect if URL is feed or page
     Auth: Bearer <JWT token>
     Body: {"url": "https://example.com"}
     Returns: {"url": "...", "type": "feed|page|unknown", "title": "..."}
     Timeout: 10 seconds
     Returns: 401 if missing/invalid token

POST /api/v1/content/discover-feed                    → Discover an RSS/Atom feed for a page URL
     Auth: Bearer <JWT token>
     Returns: 401 if missing/invalid token
```

**Content Management** (⚠️ Internal only — requires `X-Internal-API-Key` header):
```
POST   /api/v1/content                               → Create content from HTML/URL
       Auth: X-Internal-API-Key header (must match INTERNAL_API_KEY)
       Body: {"url": "...", "html": "...", "source_type": "rss|web|email",
              "source_feed_id": "...", "published_at": "..."}
       Note: no "title" field here — title is extracted from the HTML, not passed in
       Returns: 401 if missing/invalid internal API key

GET    /api/v1/content/{content_id}                  → Get content by ID
       Auth: X-Internal-API-Key header (must match INTERNAL_API_KEY)
       Returns: 401 if missing/invalid internal API key

PUT    /api/v1/content/{content_id}                  → Update existing content
       Auth: X-Internal-API-Key header (must match INTERNAL_API_KEY)
       Body: {"url": "...", "html": "...", "published_at": "..."}
       Returns: 401 if missing/invalid internal API key

POST   /api/v1/content/bulk                          → Bulk create/update (max 100)
       Auth: X-Internal-API-Key header (must match INTERNAL_API_KEY)
       Body: [{"url": "...", "html": "...", "source_type": "rss|web|email",
               "source_feed_id": "...", "title": "...", "author": "..."}, ...]
       Returns: 401 if missing/invalid internal API key

POST   /api/v1/content/check-duplicate               → Check for duplicates
       Auth: X-Internal-API-Key header (must match INTERNAL_API_KEY)
       Body: {"items": [{"content_hash": "...", "source_feed_id": "..."}, ...]}
       Returns: 401 if missing/invalid internal API key
```

**User Content Management** (⚠️ Requires JWT authentication):
```
POST   /api/v1/content/user/{user_id}                → Add URL to user's list
       Auth: Bearer <JWT token>
       Body (recommended): {"url": "...", "type": "feed|page", "title": "..."}
       - If type=feed: Subscribes via Ingest RSS service
       - If type=page: Extracts content and adds to reading list
       Returns: 401 if missing/invalid token, 403 if accessing other user's content

GET    /api/v1/content/user/{user_id}                → List user's contents
       Auth: Bearer <JWT token>
       Query: ?status=..., ?is_favorite=true, ?limit=20, ?offset=0
       Returns: 401 if missing/invalid token, 403 if accessing other user's content

GET    /api/v1/content/user/{user_id}/search         → Full-text search
       Auth: Bearer <JWT token>
       Query: ?q=golang, ?limit=20, ?offset=0
       Returns: 401 if missing/invalid token, 403 if accessing other user's content

GET    /api/v1/content/user/{user_id}/{content_id}   → Get a single user-content item (full HTML body)
       Auth: Bearer <JWT token>
       Returns: 401 if missing/invalid token, 403 if accessing other user's content

PATCH  /api/v1/content/user/{user_id}/{content_id}   → Update user metadata
       Auth: Bearer <JWT token>
       Body: {"status": "unread|reading|completed|archived", "is_favorite": true,
              "scroll_position": 0.5}
       Returns: 401 if missing/invalid token, 403 if accessing other user's content

DELETE /api/v1/content/user/{user_id}/{content_id}   → Remove from user's list
       Auth: Bearer <JWT token>
       Returns: 401 if missing/invalid token, 403 if accessing other user's content

GET    /api/v1/content/user/{user_id}/subscriptions  → List all subscriptions (RSS + email) for the user
       Auth: Bearer <JWT token>
       Note: Aggregates results from Ingest RSS and Email Ingest services

DELETE /api/v1/content/user/{user_id}/subscriptions/rss/{feed_id} → Unsubscribe from an RSS feed
       Auth: Bearer <JWT token>
       Note: Proxies to Ingest RSS Service

POST   /api/v1/content/user/bulk                     → Bulk add to authenticated user
       Auth: Bearer <JWT token>
       Body: [{"url": "...", ...}, ...]
       Returns: 401 if missing/invalid token, 403 if trying to add to other users

POST   /api/v1/internal/content/user/bulk            → Bulk add (internal service-to-service)
       Auth: X-Internal-API-Key header (must match INTERNAL_API_KEY) — NOT unauthenticated
       Body: [{"user_id": "...", "url": "...", ...}, ...]
       Note: Used by internal services (Ingest RSS, Email Ingest) only
```

### Ingest RSS Service (port 8085)

**Health Checks**:
```
GET  /health/live                                               → Liveness check
GET  /health/ready                                              → Readiness check (includes DB)
```

**Feed Subscriptions** (⚠️ Internal only — requires `X-Internal-API-Key` header):
```
POST   /api/v1/source/rss/user/{user_id}/subscription          → Subscribe to feed
       Auth: X-Internal-API-Key header (must match INTERNAL_API_KEY)
       Body: {"feed_url": "https://..."}
       Returns: 401 if missing/invalid internal API key

DELETE /api/v1/source/rss/user/{user_id}/subscription/{feed_id} → Unsubscribe
       Auth: X-Internal-API-Key header (must match INTERNAL_API_KEY)
       Returns: 401 if missing/invalid internal API key

GET    /api/v1/source/rss/user/{user_id}/subscription          → List user's feeds
       Auth: X-Internal-API-Key header (must match INTERNAL_API_KEY)
       Returns: 401 if missing/invalid internal API key
```

**Feed Management** (⚠️ Internal only — requires `X-Internal-API-Key` header):
```
PATCH  /api/v1/source/rss/feed/{feed_id}                       → Enable feed
       Auth: X-Internal-API-Key header (must match INTERNAL_API_KEY)
       Body: {"enabled": true}
       Note: {"enabled": false} is accepted but currently a no-op (disabling isn't wired up yet)
       Returns: 401 if missing/invalid internal API key
```

Note: there is no `GET /api/v1/source/rss/feed` (list all), `GET /api/v1/source/rss/feed/{feed_id}`
(get details), or `POST /api/v1/source/rss/feed/{feed_id}/refresh` endpoint — these are not
implemented in `fetcher/internal/api/router.go`.

This service has no JWT context of its own — every route under `/api/v1/source/rss` is reached
only through the Content Service gateway, which validates the caller's JWT and matches `user_id`
before proxying. `user_id` in the path is never trusted on its own; the `X-Internal-API-Key`
header is what proves the caller is Content Service.

## JWT Authentication

### Overview

The Content Service now uses JWT-based authentication to ensure users can only access their own content. All user-specific endpoints require a valid JWT token signed by the User Service.

### Protected Routes

All routes under `/api/v1/content/user/{user_id}`, plus `/api/v1/content/detect` and
`/api/v1/content/discover-feed`, are protected and require:
- JWT token in `Authorization: Bearer <token>` header
- Token must contain valid `user_id` claim matching the URL parameter (where applicable —
  `/detect` and `/discover-feed` only require a valid token, not a matching user_id)
- Expires: Token expiration validated by service
- Signature: RS256 with public key fetched from Vault at startup

**Protected endpoints**:
- `POST /api/v1/content/detect` - URL detection
- `POST /api/v1/content/discover-feed` - Feed discovery
- `GET /api/v1/content/user/{user_id}` - List user contents
- `POST /api/v1/content/user/{user_id}` - Add content to user
- `GET /api/v1/content/user/{user_id}/search` - Search user contents
- `GET /api/v1/content/user/{user_id}/{content_id}` - Get a single user-content item
- `PATCH /api/v1/content/user/{user_id}/{content_id}` - Update content metadata
- `DELETE /api/v1/content/user/{user_id}/{content_id}` - Delete from user
- `GET /api/v1/content/user/{user_id}/subscriptions` - List all subscriptions
- `DELETE /api/v1/content/user/{user_id}/subscriptions/rss/{feed_id}` - Unsubscribe from RSS feed
- `POST /api/v1/content/user/bulk` - Bulk add for authenticated user

### Internal Routes (require `X-Internal-API-Key`)

- `POST /api/v1/content/` - Create content
- `GET /api/v1/content/{content_id}` - Get content
- `PUT /api/v1/content/{content_id}` - Update content
- `POST /api/v1/content/bulk` - Bulk create
- `POST /api/v1/content/check-duplicate` - Check duplicates
- `POST /api/v1/internal/content/user/bulk` - Bulk add (lives under `/api/v1/internal`, see
  Internal Service Integration below)

### Public Routes (No Auth)

- `GET /health/live` - Liveness probe
- `GET /health/ready` - Readiness probe
- `GET /` - Service info

### Configuration

The service loads JWT public key from HashiCorp Vault at startup:

```bash
# Development
VAULT_ADDR=http://localhost:8200
VAULT_TOKEN=dev-root-token
JWT_PUBLIC_KEY_PATH=secret/data/jwt/public-key
VAULT_AUTH_PATH=approle

# Production
VAULT_ADDR=https://vault.example.com:8200
VAULT_ROLE_ID=<role-id>
VAULT_SECRET_ID=<secret-id>
JWT_PUBLIC_KEY_PATH=secret/data/jwt/public-key
VAULT_AUTH_PATH=approle
```

### Authorization Logic

Each protected handler checks:
1. Extract authenticated user ID from JWT context
2. Extract requested user ID from URL parameter
3. Compare IDs - must match or return 403 Forbidden
4. User can only access their own content

Example from `user_content_handler.go`:
```go
authenticatedUserID, err := auth.GetUserIDOrError(r.Context())  // From JWT
requestedUserID, err := uuid.Parse(chi.URLParam(r, "user_id"))

if authenticatedUserID != requestedUserID {
    // Return 403 Forbidden
}
```

### Error Responses

**401 Unauthorized** - Missing or invalid token:
```json
{"error": "unauthorized", "message": "Missing or invalid authentication token"}
```

**403 Forbidden** - User lacks permission:
```json
{"error": "forbidden", "message": "User can only access their own content"}
```

### Internal Service Integration

The Ingest RSS Service (and Email Ingest Service) call the internal bulk endpoint using a shared
API key instead of a JWT:

```bash
POST /api/v1/internal/content/user/bulk
Header: X-Internal-API-Key: <INTERNAL_API_KEY>
```

This endpoint lets internal services add content to users without providing JWT tokens, but it is
still authenticated — requests must present the correct `X-Internal-API-Key` header (validated by
`internalAuthMiddleware.RequireInternalAPIKey`, see `pkg/auth/internal_auth.go`), and the Content
Service refuses to start without `INTERNAL_API_KEY` configured. It's marked as internal-only and
should not be exposed to clients.

## Key Implementation Details

### Content Processing Pipeline

1. **Fetch URL content** (with 30s timeout)
2. **Apply go-readability** for HTML cleaning
3. **Sanitize HTML** with bluemonday
4. **Generate SHA-256 content hash**
5. **Validate content size** (5MB limit)
6. **Store in database** with deduplication check

**Key Files**:
- `content/internal/processor/content.go`
- `content/internal/processor/readability.go`
- `content/internal/processor/sanitizer.go`

### Deduplication Strategy

RSS content is deduplicated using `(content_hash, source_feed_id)`; email/manual content has no
feed to key off, so it dedupes on `(content_hash, original_url)` instead (the URL is stable per
source item — the RSS item link, or `email://<raw_email_id>` for email).

- Same hash + same feed (RSS) or same hash + same URL (email/manual) = duplicate (skip)
- Same hash + different feed/URL = not duplicate (store)
- Multiple users can share the same content with individual metadata
- Enforced at the DB level by two partial UNIQUE indexes — `idx_contents_rss_dedup
  (content_hash, source_feed_id) WHERE source_type = 'rss'` and `idx_contents_nonrss_dedup
  (content_hash, original_url) WHERE source_type != 'rss'` — plus `ON CONFLICT ... DO UPDATE`
  at every insert site (`Create`, `CreateWithTx`, `BulkCreate`), so concurrent duplicate
  deliveries resolve to the same row instead of racing past an application-level check. The
  `POST /api/v1/content/check-duplicate` endpoint is a read-only pre-check on top of this, not
  the source of truth.

### Feed Polling Strategy

**Tiered Polling**:
- **Tier 1 (Active)**: Every 1 hour - feeds with content in last 7 days
- **Tier 2 (Moderate)**: Every 6 hours - feeds with content in last 30 days
- **Tier 3 (Quiet)**: Every 24 hours - feeds inactive for 30+ days

**Tier Management**:
- Daily job evaluates `last_published_at`
- Automatically promotes/demotes tiers based on activity
- Updates `next_poll_at` accordingly

**Key Files**:
- `fetcher/internal/scheduler/poll_scheduler.go` (polling loop, tier intervals, promotion)
- `fetcher/internal/scheduler/tier_manager.go` (daily tier re-evaluation by activity)
- `fetcher/internal/worker/feed_worker.go` (per-feed fetch worker pool)

### Content Update Detection

Uses HTTP caching headers to detect when articles have been updated:

1. Store `ETag` and `Last-Modified` headers from initial fetch
2. On subsequent fetches, send `If-None-Match` and `If-Modified-Since`
3. If server returns 304 Not Modified, skip processing
4. If content changed, re-fetch, re-process, and update via Content Service

**Key Files**:
- `fetcher/internal/fetcher/conditional_fetcher.go`
- `fetcher/internal/processor/update_detector.go`

### Outbox Pattern for Reliability

The Ingest RSS Service uses the outbox pattern to ensure reliable content delivery:

1. **Content extraction** → Write to `content_outbox` table
2. **Outbox worker** → Process pending entries with retry logic
3. **Delivery** → Call Content Service API to create content
4. **Success** → Mark as delivered, store `content_service_id`
5. **Failure** → Increment retry count, schedule next retry with exponential backoff

**Retry Schedule**:
- Retry 1: 1 minute
- Retry 2: 5 minutes
- Retry 3: 15 minutes
- Retry 4: 1 hour
- Retry 5: 4 hours
- Retry 6: 12 hours
- After 6 retries: Mark as failed

**Circuit Breaker**:
- Opens once at least 5 requests have been made and ≥60% of them failed
- Half-open after 30 seconds
- Prevents overwhelming a failing Content Service

**Key Files**:
- `fetcher/internal/worker/outbox_worker.go`
- `fetcher/internal/client/content_service_client.go`

### Feed Error Handling

- Track `consecutive_error_days` for each feed
- Auto-disable feed after 7 consecutive days of errors
- Store `last_error_at` and `last_error_message` for debugging
- Can be manually re-enabled via API

### User Feed Limit

- Maximum 100 feeds per user
- Enforced by database trigger on `user_feeds` table
- API returns 400 Bad Request if limit exceeded

## Background Jobs & Workers

### Content Service

**Orphaned Content Cleanup Job**:
- **Schedule**: Daily at 2 AM
- **Purpose**: Delete content orphaned for 90+ days
- **Location**: `content/internal/jobs/cleanup_job.go`

### Ingest RSS Service

**Feed Polling Worker**:
- **Schedule**: Continuous
- **Purpose**: Poll feeds based on tiered schedule
- **Location**: `fetcher/internal/worker/feed_worker.go`

**Content Extraction Worker**:
- **Schedule**: Continuous
- **Purpose**: Process pending feed items, extract full content
- **Location**: `fetcher/internal/jobs/content_extraction_job.go`

**Outbox Delivery Worker**:
- **Schedule**: Continuous
- **Purpose**: Deliver content to Content Service with retry logic
- **Location**: `fetcher/internal/worker/outbox_worker.go`

**Tier Management Job**:
- **Schedule**: Daily
- **Purpose**: Adjust feed tiers based on activity
- **Location**: `fetcher/internal/scheduler/tier_manager.go`

**Outbox Cleanup Job**:
- **Schedule**: Daily at 3 AM
- **Purpose**: Delete delivered entries older than 7 days
- **Location**: `fetcher/internal/jobs/outbox_cleanup_job.go`

**Feed Items Cleanup Job**:
- **Schedule**: Daily at 4 AM
- **Purpose**: Delete old completed/failed feed items
- **Location**: `fetcher/internal/jobs/feed_items_cleanup_job.go`

## Testing

### Running Tests

```bash
# Run all unit tests (make test / make test-all — content + fetcher + email)
make test

# Run tests for a specific service
make test-content
make test-fetcher
cd content && go test ./...
cd fetcher && go test ./...

# Run integration tests (requires PostgreSQL; set TEST_DB_* env vars, see internal/testutil/database.go)
cd content && go test -tags=integration ./...
cd fetcher && go test -tags=integration ./...

# Run specific package tests
go test ./content/internal/service/...
go test -v ./fetcher/internal/worker/...

# Coverage (no dedicated make target — use go test directly)
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

Note: `make test-coverage` and `make test-integration` are not defined in the Makefile.

### Test Organization

**Unit Tests** (no external dependencies):
- Service layer: `*_service_test.go`
- Repository layer: `*_repository_test.go` (uses sqlmock)
- Handlers: `*_handler_test.go`
- Workers: `*_worker_test.go`
- Middleware: `*_middleware_test.go`

**Integration Tests** (requires database):
- `content/integration_test.go`
- `fetcher/integration_test.go`
- Tagged with `// +build integration`
- Use `go test -tags=integration`

### Test Coverage

Target: 80%+ across critical paths

Current coverage includes:
- ✅ Service layer business logic
- ✅ Repository CRUD operations
- ✅ API handlers and middleware
- ✅ Background workers and jobs
- ✅ Content processing pipeline
- ✅ RSS fetching and parsing
- ✅ HTTP client with retry logic
- ✅ End-to-end integration flows

## Code Conventions

### Go Code Style

**Error Handling**:
```go
// Wrap errors with context
return fmt.Errorf("failed to create content: %w", err)

// Return errors up the call stack
// Let handlers format HTTP responses
```

**Context Usage**:
```go
// Always use context.Context for database calls
func (r *ContentRepository) Create(ctx context.Context, content *Content) error

// Set timeouts for external HTTP calls
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
```

**Database Operations**:
```go
// Prefer batch operations over loops
INSERT INTO contents (...) VALUES ($1, $2), ($3, $4), ... -- Good
for range items { INSERT INTO ... } -- Bad

// Use ON CONFLICT for upsert semantics
INSERT INTO contents (...) VALUES (...)
ON CONFLICT (content_hash, source_feed_id) DO UPDATE SET ...

// Use transactions for multi-step operations
tx, err := db.BeginTx(ctx, nil)
defer tx.Rollback()
// ... perform operations ...
tx.Commit()
```

**HTTP Clients**:
```go
// Always set timeouts
client := &http.Client{
    Timeout: 30 * time.Second,
}

// Use retry logic with exponential backoff for external APIs
// Use circuit breaker for service-to-service communication
```

### Repository Pattern

**Interface**:
```go
type ContentRepository interface {
    Create(ctx context.Context, content *Content) error
    GetByID(ctx context.Context, id string) (*Content, error)
    Update(ctx context.Context, content *Content) error
    Delete(ctx context.Context, id string) error
}
```

**Implementation**:
- All database access isolated in `*_repository.go` files
- Handlers never directly import database code
- Services inject repository interfaces

### Testing Patterns

**Table-Driven Tests**:
```go
tests := []struct {
    name    string
    input   Input
    want    Output
    wantErr bool
}{
    {"valid input", validInput, expectedOutput, false},
    {"invalid input", invalidInput, nil, true},
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        got, err := Function(tt.input)
        if (err != nil) != tt.wantErr {
            t.Errorf("unexpected error: %v", err)
        }
        if !reflect.DeepEqual(got, tt.want) {
            t.Errorf("got %v, want %v", got, tt.want)
        }
    })
}
```

**Mocking**:
```go
// Use testify/mock for service layer tests
mockRepo := new(MockContentRepository)
mockRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

// Use go-sqlmock for repository layer tests
db, mock, err := sqlmock.New()
mock.ExpectQuery("SELECT").WillReturnRows(...)
```

## Environment Variables

There is no `DATABASE_URL` connection-string variable — both services (and the shared
`pkg/config` package they use) read discrete `DB_HOST`/`DB_PORT`/`DB_USER`/`DB_PASSWORD`/`DB_NAME`
variables. The port variable is `PORT`, not `SERVER_PORT`. `MAX_CONTENT_SIZE` (5MB) and the
90-day orphaned-content cutoff are hardcoded constants, not environment variables — see
`content/internal/processor/content.go` and `content/internal/jobs/cleanup_job.go`. Likewise,
the 100-feed-per-user limit, the 7-day error auto-disable threshold, and the tiered polling
intervals (1h/6h/24h) are hardcoded in the fetcher (`fetcher/internal/scheduler/`), not configurable
via env vars.

### Content Service

```bash
PORT=8080                              # Container port; mapped to host 8083 in dev docker-compose
DB_HOST=cairn-db
DB_PORT=5432
DB_USER=cairn_content
DB_PASSWORD=...
DB_NAME=content_service
LOG_LEVEL=info
VAULT_ADDR=http://vault:8200
VAULT_TOKEN=...                        # Or VAULT_ROLE_ID / VAULT_SECRET_ID for AppRole auth
JWT_PUBLIC_KEY_PATH=secret/data/jwt/public-key
INGEST_RSS_SERVICE_URL=http://ingest-rss:8081
EMAIL_INGEST_SERVICE_URL=http://email-ingest:8087
INTERNAL_API_KEY=...                   # Required; validates X-Internal-API-Key on /api/v1/internal
```

### Ingest RSS Service

```bash
PORT=8081                              # Container port; mapped to host 8085 in dev docker-compose
DB_HOST=cairn-db
DB_PORT=5432
DB_USER=cairn_rss
DB_PASSWORD=...
DB_NAME=rss_fetcher_service
LOG_LEVEL=info
INTERNAL_API_KEY=...                   # Required; validates X-Internal-API-Key on /api/v1/source/rss
```

### Ingest RSS Worker (`cmd/ingest_rss_worker`)

```bash
CONTENT_SERVICE_URL=http://content-service:8080
INTERNAL_API_KEY=...                   # Sent as X-Internal-API-Key when calling Content Service
OUTBOX_CLEANUP_CRON=0 3 * * *
FEED_ITEMS_CLEANUP_CRON=0 4 * * *
HEALTH_PORT=8083                       # Default; dev docker-compose overrides to 8086
```

## Database Migrations

Migrations use `golang-migrate` with numbered SQL files.

### Creating Migrations

```bash
# Create new migration (per-service targets — there is no combined migrate-create)
make migrate-create-content name=add_user_notes
make migrate-create-fetcher name=add_feed_field

# This creates:
# content/migrations/NNN_add_user_notes.up.sql
# content/migrations/NNN_add_user_notes.down.sql
```

### Migration Files

**Up Migration** (`NNN_migration_name.up.sql`):
```sql
-- Add new functionality
CREATE TABLE new_table (...);
ALTER TABLE existing_table ADD COLUMN new_column TEXT;
```

**Down Migration** (`NNN_migration_name.down.sql`):
```sql
-- Reverse the changes
DROP TABLE new_table;
ALTER TABLE existing_table DROP COLUMN new_column;
```

### Running Migrations

```bash
# Apply all pending migrations (both services)
make migrate-up

# Rollback last migration (both services)
make migrate-down

# Check current migration version (per-service — there is no combined migrate-status)
make migrate-status-content
make migrate-status-fetcher
```

Note: there is no `make migrate-force` target in this Makefile.

**Important**: Migrations run automatically on service startup (`database.RunMigrations` in
`content/cmd/content/main.go` and `fetcher/cmd/ingest_rss/main.go`).

## Technology Stack

### Core Dependencies

```go
// Content processing
github.com/go-shiori/go-readability  // HTML readability extraction
github.com/microcosm-cc/bluemonday   // HTML sanitization

// RSS/Atom parsing
github.com/mmcdole/gofeed            // Feed parsing

// Database
github.com/lib/pq                    // PostgreSQL driver
github.com/golang-migrate/migrate/v4 // Database migrations

// HTTP routing
github.com/go-chi/chi                // HTTP router and middleware

// Background jobs
github.com/robfig/cron/v3            // Job scheduling

// Resilience
github.com/sony/gobreaker            // Circuit breaker pattern

// Testing
github.com/stretchr/testify          // Testing framework
github.com/DATA-DOG/go-sqlmock       // SQL mocking
```

## Common Development Tasks

### Adding a New API Endpoint

1. **Define handler** in `internal/api/handlers/`
2. **Register route** in `internal/api/server.go`
3. **Add service method** in `internal/service/`
4. **Add repository method** if database access needed
5. **Write tests** for handler and service
6. **Update OpenAPI spec** in `api/openapi.yaml`

### Adding a New Database Table

1. **Create migration** with `make migrate-create-content name=add_table` (or `migrate-create-fetcher`)
2. **Write up migration** (`NNN_add_table.up.sql`)
3. **Write down migration** (`NNN_add_table.down.sql`)
4. **Run migration** with `make migrate-up`
5. **Create model struct** in `internal/models/`
6. **Create repository** in `internal/repository/`
7. **Write repository tests**

### Adding a New Background Job

1. **Create job file** in `internal/jobs/job_name.go`
2. **Implement job interface**:
   ```go
   type Job interface {
       Run(ctx context.Context) error
       Name() string
   }
   ```
3. **Register job** in the relevant worker entry point (`content/cmd/worker/main.go` or `fetcher/cmd/ingest_rss_worker/main.go`)
4. **Add cron schedule** if needed
5. **Write job tests**

### Debugging

**View logs** (from `infrastructure/docker/dev/`):
```bash
# All services
docker compose logs -f

# Specific service
docker compose logs -f content-service
docker compose logs -f ingest-rss

# Filter by level
docker compose logs -f | grep ERROR
```

**Access database**: dev uses one consolidated `cairn-db` Postgres container with separate
logical databases per service — there is no `postgres-content`/`postgres-fetcher` container:
```bash
# Content Service database
docker compose exec cairn-db psql -U cairn_content -d content_service

# Ingest RSS database
docker compose exec cairn-db psql -U cairn_rss -d rss_fetcher_service
```

**Useful SQL queries**:
```sql
-- Check orphaned content
SELECT id, title, orphaned_at FROM contents WHERE orphaned_at IS NOT NULL;

-- Check feed polling status
SELECT id, title, polling_tier, next_poll_at, status FROM feeds;

-- Check outbox queue
SELECT delivery_status, COUNT(*) FROM content_outbox GROUP BY delivery_status;

-- Check feed items status
SELECT processing_status, COUNT(*) FROM feed_items GROUP BY processing_status;
```

## Important Notes

### Service Communication

- **ALWAYS** use REST APIs for inter-service communication
- **NEVER** access another service's database directly
- Content Service and Ingest RSS Service have separate databases
- Use the HTTP client in `fetcher/internal/client/content_service_client.go`

### Content Updates

When updating content:
- Use `PUT /api/v1/content/{content_id}` endpoint
- Content Service preserves all user-content relationships
- Only the content itself is updated, not user metadata

### Feed Management

- Feeds are auto-disabled after 7 consecutive error days
- Can be re-enabled via `PATCH /api/v1/source/rss/feed/{feed_id}`
- Tier management runs daily to optimize polling frequency

### Content Size Limits

- Maximum content size: 5MB
- Enforced during content processing
- Returns 400 Bad Request if exceeded

### Deduplication

- Content is deduplicated by `(content_hash, source_feed_id)`
- Check for duplicates before creating content
- Use `/api/v1/content/check-duplicate` endpoint

## Documentation References

- **Main README**: `/services/read/README.md` - Service documentation (also drifted in places — trust this file and the Go source over it)
- **Email Ingest Service**: `/services/read/email/CLAUDE.md` - Third sub-service under this directory
- **Content Service API**: `/services/read/content/api/openapi.yaml` - OpenAPI specification
- **Ingest RSS API**: `/services/read/fetcher/api/openapi.yaml` - OpenAPI specification
- **Root CLAUDE.md**: `/CLAUDE.md` - Project-wide guidance and conventions

Note: `IMPLEMENTATION_PLAN.md` and `INTEGRATION_TESTS.md` do not exist in this directory.

## Status & Roadmap

**Current Status**: ✅ Core functionality complete (Phases 0-6)

**Completed**:
- ✅ Content Service with full CRUD, search, deduplication
- ✅ Ingest RSS Service with feed subscriptions and tiered polling
- ✅ Outbox pattern for reliable content delivery
- ✅ Content update detection via HTTP caching headers
- ✅ Background workers and scheduled jobs
- ✅ Comprehensive test coverage (80%+)
- ✅ **JWT Authentication** (Phase 6) - User-content access control with RS256 token validation

**Remaining Work**:
- 🔲 API documentation (OpenAPI/Swagger UI) — specs exist (`api/openapi.yaml`) but no served UI
- 🔲 Production observability (metrics, structured logging)
- 🔲 Performance optimization
- 🔲 Rate limiting (CORS is already applied via `pkg/middleware` in both routers)

**Future Enhancements**:
- Recommendation engine
- Import/export functionality
- GraphQL API option
- WebSocket support for real-time updates

## Getting Help

For issues or questions:
- Check logs: `docker compose logs -f` (from `infrastructure/docker/dev/`)
- Check database state: `docker compose exec cairn-db psql -U cairn_content -d content_service`
- Review tests for usage examples
- Consult OpenAPI specifications for API details
- See troubleshooting guide: `/services/read/README.md#troubleshooting`

# CLAUDE.md - Read Service

This file provides guidance to Claude Code (claude.ai/code) when working with the Read service in this directory.

## Service Overview

The Read service is a microservices-based backend system that provides article storage, RSS feed management, and content delivery functionality for the Cairn read-it-later application.

**Location**: `/services/read/`

**Components**:
- **Content Service** (`content/`): Stores and serves article content with user-specific metadata (port 8083)
- **Ingest RSS Service** (`fetcher/`): Manages RSS feed subscriptions and content delivery (port 8085)

## Quick Start

### Start All Services

```bash
cd /home/user/cairn/services/read

# Start services with Docker Compose
make docker-up

# Or manually
docker-compose up -d
```

This starts:
- Content Service (http://localhost:8083)
- Ingest RSS Service (http://localhost:8085)
- PostgreSQL databases for both services
- Background workers for feed polling and content delivery

### Verify Services

```bash
# Check Content Service
curl http://localhost:8083/health/ready

# Check Ingest RSS Service
curl http://localhost:8085/health/ready
```

### Common Commands

```bash
# Build services
make build                   # Build both services

# Testing
make test                    # Run all tests
make test-coverage           # Generate coverage report
make test-integration        # Run integration tests

# Database migrations
make migrate-up              # Apply pending migrations
make migrate-down            # Rollback last migration
make migrate-status          # Check migration status
make migrate-create name=... # Create new migration

# Docker operations
make docker-up               # Start services
make docker-down             # Stop services
make docker-logs             # Show logs
make docker-restart          # Restart all services

# Code quality
make fmt                     # Format Go code
make vet                     # Run go vet
make lint                    # Run linter
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
     ┌─────▼─────┐                   ┌─────▼─────┐
     │ PostgreSQL│                   │ PostgreSQL│
     │ (content) │                   │(ingest_rss)│
     └───────────┘                   └───────────┘
```

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
│   │   ├── server/           # HTTP server entry point
│   │   └── worker/           # Background worker entry point
│   ├── internal/
│   │   ├── api/              # HTTP handlers, middleware, DTOs
│   │   ├── repository/       # Database layer
│   │   ├── service/          # Business logic
│   │   ├── processor/        # Content processing (readability, sanitization)
│   │   ├── jobs/             # Background jobs (cleanup)
│   │   └── config/           # Configuration
│   ├── migrations/           # Database migrations
│   ├── Dockerfile            # API server image
│   ├── Dockerfile.worker     # Background worker image
│   └── integration_test.go   # Integration tests
│
├── fetcher/                   # Ingest RSS Service
│   ├── api/                   # OpenAPI specs
│   ├── cmd/
│   │   ├── server/           # HTTP server entry point
│   │   └── worker/           # Background worker entry point
│   ├── internal/
│   │   ├── api/              # HTTP handlers, middleware, DTOs
│   │   ├── repository/       # Database layer
│   │   ├── service/          # Business logic
│   │   ├── fetcher/          # Feed fetching and parsing
│   │   ├── processor/        # Content extraction, update detection
│   │   ├── worker/           # Background workers (outbox, feed polling)
│   │   ├── jobs/             # Scheduled jobs (cleanup, tier management)
│   │   ├── client/           # Content Service HTTP client
│   │   └── config/           # Configuration
│   ├── migrations/           # Database migrations
│   ├── Dockerfile            # API server image
│   ├── Dockerfile.worker     # Background worker image
│   └── integration_test.go   # Integration tests
│
├── api/                       # Shared OpenAPI documentation
├── scripts/                   # Utility scripts
├── docker-compose.yml         # Docker Compose configuration
├── Makefile                   # Build and development commands
├── README.md                  # Main documentation
├── IMPLEMENTATION_PLAN.md     # Implementation details and roadmap
└── INTEGRATION_TESTS.md       # Integration testing guide
```

## Data Models

### Content Service Database (`content_service`)

**contents** table:
```sql
id                UUID PRIMARY KEY
source_url        VARCHAR(2048) NOT NULL UNIQUE
canonical_url     VARCHAR(2048)
title             TEXT NOT NULL
author            VARCHAR(255)
published_at      TIMESTAMP WITH TIME ZONE
content_html      TEXT NOT NULL
content_text      TEXT NOT NULL
excerpt           TEXT
content_length    INTEGER NOT NULL
reading_time_mins INTEGER
source_type       VARCHAR(50) NOT NULL DEFAULT 'manual'
source_feed_id    VARCHAR(255)
content_hash      CHAR(64) NOT NULL
image_url         TEXT
site_name         VARCHAR(255)
created_at        TIMESTAMP WITH TIME ZONE DEFAULT NOW()
updated_at        TIMESTAMP WITH TIME ZONE DEFAULT NOW()
orphaned_at       TIMESTAMP WITH TIME ZONE

-- Indexes
UNIQUE(content_hash, source_feed_id)  -- Deduplication
INDEX(source_feed_id)
INDEX(orphaned_at) WHERE orphaned_at IS NOT NULL
```

**user_contents** table:
```sql
user_id           VARCHAR(255) NOT NULL
content_id        UUID NOT NULL REFERENCES contents(id) ON DELETE CASCADE
status            VARCHAR(50) NOT NULL DEFAULT 'unread'
is_favorite       BOOLEAN NOT NULL DEFAULT FALSE
scroll_position   NUMERIC(5,4) DEFAULT 0.0
notes             TEXT
added_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW()
read_at           TIMESTAMP WITH TIME ZONE
archived_at       TIMESTAMP WITH TIME ZONE

PRIMARY KEY (user_id, content_id)

-- Indexes
INDEX(user_id, status)
INDEX(user_id, is_favorite)
INDEX(user_id, added_at DESC)
INDEX(content_id)  -- For CASCADE DELETE performance

-- Full-text search
GIN INDEX on (to_tsvector('english', title || ' ' || COALESCE(author, '')))
```

**Triggers**:
- **orphaned_content_tracker**: Sets `orphaned_at` when last `user_contents` row is deleted
- **unorphan_content**: Clears `orphaned_at` when new `user_contents` row is added

### Ingest RSS Service Database (`ingest_rss`)

**feeds** table:
```sql
id                VARCHAR(255) PRIMARY KEY
feed_url          VARCHAR(2048) NOT NULL UNIQUE
title             VARCHAR(500)
description       TEXT
site_url          VARCHAR(2048)
polling_tier      VARCHAR(20) NOT NULL DEFAULT 'tier1'
status            VARCHAR(20) NOT NULL DEFAULT 'active'
last_fetched_at   TIMESTAMP WITH TIME ZONE
last_published_at TIMESTAMP WITH TIME ZONE
next_poll_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
error_count       INTEGER DEFAULT 0
consecutive_error_days INTEGER DEFAULT 0
last_error_at     TIMESTAMP WITH TIME ZONE
last_error_message TEXT
http_etag         VARCHAR(255)
http_last_modified VARCHAR(255)
created_at        TIMESTAMP WITH TIME ZONE DEFAULT NOW()
updated_at        TIMESTAMP WITH TIME ZONE DEFAULT NOW()

-- Indexes
INDEX(next_poll_at) WHERE status = 'active'
INDEX(polling_tier)
INDEX(status)
```

**user_feeds** table:
```sql
user_id           VARCHAR(255) NOT NULL
feed_id           VARCHAR(255) NOT NULL REFERENCES feeds(id) ON DELETE CASCADE
subscribed_at     TIMESTAMP WITH TIME ZONE DEFAULT NOW()

PRIMARY KEY (user_id, feed_id)

-- Indexes
INDEX(user_id)
INDEX(feed_id)

-- Trigger: Enforce 100 feed limit per user
```

**feed_items** table:
```sql
id                UUID PRIMARY KEY
feed_id           VARCHAR(255) NOT NULL REFERENCES feeds(id) ON DELETE CASCADE
item_guid         VARCHAR(500) NOT NULL
source_url        VARCHAR(2048) NOT NULL
title             TEXT NOT NULL
author            VARCHAR(255)
published_at      TIMESTAMP WITH TIME ZONE
description       TEXT
content_hash      CHAR(64)
processing_status VARCHAR(20) NOT NULL DEFAULT 'pending'
http_etag         VARCHAR(255)
http_last_modified VARCHAR(255)
last_checked_at   TIMESTAMP WITH TIME ZONE
retry_count       INTEGER DEFAULT 0
error_message     TEXT
created_at        TIMESTAMP WITH TIME ZONE DEFAULT NOW()
updated_at        TIMESTAMP WITH TIME ZONE DEFAULT NOW()

UNIQUE(feed_id, item_guid)

-- Indexes
INDEX(processing_status)
INDEX(feed_id)
INDEX(created_at)
```

**content_outbox** table:
```sql
id                UUID PRIMARY KEY
feed_item_id      UUID REFERENCES feed_items(id) ON DELETE CASCADE
payload           JSONB NOT NULL
user_ids          TEXT[] NOT NULL
delivery_status   VARCHAR(20) NOT NULL DEFAULT 'pending'
content_service_id UUID
retry_count       INTEGER DEFAULT 0
next_retry_at     TIMESTAMP WITH TIME ZONE DEFAULT NOW()
last_error        TEXT
created_at        TIMESTAMP WITH TIME ZONE DEFAULT NOW()
delivered_at      TIMESTAMP WITH TIME ZONE

-- Indexes
INDEX(delivery_status, next_retry_at) WHERE delivery_status = 'pending'
INDEX(created_at)
```

## API Endpoints

### Content Service (port 8083)

**Health Checks**:
```
GET  /health/live                                    → Liveness check
GET  /health/ready                                   → Readiness check (includes DB)
```

**URL Detection (Smart Submission)**:
```
POST /api/v1/content/detect                          → Detect if URL is feed or page
     Body: {"url": "https://example.com"}
     Returns: {"url": "...", "type": "feed|page|unknown", "title": "..."}
     Timeout: 10 seconds
```

**Content Management**:
```
POST   /api/v1/content                               → Create content from HTML/URL
       Body: {"html": "...", "source_url": "...", "title": "...", "source_type": "manual"}

GET    /api/v1/content/{content_id}                  → Get content by ID

PUT    /api/v1/content/{content_id}                  → Update existing content
       Body: {"html": "...", "title": "...", ...}

POST   /api/v1/content/bulk                          → Bulk create/update (max 100)
       Body: [{"html": "...", "source_url": "...", ...}, ...]

POST   /api/v1/content/check-duplicate               → Check for duplicates
       Body: {"items": [{"content_hash": "...", "source_feed_id": "..."}, ...]}
```

**User Content Management**:
```
POST   /api/v1/content/user/{user_id}                → Add URL to user's list
       Body (recommended): {"url": "...", "type": "feed|page", "title": "..."}
       - If type=feed: Subscribes via Ingest RSS service
       - If type=page: Extracts content and adds to reading list
       Body (legacy): {"content_id": "uuid"}

GET    /api/v1/content/user/{user_id}                → List user's contents
       Query: ?status=..., ?is_favorite=true, ?limit=20, ?offset=0

GET    /api/v1/content/user/{user_id}/search         → Full-text search
       Query: ?q=golang, ?limit=20, ?offset=0

PATCH  /api/v1/content/user/{user_id}/{content_id}   → Update user metadata
       Body: {"status": "reading|completed|archived", "is_favorite": true,
              "scroll_position": 0.5, "notes": "..."}

DELETE /api/v1/content/user/{user_id}/{content_id}   → Remove from user's list

POST   /api/v1/content/user/bulk                     → Bulk add to multiple users
       Body: [{"user_id": "...", "url": "...", ...}, ...]
```

### Ingest RSS Service (port 8085)

**Health Checks**:
```
GET  /health/live                                               → Liveness check
GET  /health/ready                                              → Readiness check (includes DB)
```

**Feed Subscriptions**:
```
POST   /api/v1/source/rss/user/{user_id}/subscription          → Subscribe to feed
       Body: {"feed_url": "https://..."}

DELETE /api/v1/source/rss/user/{user_id}/subscription/{feed_id} → Unsubscribe

GET    /api/v1/source/rss/user/{user_id}/subscription          → List user's feeds
```

**Feed Management (Admin)**:
```
GET    /api/v1/source/rss/feed                                 → List all feeds

GET    /api/v1/source/rss/feed/{feed_id}                       → Get feed details

PATCH  /api/v1/source/rss/feed/{feed_id}                       → Enable/disable feed
       Body: {"enabled": true|false}

POST   /api/v1/source/rss/feed/{feed_id}/refresh               → Manually trigger refresh
```

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

Content is deduplicated using: `(content_hash, source_feed_id)`

- Same hash + same feed = duplicate (skip)
- Same hash + different feed = not duplicate (store)
- Multiple users can share the same content with individual metadata

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
- `fetcher/internal/worker/feed_worker.go`
- `fetcher/internal/jobs/tier_management_job.go`

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
- Opens after 5 consecutive failures
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
- **Location**: `fetcher/internal/jobs/tier_management_job.go`

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
# Run all unit tests
make test
go test ./...

# Run with coverage
make test-coverage

# Run integration tests (requires PostgreSQL)
make test-integration

# Run tests for specific service
cd content && go test ./...
cd fetcher && go test ./...

# Run specific package tests
go test ./content/internal/service/...
go test -v ./fetcher/internal/worker/...
```

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

### Content Service

```bash
DATABASE_URL=postgres://user:pass@localhost:5432/content_service?sslmode=disable
SERVER_PORT=8083
LOG_LEVEL=info
MAX_CONTENT_SIZE=5242880       # 5MB limit
ORPHANED_CONTENT_DAYS=90       # Delete after 90 days
```

### Ingest RSS Service

```bash
DATABASE_URL=postgres://user:pass@localhost:5433/ingest_rss?sslmode=disable
SERVER_PORT=8085
CONTENT_SERVICE_URL=http://content-service:8083
LOG_LEVEL=info
MAX_FEEDS_PER_USER=100
FEED_ERROR_THRESHOLD=7         # Days before disabling feed
POLL_INTERVAL_TIER1=1h         # Active feeds
POLL_INTERVAL_TIER2=6h         # Moderate feeds
POLL_INTERVAL_TIER3=24h        # Quiet feeds
```

## Database Migrations

Migrations use `golang-migrate` with numbered SQL files.

### Creating Migrations

```bash
# Create new migration
make migrate-create name=add_user_notes

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
# Apply all pending migrations
make migrate-up

# Rollback last migration
make migrate-down

# Check current migration version
make migrate-status

# Force to specific version
make migrate-force version=5
```

**Important**: Migrations run automatically on service startup in Docker.

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

1. **Create migration** with `make migrate-create name=add_table`
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
3. **Register job** in `cmd/worker/main.go`
4. **Add cron schedule** if needed
5. **Write job tests**

### Debugging

**View logs**:
```bash
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f content-service
docker-compose logs -f ingest-rss

# Filter by level
docker-compose logs -f | grep ERROR
```

**Access database**:
```bash
# Content Service database
docker-compose exec postgres-content psql -U cairn -d content_service

# Ingest RSS database
docker-compose exec postgres-fetcher psql -U cairn -d ingest_rss
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

- **Main README**: `/services/read/README.md` - Comprehensive service documentation
- **Implementation Plan**: `/services/read/IMPLEMENTATION_PLAN.md` - Detailed implementation roadmap
- **Integration Tests**: `/services/read/INTEGRATION_TESTS.md` - Integration testing guide
- **Content Service API**: `/services/read/content/api/openapi.yaml` - OpenAPI specification
- **Ingest RSS API**: `/services/read/fetcher/api/openapi.yaml` - OpenAPI specification
- **Root CLAUDE.md**: `/CLAUDE.md` - Project-wide guidance and conventions

## Status & Roadmap

**Current Status**: ✅ Core functionality complete (Phases 0-5)

**Completed**:
- ✅ Content Service with full CRUD, search, deduplication
- ✅ Ingest RSS Service with feed subscriptions and tiered polling
- ✅ Outbox pattern for reliable content delivery
- ✅ Content update detection via HTTP caching headers
- ✅ Background workers and scheduled jobs
- ✅ Comprehensive test coverage (80%+)

**Remaining Work**:
- 🔲 API documentation (OpenAPI/Swagger UI)
- 🔲 Production observability (metrics, structured logging)
- 🔲 Performance optimization
- 🔲 Security hardening (rate limiting, CORS)

**Future Enhancements**:
- Recommendation engine
- Import/export functionality
- GraphQL API option
- WebSocket support for real-time updates

## Getting Help

For issues or questions:
- Check logs: `docker-compose logs -f`
- Check database state: `docker-compose exec postgres-content psql -U cairn -d content_service`
- Review tests for usage examples
- Consult OpenAPI specifications for API details
- See troubleshooting guide: `/services/read/README.md#troubleshooting`

# Content Service

The Content Service is responsible for storing and serving article content with user-specific metadata.

## Phase 1.1: Database Layer (Completed)

This phase implements the foundational database layer for the Content Service:

### Implemented Features

- **Database Connection Pooling**: Configured PostgreSQL connection pooling with customizable settings
- **Data Models**:
  - `Content`: Stores unique content items (shared across users)
  - `UserContent`: Junction table mapping users to content with user-specific metadata
- **Repository Pattern**:
  - `ContentRepository`: Interface and implementation for content CRUD operations
  - `UserContentRepository`: Interface and implementation for user-content relationship operations
- **Transaction Support**: Helper functions for executing operations within database transactions
- **Database Schema**: Migration files with tables, indexes, and triggers for orphaned content tracking

### Directory Structure

```
content-service/
├── cmd/
│   └── server/
│       └── main.go              # Application entry point
├── internal/
│   ├── database/
│   │   └── connection.go        # Database connection pooling
│   ├── models/
│   │   └── models.go            # Data models (Content, UserContent)
│   └── repository/
│       ├── content.go           # ContentRepository implementation
│       └── user_content.go      # UserContentRepository implementation
├── migrations/
│   ├── 000001_initial_schema.up.sql    # Initial schema migration
│   └── 000001_initial_schema.down.sql  # Schema rollback
├── Dockerfile                   # Docker image configuration
├── .env.example                 # Example environment configuration
├── go.mod                       # Go module definition
└── README.md                    # This file
```

### Configuration

The service is configured via environment variables. See `.env.example` for available options:

- **Server**: `PORT`
- **Database**: `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSL_MODE`
- **Connection Pool**: `DB_MAX_OPEN_CONNS`, `DB_MAX_IDLE_CONNS`, `DB_CONN_MAX_LIFETIME`, `DB_CONN_MAX_IDLE_TIME`

### Building and Running

#### Local Development

```bash
# Install dependencies
go mod download

# Build the service
go build -o bin/content-service ./cmd/server

# Run the service
./bin/content-service
```

#### Docker

```bash
# Build Docker image
docker build -t content-service .

# Run container
docker run -p 8080:8080 \
  -e DB_HOST=localhost \
  -e DB_PORT=5433 \
  -e DB_USER=cairn_content \
  -e DB_PASSWORD=cairn_content_pass \
  -e DB_NAME=cairn_content \
  content-service
```

#### Docker Compose

From the repository root:

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f content-service

# Stop services
docker-compose down
```

### Database Migrations

Run migrations from the repository root:

```bash
# Apply all pending migrations
make migrate-up-content

# Rollback last migration
make migrate-down-content

# Create new migration
make migrate-create-content name=add_new_feature

# Check migration status
make migrate-status-content
```

### Health Check

The service exposes a health check endpoint:

```bash
curl http://localhost:8080/health
```

Response:
```json
{
  "status": "ok",
  "service": "content-service",
  "message": "Phase 1.1: Database layer complete"
}
```

### Database Schema

#### Contents Table

Stores unique content items (shared across users):

- Content hash for deduplication (SHA-256)
- Cleaned HTML (max 5MB)
- Original and canonical URLs
- Metadata (title, author, published_at, description, image_urls)
- Source information (source_type, source_feed_id)
- Timestamps (created_at, updated_at, orphaned_at)

#### User-Contents Table

Junction table mapping users to content with user-specific metadata:

- User ID (external reference)
- Content ID (foreign key to contents)
- Status (unread, read, archived)
- Scroll position (character offset)
- Is favorite (boolean)
- Timestamps (added_at, updated_at)

#### Triggers

- **mark_content_orphaned**: Automatically marks content as orphaned when the last user-content relationship is deleted
- **clear_orphaned_status**: Clears the orphaned status when content is re-saved by a user

## URL Detection Feature

The Content Service now includes smart URL detection and submission capabilities:

### URL Detection Endpoint

**POST /api/v1/content/detect**

Determines if a URL is an RSS/Atom feed or a web page:

```bash
curl -X POST http://localhost:8080/api/v1/content/detect \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com/feed.xml"}'
```

Response:
```json
{
  "url": "https://example.com/feed.xml",
  "type": "feed",
  "title": "Example Blog"
}
```

Possible types: `feed`, `page`, `unknown` (on timeout/error)

### Smart URL Submission

**POST /api/v1/content/user/{user_id}**

Automatically routes URLs based on type:
- **feed**: Subscribes user to RSS feed via Ingest RSS service
- **page**: Extracts and saves article content

```bash
# Auto-detect and add
curl -X POST http://localhost:8080/api/v1/content/user/{user_id} \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com/article"}'

# With type hint (from detection)
curl -X POST http://localhost:8080/api/v1/content/user/{user_id} \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com/feed.xml",
    "type": "feed",
    "title": "Example Blog"
  }'
```

Feed response:
```json
{
  "type": "feed",
  "feed_id": "...",
  "subscription": {
    "id": "...",
    "user_id": "...",
    "feed_id": "...",
    "feed_url": "https://example.com/feed.xml",
    "title": "Example Blog",
    "subscribed_at": "2025-01-04T10:00:00Z"
  }
}
```

Page response:
```json
{
  "type": "page",
  "content": {
    "user_id": "...",
    "content_id": "...",
    "status": "unread",
    "content": {
      "title": "Article Title",
      "cleaned_html": "...",
      ...
    }
  }
}
```

### Implementation Details

- **Timeout**: 10 seconds for URL detection (non-blocking)
- **Feed Detection**: Uses gofeed parser for RSS/Atom feeds
- **Page Extraction**: go-readability for article content
- **Feed Management**: Integrates with Ingest RSS service for subscriptions
- **Error Handling**: Returns specific error codes for duplicate subscriptions, feed limits, etc.

### Mobile Integration

The mobile app uses these endpoints to provide a seamless "Add" experience:
1. User enters URL
2. App calls `/detect` endpoint (non-blocking, 10s timeout)
3. UI updates button text ("Add" → "Add Feed" if feed detected)
4. User submits with detected type hint
5. Backend routes to appropriate handler

See `apps/mobile/src/components/AddLinkModal.tsx` for implementation.

### Next Steps

- **Phase 1.2**: Content processing (readability, sanitization, deduplication) ✅ Complete
- **Phase 1.3**: REST API - Basic operations ✅ Complete
- **Phase 1.4**: REST API - Bulk operations ✅ Complete
- **URL Detection**: Smart URL submission and feed detection ✅ Complete

## Development

### Dependencies

- Go 1.21+
- PostgreSQL 15+
- golang-migrate (for migrations)

### Testing

```bash
# Run tests (when available)
go test ./...

# Run tests with coverage
go test -cover ./...
```

## License

Proprietary

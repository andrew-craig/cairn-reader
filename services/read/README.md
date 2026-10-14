# Cairn Backend

A read-it-later backend built with Go, PostgreSQL, and Docker. Cairn enables users to subscribe to RSS feeds and email senders, automatically receive new content, save articles, and manage two lists: **Feed** (skim) and **Reads** (read or triage).

## Overview

Cairn is a microservices-based backend that provides:

- **RSS Feed Management**: Subscribe to RSS/Atom feeds with intelligent tiered polling
- **Email Ingestion**: Turn newsletters into articles (see [email/CLAUDE.md](/services/read/email/CLAUDE.md))
- **Feed and Reads Lists**: The user picks the list per source (new email senders default to Reads, saved links always go to Reads). Re-routing a source affects new items only, and non-favorited Feed items are deleted after 30 days
- **Content Storage**: Store and serve article content with readability extraction and HTML sanitization
- **User Reading Lists**: Personal reading lists with status tracking, favorites, and scroll position
- **Content Deduplication**: Automatic deduplication to avoid storing duplicate articles
- **Content Updates**: Detect and sync article updates using HTTP caching headers
- **Search**: Full-text search across article titles and authors
- **Reliable Delivery**: Outbox pattern ensures content delivery even during service failures

## Architecture

Cairn consists of three microservices (Content, Ingest RSS, Email Ingest) plus background workers:

```
┌─────────────────────┐         ┌──────────────────────┐
│  Content Service    │         │   Ingest RSS Service │
│                     │         │    (ingest_rss)      │
│  - Content Storage  │◄────────│  - Feed Polling      │
│  - User Lists       │  REST   │  - Content Extraction│
│  - Search           │   API   │  - Subscription Mgmt │
│  - Metadata         │         │  - Outbox Delivery   │
└──────────┬──────────┘         └──────────┬───────────┘
           │   ▲                           │
           │   └──── Email Ingest ─────────┤  (same delivery path;
     ┌─────▼─────┐   Service           ┌───▼───────┐   see email/CLAUDE.md)
     │ PostgreSQL│                     │ PostgreSQL│
     │ (Content) │                     │(rss_fetcher)│
     └───────────┘                     └───────────┘
```

### Services

#### Content Service (`content/`, [README](content/README.md))

Manages article content, user lists and Feed/Reads routing:

- **Port**: 8080 in the container, 8083 on the dev host
- **Database**: PostgreSQL (`content_service`)
- **Responsibilities**:
  - Store and serve cleaned article content
  - Manage user-content relationships (status, favorites, scroll position, list)
  - Own the Feed/Reads routing (`source_routes`, `user_contents.list`)
  - Provide search functionality with PostgreSQL full-text search
  - Handle content deduplication by hash and feed (RSS) or URL (email/manual)
  - Support bulk operations for Ingest RSS and Email Ingest
  - Aggregate and proxy subscription management for RSS and email

#### Ingest RSS Service (`fetcher/`, binary `ingest_rss`)

Manages RSS feed subscriptions and content delivery:

- **Port**: 8081 in the container, 8085 on the dev host
- **Database**: PostgreSQL (`rss_fetcher_service`)
- **Responsibilities**:
  - Subscribe/unsubscribe users to RSS feeds
  - Poll feeds using tiered strategy (hourly, 6-hourly, daily)
  - Extract and process article content
  - Detect content updates via ETag/Last-Modified headers
  - Deliver content to Content Service via outbox pattern
  - Auto-disable feeds after 7 consecutive error days

#### Email Ingest Service (`email/`)

Email-to-article ingestion pipeline. See [email/CLAUDE.md](/services/read/email/CLAUDE.md).

### Key Design Principles

1. **Service Isolation**: Each service has its own database; communication is REST-only
2. **Reliability**: Outbox pattern ensures content delivery survives service failures
3. **Efficiency**: Tiered polling reduces load on inactive feeds
4. **Resilience**: Circuit breaker protects against cascading failures
5. **Deduplication**: Content hash prevents storing duplicate articles
6. **Update Detection**: HTTP caching headers minimize unnecessary re-fetching

## Prerequisites

- **Docker** 20.10+
- **Docker Compose** 2.0+
- **Go** 1.25+ (for local development; see `go.mod`)
- **Make** (optional, for convenience commands)

## Quick Start

### 1. Clone the Repository

```bash
git clone https://github.com/andrew-craig/cairn-reader.git
cd cairn-reader
```

### 2. Start Services with Docker Compose

There is no `docker-compose.yml` under `services/read/`; the dev stack lives in `infrastructure/docker/dev/`.

```bash
cd infrastructure/docker/dev
docker compose up -d
```

This will start (among other services):
- Content Service (http://localhost:8083)
- Ingest RSS Service (http://localhost:8085)
- Email Ingest Service (http://localhost:8089)
- One consolidated PostgreSQL container with a logical database per service
- Background workers (`content-worker`, `ingest-rss-worker`, email worker)

### 3. Verify Services are Running

```bash
curl http://localhost:8083/health/ready
curl http://localhost:8085/health/ready
```

### 4. Run Database Migrations

Migrations run automatically on service startup. To run manually from `services/read/`
(per-service targets; the Makefile targets the standalone `localhost:5433`/`5434` databases):

```bash
make migrate-up
make migrate-status-content
make migrate-status-fetcher
```

## Usage Examples

All user-facing calls go through the Content Service with a JWT from the User Service.
The Ingest RSS routes (`/api/v1/source/rss`) are internal and require `X-Internal-API-Key`.

### Subscribe to a Feed (and choose Feed or Reads)

```bash
curl -X POST http://localhost:8083/api/v1/content/user/550e8400-e29b-41d4-a716-446655440000 \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "url": "https://blog.golang.org/feed.atom",
    "type": "feed",
    "list": "feed"
  }'
```

### Add Content to User's List

```bash
curl -X POST http://localhost:8083/api/v1/content/user/550e8400-e29b-41d4-a716-446655440000 \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "url": "https://example.com/article",
    "type": "page"
  }'
```

### List User's Contents

```bash
curl -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  "http://localhost:8083/api/v1/content/user/550e8400-e29b-41d4-a716-446655440000?list=reads&limit=20"
```

### Search User's Contents

```bash
curl -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  "http://localhost:8083/api/v1/content/user/550e8400-e29b-41d4-a716-446655440000/search?q=golang"
```

### Update Reading Status

```bash
curl -X PATCH http://localhost:8083/api/v1/content/user/550e8400-e29b-41d4-a716-446655440000/123e4567-e89b-12d3-a456-426614174001 \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "status": "completed",
    "is_favorite": true
  }'
```

## API Documentation

Comprehensive API documentation is available in OpenAPI 3.0 format:

- **Content Service API**: [content/api/openapi.yaml](content/api/openapi.yaml)
- **Ingest RSS Service API**: [fetcher/api/openapi.yaml](fetcher/api/openapi.yaml)
- **Email Ingest Service API**: [email/api/openapi.yaml](email/api/openapi.yaml)

You can view these specifications using any OpenAPI viewer like [Swagger Editor](https://editor.swagger.io/).

## Development

### Project Structure

See the Directory Structure section of [CLAUDE.md](/services/read/CLAUDE.md). In short:
`content/`, `fetcher/` and `email/` each hold `cmd/` (server and worker entry points), `internal/`,
`migrations/`, `api/openapi.yaml` and Dockerfiles; `scripts/` and the `Makefile` live at this level.

### Building Locally

```bash
make build-all        # content, content-worker, fetcher (ingest_rss), fetcher-worker, email, email-worker -> bin/
make build-content    # one service (also build-fetcher, build-email, build-*-worker)
```

### Running Tests

```bash
# Run all unit tests (content, fetcher, email)
make test

# Run tests for a specific service
make test-content     # also test-fetcher, test-email

# Integration tests use the `integration` build tag and need PostgreSQL (TEST_DB_* env vars)
cd content && go test -tags=integration ./...
cd fetcher && go test -tags=integration ./...

# Coverage (no make target)
cd content && go test -coverprofile=coverage.out ./...
```

### Database Migrations

Per-service targets (content and fetcher; email has its own migrations):

```bash
make migrate-up                           # content + fetcher
make migrate-down                         # roll back last migration in each
make migrate-create-content name=add_new_feature   # or migrate-create-fetcher
make migrate-status-content               # or migrate-status-fetcher
```

## Configuration

Services are configured via environment variables. See:

- [../../docs/CONFIGURATION.md](../../docs/CONFIGURATION.md) - Complete configuration reference
- [content/.env.example](content/.env.example), [fetcher/.env.example](fetcher/.env.example) - Example environment files
- The Environment Variables section of [CLAUDE.md](/services/read/CLAUDE.md) - per-service and worker variables

Key environment variables:

```bash
# Content Service
PORT=8080
DB_HOST=cairn-db  DB_PORT=5432  DB_USER=...  DB_PASSWORD=...  DB_NAME=content_service
INTERNAL_API_KEY=...
LOG_LEVEL=info

# Ingest RSS Service
PORT=8081
DB_HOST=cairn-db  DB_PORT=5432  DB_USER=...  DB_PASSWORD=...  DB_NAME=rss_fetcher_service
INTERNAL_API_KEY=...
# Ingest RSS worker also needs CONTENT_SERVICE_URL=http://content-service:8080
```

## Deployment

For detailed deployment instructions, see [../../docs/DEPLOYMENT.md](../../docs/DEPLOYMENT.md).

### Docker Compose (Recommended for Single-Server)

Run from `infrastructure/docker/dev/` (dev) or see the deployment docs for production:

```bash
docker compose up -d      # Start all services
docker compose logs -f    # View logs
docker compose down       # Stop all services
docker compose down -v    # Stop and remove volumes (WARNING: deletes data)
```

### Health Checks

Each service provides health check endpoints (dev host ports shown):

```bash
# Liveness - is the service running?
curl http://localhost:8083/health/live
curl http://localhost:8085/health/live

# Readiness - is the service ready (includes DB check)?
curl http://localhost:8083/health/ready
curl http://localhost:8085/health/ready
```

## Background Jobs & Workers

### Content Service

- **Cleanup Job** (`content-worker`): Runs daily at 2 AM (`CLEANUP_CRON`); first deletes non-favorited Feed items older than 30 days, then content orphaned for 90+ days

### Ingest RSS Service

- **Feed Polling Worker**: Continuously polls feeds based on tiered schedule
  - Tier 1 (Active): Every 1 hour
  - Tier 2 (Moderate): Every 6 hours
  - Tier 3 (Quiet): Every 24 hours
- **Tier Management Job**: Runs daily, adjusts feed tiers based on activity
- **Content Extraction Worker**: Processes pending feed items, extracts full content
- **Outbox Delivery Worker**: Delivers content to Content Service with retry logic
- **Outbox Cleanup Job**: Runs daily at 3 AM, removes delivered entries older than 7 days
- **Feed Items Cleanup Job**: Runs daily at 4 AM, removes completed items older than 7 days and failed items older than 30 days

## Features

### Content Features

- ✅ HTML readability extraction using [go-readability](https://github.com/go-shiori/go-readability)
- ✅ HTML sanitization using [bluemonday](https://github.com/microcosm-cc/bluemonday)
- ✅ Content deduplication by hash + feed ID (RSS) or hash + URL (email/manual)
- ✅ Feed and Reads lists with per-source routing and 30-day Feed expiry
- ✅ Content size limit (5MB)
- ✅ URL canonicalization
- ✅ Full-text search (PostgreSQL GIN index)
- ✅ Cursor-based pagination (20 items/page by default, max 100)
- ✅ Content update detection via HTTP caching headers

### Feed Features

- ✅ RSS/Atom feed parsing using [gofeed](https://github.com/mmcdole/gofeed)
- ✅ 100 feed limit per user
- ✅ Tiered polling strategy (active/moderate/quiet)
- ✅ Auto-disable after 7 consecutive error days
- ✅ Feed metadata extraction
- ✅ Duplicate prevention per feed

### User Features

- ✅ Reading status tracking (unread/reading/completed/archived)
- ✅ Scroll position saving
- ✅ Favorites
- ✅ Per-user filtering and search
- ✅ Individual user-content metadata while sharing underlying content

### Reliability Features

- ✅ Outbox pattern for guaranteed content delivery
- ✅ Circuit breaker for Content Service calls
- ✅ Exponential backoff retry logic
- ✅ Graceful degradation
- ✅ Database triggers for orphaned content tracking

## Monitoring & Logging

### Logs

```bash
# View all logs
docker compose logs -f

# View specific service logs
docker compose logs -f content-service
docker compose logs -f ingest-rss

# Filter logs by level
docker compose logs -f | grep ERROR
```

### Database Access

```bash
# Connect to Content Service database
docker compose exec cairn-db psql -U cairn_content -d content_service

# Connect to Ingest RSS database
docker compose exec cairn-db psql -U cairn_rss -d rss_fetcher_service
```

## Troubleshooting

For common issues and solutions, see [../../docs/TROUBLESHOOTING.md](../../docs/TROUBLESHOOTING.md).

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Run tests (`make test`)
5. Commit your changes (`git commit -m 'Add amazing feature'`)
6. Push to the branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

## Testing

Cairn has comprehensive test coverage:

- **Unit Tests**: Service layer, repository layer, handlers, middleware
- **Integration Tests**: End-to-end API and database flows
- **Test Coverage**: 80%+ across critical paths

See [../../docs/TESTING.md](../../docs/TESTING.md) for details on running integration tests.

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Technology Stack

### Core

- **Language**: Go 1.25+
- **Database**: PostgreSQL 18
- **Containerization**: Docker & Docker Compose

### Libraries

#### Content Service
- `github.com/go-shiori/go-readability` - HTML readability extraction
- `github.com/microcosm-cc/bluemonday` - HTML sanitization
- `github.com/lib/pq` - PostgreSQL driver
- `github.com/go-chi/chi` - HTTP routing
- `github.com/golang-migrate/migrate/v4` - Database migrations

#### Ingest RSS Service
- `github.com/mmcdole/gofeed` - RSS/Atom parsing
- `github.com/lib/pq` - PostgreSQL driver
- `github.com/go-chi/chi` - HTTP routing
- `github.com/robfig/cron/v3` - Job scheduling
- `github.com/sony/gobreaker` - Circuit breaker
- `github.com/golang-migrate/migrate/v4` - Database migrations

#### Testing
- `github.com/stretchr/testify` - Testing framework
- `github.com/DATA-DOG/go-sqlmock` - SQL mocking

## Roadmap

Future enhancements planned:

- [ ] Prometheus metrics endpoint
- [ ] Rate limiting middleware
- [ ] Kubernetes deployment manifests
- [ ] GraphQL API option
- [ ] WebSocket support for real-time updates
- [ ] Import/export functionality

## Support

For issues, questions, or contributions:

- **Issues**: [GitHub Issues](https://github.com/andrew-craig/cairn-reader/services/read/issues)
- **Documentation**: See `/docs` directory
- **Architecture**: [../../docs/ARCHITECTURE.md](../../docs/ARCHITECTURE.md)

## Acknowledgments

Built with excellent open-source libraries:
- [go-readability](https://github.com/go-shiori/go-readability) - Content extraction
- [gofeed](https://github.com/mmcdole/gofeed) - RSS/Atom parsing
- [bluemonday](https://github.com/microcosm-cc/bluemonday) - HTML sanitization
- [Chi](https://github.com/go-chi/chi) - HTTP routing
- [golang-migrate](https://github.com/golang-migrate/migrate) - Database migrations

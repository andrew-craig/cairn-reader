# Cairn Backend

A modern, scalable read-it-later backend system built with Go, PostgreSQL, and Docker. Cairn enables users to subscribe to RSS feeds, automatically receive new content, save articles, and manage their reading list.

## Overview

Cairn is a microservices-based backend that provides:

- **RSS Feed Management**: Subscribe to RSS/Atom feeds with intelligent tiered polling
- **Content Storage**: Store and serve article content with readability extraction and HTML sanitization
- **User Reading Lists**: Personal reading lists with status tracking, favorites, and scroll position
- **Content Deduplication**: Automatic deduplication to avoid storing duplicate articles
- **Content Updates**: Detect and sync article updates using HTTP caching headers
- **Search**: Full-text search across article titles and authors
- **Reliable Delivery**: Outbox pattern ensures content delivery even during service failures

## Architecture

Cairn consists of two main microservices:

```
┌─────────────────────┐         ┌──────────────────────┐
│  Content Service    │         │   Ingest RSS Service │
│                     │         │    (ingest_rss)      │
│  - Content Storage  │◄────────│  - Feed Polling      │
│  - User Lists       │  REST   │  - Content Extraction│
│  - Search           │   API   │  - Subscription Mgmt │
│  - Metadata         │         │  - Outbox Delivery   │
└──────────┬──────────┘         └──────────┬───────────┘
           │                               │
           │                               │
     ┌─────▼─────┐                   ┌─────▼─────┐
     │ PostgreSQL│                   │ PostgreSQL│
     │ (Content) │                   │(ingest_rss)│
     └───────────┘                   └───────────┘
```

### Services

#### Content Service (`services/content-service/`)

Manages article content and user reading lists:

- **Port**: 8080
- **Database**: PostgreSQL (content_service)
- **Responsibilities**:
  - Store and serve cleaned article content
  - Manage user-content relationships (status, favorites, scroll position)
  - Provide search functionality with PostgreSQL full-text search
  - Handle content deduplication by hash and feed ID
  - Support bulk operations for RSS Fetcher integration

#### Ingest RSS Service (ingest_rss) (`services/rss-fetcher-service/`)

Manages RSS feed subscriptions and content delivery:

- **Port**: 8081
- **Database**: PostgreSQL (ingest_rss)
- **Responsibilities**:
  - Subscribe/unsubscribe users to RSS feeds
  - Poll feeds using tiered strategy (hourly, 6-hourly, daily)
  - Extract and process article content
  - Detect content updates via ETag/Last-Modified headers
  - Deliver content to Content Service via outbox pattern
  - Auto-disable feeds after 7 consecutive error days

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
- **Go** 1.21+ (for local development)
- **Make** (optional, for convenience commands)

## Quick Start

### 1. Clone the Repository

```bash
git clone https://github.com/cairn-app/cairn-reader.git
cd cairn/services/read
```

### 2. Start Services with Docker Compose

```bash
docker-compose up -d
```

This will start:
- Content Service (http://localhost:8080)
- Ingest RSS Service (http://localhost:8081)
- PostgreSQL databases for both services
- Background workers for feed polling and content delivery

### 3. Verify Services are Running

```bash
# Check Content Service health
curl http://localhost:8080/health/ready

# Check Ingest RSS Service health
curl http://localhost:8081/health/ready
```

### 4. Run Database Migrations

Migrations run automatically on service startup. To run manually:

```bash
# Apply all pending migrations
make migrate-up

# Check migration status
make migrate-status
```

## Usage Examples

### Subscribe to a Feed

```bash
curl -X POST http://localhost:8085/api/v1/source/rss/user/550e8400-e29b-41d4-a716-446655440000/subscription \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "feed_url": "https://blog.golang.org/feed.atom"
  }'
```

### Add Content to User's List

```bash
curl -X POST http://localhost:8083/api/v1/content/user/550e8400-e29b-41d4-a716-446655440000 \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "url": "https://example.com/article",
    "source_type": "manual"
  }'
```

### List User's Contents

```bash
curl -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  "http://localhost:8083/api/v1/content/user/550e8400-e29b-41d4-a716-446655440000?limit=20"
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

- **Content Service API**: [services/content-service/api/openapi.yaml](services/content-service/api/openapi.yaml)
- **Ingest RSS Service API**: [services/rss-fetcher-service/api/openapi.yaml](services/rss-fetcher-service/api/openapi.yaml)

You can view these specifications using any OpenAPI viewer like [Swagger Editor](https://editor.swagger.io/).

## Development

### Project Structure

```
cairn/services/read/
├── services/
│   ├── content-service/           # Content Service
│   │   ├── api/                   # OpenAPI specs
│   │   ├── cmd/                   # Entry points
│   │   ├── internal/              # Internal packages
│   │   │   ├── api/              # HTTP handlers & middleware
│   │   │   ├── repository/       # Database layer
│   │   │   ├── service/          # Business logic
│   │   │   ├── processor/        # Content processing
│   │   │   └── jobs/             # Background jobs
│   │   ├── migrations/           # Database migrations
│   │   └── Dockerfile
│   │
│   └── rss-fetcher-service/      # Ingest RSS Service (ingest_rss)
│       ├── api/                   # OpenAPI specs
│       ├── cmd/                   # Entry points
│       ├── internal/              # Internal packages
│       │   ├── api/              # HTTP handlers & middleware
│       │   ├── repository/       # Database layer
│       │   ├── service/          # Business logic
│       │   ├── fetcher/          # Feed fetching
│       │   ├── processor/        # Content processing
│       │   ├── worker/           # Background workers
│       │   ├── jobs/             # Scheduled jobs
│       │   └── client/           # Content Service client
│       ├── migrations/           # Database migrations
│       └── Dockerfile
│
├── docs/                          # Documentation
├── scripts/                       # Utility scripts
├── docker-compose.yml            # Docker Compose config
├── Makefile                      # Build commands
└── README.md                     # This file
```

### Building Locally

```bash
# Build Content Service
cd services/content-service
go build -o bin/content-service ./cmd/server

# Build Ingest RSS Service
cd services/rss-fetcher-service
go build -o bin/ingest-rss ./cmd/server
```

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run integration tests (requires PostgreSQL)
make test-integration

# Run tests for a specific service
cd services/content-service
go test ./...
```

### Database Migrations

```bash
# Apply all pending migrations
make migrate-up

# Rollback last migration
make migrate-down

# Create new migration
make migrate-create name=add_new_feature

# Check migration status
make migrate-status
```

## Configuration

Services are configured via environment variables. See:

- [../../docs/CONFIGURATION.md](../../docs/CONFIGURATION.md) - Complete configuration reference
- [.env.example](services/content-service/.env.example) - Example environment files

Key environment variables:

```bash
# Content Service
DATABASE_URL=postgres://user:pass@localhost:5432/content_service?sslmode=disable
SERVER_PORT=8080
LOG_LEVEL=info

# Ingest RSS Service
DATABASE_URL=postgres://user:pass@localhost:5433/ingest_rss?sslmode=disable
SERVER_PORT=8081
CONTENT_SERVICE_URL=http://content-service:8080
LOG_LEVEL=info
```

## Deployment

For detailed deployment instructions, see [../../docs/DEPLOYMENT.md](../../docs/DEPLOYMENT.md).

### Docker Compose (Recommended for Single-Server)

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop all services
docker-compose down

# Stop and remove volumes (WARNING: deletes data)
docker-compose down -v
```

### Health Checks

Both services provide health check endpoints:

```bash
# Liveness - is the service running?
curl http://localhost:8080/health/live
curl http://localhost:8081/health/live

# Readiness - is the service ready (includes DB check)?
curl http://localhost:8080/health/ready
curl http://localhost:8081/health/ready
```

## Background Jobs & Workers

### Content Service

- **Orphaned Content Cleanup**: Runs daily at 2 AM, deletes content orphaned for 90+ days

### Ingest RSS Service

- **Feed Polling Worker**: Continuously polls feeds based on tiered schedule
  - Tier 1 (Active): Every 1 hour
  - Tier 2 (Moderate): Every 6 hours
  - Tier 3 (Quiet): Every 24 hours
- **Tier Management Job**: Runs daily, adjusts feed tiers based on activity
- **Content Extraction Worker**: Processes pending feed items, extracts full content
- **Outbox Delivery Worker**: Delivers content to Content Service with retry logic
- **Outbox Cleanup Job**: Runs daily at 3 AM, removes delivered entries older than 7 days
- **Feed Items Cleanup Job**: Runs daily at 4 AM, removes old completed/failed items

## Features

### Content Features

- ✅ HTML readability extraction using [go-readability](https://github.com/go-shiori/go-readability)
- ✅ HTML sanitization using [bluemonday](https://github.com/microcosm-cc/bluemonday)
- ✅ Content deduplication by hash + feed ID
- ✅ Content size limit (5MB)
- ✅ URL canonicalization
- ✅ Full-text search (PostgreSQL GIN index)
- ✅ Cursor-based pagination (20 items/page)
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
docker-compose logs -f

# View specific service logs
docker-compose logs -f content-service
docker-compose logs -f ingest-rss

# Filter logs by level
docker-compose logs -f | grep ERROR
```

### Database Access

```bash
# Connect to Content Service database
docker-compose exec postgres-content psql -U cairn -d content_service

# Connect to Ingest RSS database
docker-compose exec postgres-fetcher psql -U cairn -d ingest_rss
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

See [INTEGRATION_TESTS.md](INTEGRATION_TESTS.md) for details on running integration tests.

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Technology Stack

### Core

- **Language**: Go 1.21+
- **Database**: PostgreSQL 15+
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

- [ ] Authentication & authorization (JWT)
- [ ] Prometheus metrics endpoint
- [ ] Structured logging with Zap
- [ ] Rate limiting middleware
- [ ] Kubernetes deployment manifests
- [ ] GraphQL API option
- [ ] WebSocket support for real-time updates
- [ ] Recommendation engine
- [ ] Import/export functionality

## Support

For issues, questions, or contributions:

- **Issues**: [GitHub Issues](https://github.com/cairn-app/cairn-reader/services/read/issues)
- **Documentation**: See `/docs` directory
- **Architecture**: [../../docs/ARCHITECTURE.md](../../docs/ARCHITECTURE.md)

## Acknowledgments

Built with excellent open-source libraries:
- [go-readability](https://github.com/go-shiori/go-readability) - Content extraction
- [gofeed](https://github.com/mmcdole/gofeed) - RSS/Atom parsing
- [bluemonday](https://github.com/microcosm-cc/bluemonday) - HTML sanitization
- [Chi](https://github.com/go-chi/chi) - HTTP routing
- [golang-migrate](https://github.com/golang-migrate/migrate) - Database migrations

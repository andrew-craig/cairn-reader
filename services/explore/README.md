# Cairn Explore - RSS Fetcher and Recommendation Engine

A system for fetching RSS feeds and recommending content to users. These services provide the Explore functionality of the [Cairn App](https://github.com/cairn-app/cairn-reader).

## Architecture

This project consists of two microservices:

### 1. Explore Fetcher (explore_fetcher)
- Discovers and fetches content from RSS feeds
- Parses feed items and extracts relevant metadata
- Sends discovered content to the Recommender service via HTTP API

### 2. Explore Recommender (explore_recommender)
- Receives content from the Fetcher service
- Stores content in the PostgreSQL database
- Implements recommendation algorithm to suggest the next 5 articles for users
- Exposes API for retrieving recommendations

## Getting Started

### Prerequisites
- Go 1.23+
- Docker and Docker Compose

### Running Locally

```bash
# Start all services
docker compose up --build

# The services will be available at:
# - Fetcher: http://localhost:8080
# - Recommender: http://localhost:8081
```

### Development

```bash
# Run fetcher service
cd fetcher
go run cmd/fetcher/main.go

# Run recommender service
cd recommender
go run cmd/recommender/main.go
```

## Testing

The Explore services include both unit tests and integration tests using [Testcontainers](https://testcontainers.com/) for PostgreSQL.

### Running Tests

```bash
# Run all tests (unit + integration)
# Requires Docker to be running for integration tests
go test ./... -v

# Run only unit tests (skip integration tests)
go test ./... -v -short

# Run specific integration tests
go test -tags=integration -v ./recommender/internal/db/

# Run unit tests in a specific package
go test -v ./recommender/internal/recommend/
```

### Test Types

#### Unit Tests
- Located alongside source code files (e.g., `engine_test.go`)
- Test business logic in isolation using mocks
- Run quickly without external dependencies
- Use `-short` flag to skip integration tests: `go test -short ./...`

#### Integration Tests
- Located in files with `_integration_test.go` suffix
- Use `//go:build integration` build tag
- Automatically spin up PostgreSQL containers using Testcontainers
- Run database migrations automatically
- Clean up containers after tests complete
- **Require Docker to be running**

### Integration Test Setup

Integration tests use Testcontainers to automatically:
1. Start a PostgreSQL 16 container
2. Run all database migrations
3. Provide a clean database for each test
4. Terminate the container after tests complete

No manual database setup required! Just ensure Docker is running.

**Example integration test:**

```go
func TestIntegration_Something(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    pool, cleanup := testutil.SetupTestDB(t)
    defer cleanup()

    repo := db.NewArticleRepository(pool)
    // ... test code
}
```

### CI/CD Considerations

Integration tests require Docker to run. In CI/CD pipelines:
- Use Docker-in-Docker or provide a Docker socket
- Or skip integration tests with: `go test -short ./...`
- GitHub Actions example:

```yaml
- name: Run tests
  run: |
    # Unit tests only (no Docker required)
    go test -short ./...

    # Or with Docker service for integration tests
    go test ./...
```

## Project Structure

```
.
├── fetcher/              # RSS fetcher service
│   ├── cmd/
│   │   └── fetcher/
│   │       └── main.go
│   ├── internal/
│   │   ├── fetcher/     # Core fetching logic
│   │   └── client/      # HTTP client for recommender API
│   └── Dockerfile
├── recommender/          # Recommendation service
│   ├── cmd/
│   │   └── recommender/
│   │       └── main.go
│   ├── internal/
│   │   ├── api/         # HTTP handlers
│   │   ├── db/          # Database access layer
│   │   └── recommend/   # Recommendation algorithm
│   └── Dockerfile
├── pkg/                  # Shared packages
│   └── models/          # Shared data models
├── docker-compose.yml
└── go.mod
```

## API Endpoints

### Explore Fetcher (explore_fetcher, port 8080)

#### Health Checks
- `GET /health/live` - Liveness check
- `GET /health/ready` - Readiness check (includes database connectivity)

#### Feed Management
- `POST /api/v1/explore/feed/fetch` - Manually trigger feed fetch
- `POST /api/v1/explore/feed/sync` - Sync feeds from Kagi Small Web
- `GET /api/v1/explore/feed/stats` - Get feed statistics

**Example API Calls:**

```bash
# Health checks
curl http://localhost:8080/health/live
curl http://localhost:8080/health/ready

# Manual feed fetch
curl -X POST http://localhost:8080/api/v1/explore/feed/fetch

# Feed statistics
curl http://localhost:8080/api/v1/explore/feed/stats

# Sync feeds
curl -X POST http://localhost:8080/api/v1/explore/feed/sync
```

### Explore Recommender (explore_recommender, port 8081)

#### Health Checks
- `GET /health/live` - Liveness check
- `GET /health/ready` - Readiness check (includes database connectivity)

#### Article Management
- `POST /api/v1/explore/article` - Submit article (from fetcher)

#### Recommendations
- `GET /api/v1/explore/recommendation/{user_id}` - Get 5 recommendations (requires auth)

#### User Interactions
- `POST /api/v1/explore/article/{article_id}/read` - Mark article as read (requires auth)
- `POST /api/v1/explore/article/{article_id}/vote` - Vote on article (requires auth)
- `DELETE /api/v1/explore/article/{article_id}/vote` - Remove vote (requires auth)
- `GET /api/v1/explore/article/{article_id}/vote` - Get vote counts (requires auth)

**Example API Calls:**

```bash
# Health checks
curl http://localhost:8081/health/live
curl http://localhost:8081/health/ready

# Get recommendations (requires auth)
curl -H "Authorization: Bearer <JWT>" \
  http://localhost:8081/api/v1/explore/recommendation/user123

# Mark article as read (requires auth)
curl -X POST \
  -H "Authorization: Bearer <JWT>" \
  http://localhost:8081/api/v1/explore/article/{article_id}/read

# Vote on article (requires auth)
curl -X POST \
  -H "Authorization: Bearer <JWT>" \
  -H "Content-Type: application/json" \
  -d '{"vote_type":"upvote"}' \
  http://localhost:8081/api/v1/explore/article/{article_id}/vote

# Remove vote (requires auth)
curl -X DELETE \
  -H "Authorization: Bearer <JWT>" \
  http://localhost:8081/api/v1/explore/article/{article_id}/vote

# Get vote counts (requires auth)
curl -H "Authorization: Bearer <JWT>" \
  http://localhost:8081/api/v1/explore/article/{article_id}/vote
```

## Future Improvements

### Database Migration System
**Priority**: MEDIUM - Nice to have for migration management

Currently, migrations are automatically run when PostgreSQL containers are first created. A more robust migration system could include:

- **Migration Runner**: Create `fetcher/internal/db/migrations.go` and `recommender/internal/db/migrations.go` with programmatic migration execution
- **Version Tracking**: Track applied migrations in a migrations table to prevent re-running
- **Rollback Support**: Add down migrations for reverting changes
- **Migration CLI**: Create standalone commands for running/reverting migrations
- **Consider Migration Tools**: Evaluate tools like [golang-migrate](https://github.com/golang-migrate/migrate) or [goose](https://github.com/pressly/goose) for production use

### Monitoring & Observability
**Priority**: LOW - Nice to have for production deployments

#### Fetcher Metrics
- Fetch duration per feed (histogram)
- Number of new articles found per fetch
- Number of articles successfully sent to recommender
- Success/failure rate per feed
- Articles filtered (too old) count
- Feeds enabled/disabled count over time
- Average time between fetches per feed

#### Recommender Metrics
- Article ingestion rate
- Recommendation request latency
- Vote activity (upvotes/downvotes per hour)
- Quality score distribution
- Cache hit rates (if caching implemented)

#### Structured Logging
- Log each feed fetch with timing and outcome
- Log article filtering decisions with reasons
- Log feed health status changes (enable/disable events)
- Log database operations with query timing
- Log recommendation selections with scores
- Use structured logging format (JSON) for easier parsing

#### Observability Tools
- **Prometheus**: Metrics collection and alerting
- **Grafana**: Dashboards for visualizing metrics
- **Loki**: Log aggregation and querying
- **Jaeger/Zipkin**: Distributed tracing across services
- **Health Check Endpoints**: Expand health checks to include dependency status

### Additional Enhancements
- **Admin Dashboard**: Web UI for managing feeds and monitoring system health
- **Feed Discovery**: Auto-discover feeds from OPML imports or user submissions
- **Content Filtering**: Add keyword-based filtering or content categorization
- **Performance Optimization**: Add caching layer (Redis) for frequently accessed data
- **Rate Limiting**: Implement per-user rate limiting for API endpoints
- **Full-Text Search**: Add search capabilities across articles using PostgreSQL full-text search or Elasticsearch
- **Article Summarization**: Integrate AI-powered summarization for long articles
- **Email Digests**: Send daily/weekly email summaries of top recommendations

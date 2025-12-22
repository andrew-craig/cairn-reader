# Fetcher Test Suite

This directory contains comprehensive tests for the Cairn Explore fetcher service.

## Test Files

### Unit Tests
- **`internal/db/feed_repository_test.go`** (16 tests) - Feed repository database operations
- **`internal/sync/feed_sync_test.go`** (7 tests) - Feed synchronization logic
- **`internal/fetcher/fetcher_test.go`** (12 tests) - RSS fetching and article filtering

### Integration Tests
- **`integration_test.go`** (4 tests) - End-to-end workflow tests

### Test Utilities
- **`internal/testutil/helpers.go`** - Shared test utilities and fixtures

## Prerequisites

### Database Setup
Tests require the fetcher PostgreSQL database to be running. Start it with:

```bash
# From project root
docker compose up -d fetcher_db

# Verify database is running
docker compose ps fetcher_db
```

The test database connects to:
- **Host**: localhost
- **Port**: 5433
- **User**: fetcher
- **Password**: fetcher_password
- **Database**: fetcher_db

### Schema Initialization
The database schema is automatically initialized from `migrations/001_init_schema.sql` when the container first starts. If you need to reset the database:

```bash
# Stop and remove database
docker compose down
docker volume rm cairn-explore_fetcher_postgres_data

# Start fresh
docker compose up -d fetcher_db
```

## Running Tests

### All Tests
```bash
# From project root
go test ./fetcher/...

# With verbose output
go test -v ./fetcher/...

# With coverage
go test -cover ./fetcher/...
go test -coverprofile=coverage.out ./fetcher/...
go tool cover -html=coverage.out  # View coverage report
```

### Specific Test Suites
```bash
# Feed repository tests
go test -v ./fetcher/internal/db/

# Feed syncer tests
go test -v ./fetcher/internal/sync/

# Fetcher logic tests
go test -v ./fetcher/internal/fetcher/

# Integration tests
go test -v ./fetcher/
```

### Running Individual Tests
```bash
# Run a specific test
go test -v ./fetcher/internal/db/ -run TestGetNextFeed_PrioritizesNeverFetched

# Run tests matching a pattern
go test -v ./fetcher/... -run ".*Priority.*"
```

## Test Coverage

### Test Statistics
- **Total Tests**: 39
- **Unit Tests**: 35
- **Integration Tests**: 4

### Coverage by Component

#### Feed Repository (16 tests)
- ✅ GetNextFeed prioritization logic
- ✅ UpdateFetchResult success/failure handling
- ✅ Auto-disable after 10 failures
- ✅ ImportFeeds with duplicate handling
- ✅ RecordFetchHistory
- ✅ ListFeeds with filtering

#### Feed Syncer (7 tests)
- ✅ Sync from Kagi list
- ✅ Comment and blank line handling
- ✅ Metadata preservation on re-sync
- ✅ HTTP error handling
- ✅ Empty response handling
- ✅ Duplicate URL handling

#### Fetcher Logic (12 tests)
- ✅ Article filtering by publish date
- ✅ First fetch vs. incremental fetch
- ✅ Article conversion to models
- ✅ ID generation consistency
- ✅ RSS fetching success/failure
- ✅ Recommender submission handling

#### Integration Tests (4 tests)
- ✅ Complete end-to-end workflow
- ✅ Feed health tracking and auto-disable
- ✅ Feed prioritization (never-fetched > oldest)
- ✅ Daily sync metadata preservation

## Test Scenarios

### 1. Feed List Management
Tests verify:
- Initial sync from Kagi feed list
- New feeds are added to database
- Existing feeds preserve metadata (enabled, failures, last_fetched_at)
- Comment lines and blank lines are ignored
- Duplicate URLs are handled correctly

### 2. Feed Prioritization
Tests verify:
- Never-fetched feeds (last_fetched_at IS NULL) are selected first
- After all never-fetched feeds, oldest-fetched feeds are selected
- Disabled feeds are completely skipped
- Returns nil when no enabled feeds exist

### 3. Article Collection
Tests verify:
- RSS feeds are fetched and parsed successfully
- Only new articles (published after last_fetched_at) are sent
- First fetch sends all articles
- Articles are converted to correct model format
- Articles are submitted to recommender service

### 4. Failure Handling
Tests verify:
- consecutive_failures counter increments on fetch failure
- Feed is automatically disabled after 10 consecutive failures
- Counter resets to 0 on successful fetch
- Fetch history is recorded for all attempts (success and failure)
- Error messages are stored in fetch history

### 5. New Content Detection
Tests verify:
- Articles published after last fetch are included
- Articles published before last fetch are excluded
- Falls back to UpdatedParsed when PublishedParsed is nil
- Empty feeds are handled gracefully

## Troubleshooting

### Database Connection Errors
If you see `connection refused` errors:
```bash
# Check if database is running
docker compose ps fetcher_db

# Check database logs
docker compose logs fetcher_db

# Restart database
docker compose restart fetcher_db
```

### Schema Errors
If you see `relation "feeds" does not exist` errors:
```bash
# Reset database to run migrations
docker compose down
docker volume rm cairn-explore_fetcher_postgres_data
docker compose up -d fetcher_db

# Wait for database to initialize (check logs)
docker compose logs -f fetcher_db
```

### Test Data Cleanup
Tests use `defer testutil.CleanupTestDB()` to clean up after each test. If tests fail and leave data:
```bash
# Connect to database and manually clean
docker compose exec fetcher_db psql -U fetcher -d fetcher_db

# In psql:
DELETE FROM fetch_history;
DELETE FROM feeds;
```

## CI/CD Integration

### GitHub Actions Example
```yaml
name: Test Fetcher

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest

    services:
      postgres:
        image: postgres:15-alpine
        env:
          POSTGRES_DB: fetcher_db
          POSTGRES_USER: fetcher
          POSTGRES_PASSWORD: fetcher_password
        ports:
          - 5433:5432
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.23'

      - name: Run migrations
        run: |
          PGPASSWORD=fetcher_password psql -h localhost -p 5433 -U fetcher -d fetcher_db -f fetcher/migrations/001_init_schema.sql

      - name: Run tests
        run: go test -v -cover ./fetcher/...
```

## Writing New Tests

### Test Template
```go
func TestNewFeature(t *testing.T) {
    // Setup database
    database := testutil.SetupTestDB(t)
    defer database.Close()
    defer testutil.CleanupTestDB(t, database)

    // Create repository
    repo := db.NewFeedRepository(database)
    ctx := context.Background()

    // Test logic here

    // Assertions
    if result != expected {
        t.Errorf("Expected %v, got %v", expected, result)
    }
}
```

### Best Practices
1. Always use `defer testutil.CleanupTestDB()` to clean up test data
2. Use table-driven tests for multiple similar test cases
3. Use descriptive test names: `TestFunction_Scenario_ExpectedBehavior`
4. Mock external dependencies (HTTP servers, recommender client)
5. Test both success and failure paths
6. Verify database state changes with helper functions

## Further Reading

- [Go Testing Documentation](https://golang.org/pkg/testing/)
- [Table-Driven Tests in Go](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests)
- [Testing Best Practices](https://github.com/golang/go/wiki/TestComments)

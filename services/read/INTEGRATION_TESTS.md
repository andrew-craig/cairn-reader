# Integration Tests

This document describes how to run the integration tests for the Cairn backend services.

## Overview

The integration tests cover end-to-end flows from HTTP requests through the service layer to the database. They test:

### Content Service
- Full content creation flow (HTTP → Service → Repository → Database)
- User-content relationship management
- Content update propagation (preserves user relationships)
- Search functionality with PostgreSQL full-text search
- Content deduplication by hash and feed ID
- Bulk operations

### RSS Fetcher Service
- Feed subscription and management
- Feed limit enforcement (100 feeds per user)
- Outbox pattern delivery flow
- RSS item processing and deduplication
- Feed polling logic
- Error tracking and auto-disable after 7 consecutive days

## Prerequisites

The integration tests require PostgreSQL databases to be running. You have two options:

### Option 1: Use Docker Compose (Recommended)

Start the databases using docker-compose:

```bash
docker-compose up -d content-db rss-db
```

This will start:
- Content Service database on port 5433
- RSS Fetcher Service database on port 5434

### Option 2: Use Local PostgreSQL

If you have PostgreSQL running locally, create test databases and set environment variables:

```bash
# For Content Service tests
export TEST_DB_HOST=localhost
export TEST_DB_PORT=5433
export TEST_DB_USER=cairn_content
export TEST_DB_PASSWORD=cairn_content_pass
export TEST_DB_NAME=cairn_content_test

# For RSS Fetcher Service tests
export TEST_DB_HOST=localhost
export TEST_DB_PORT=5434
export TEST_DB_USER=cairn_rss
export TEST_DB_PASSWORD=cairn_rss_pass
export TEST_DB_NAME=cairn_rss_test
```

## Running Integration Tests

### Run All Integration Tests

From the project root:

```bash
# Content Service integration tests
cd services/content-service
go test -tags=integration -v ./...

# RSS Fetcher Service integration tests
cd services/rss-fetcher-service
go test -tags=integration -v ./...
```

### Run Specific Test

```bash
# Run a specific test function
go test -tags=integration -v -run TestContentCreationIntegration

# Run tests matching a pattern
go test -tags=integration -v -run "TestContent.*"
```

### Run with Coverage

```bash
# Generate coverage report
go test -tags=integration -coverprofile=coverage.out -covermode=atomic ./...

# View coverage in browser
go tool cover -html=coverage.out
```

### Skip Integration Tests

Integration tests are marked with the `integration` build tag and can be skipped:

```bash
# Unit tests only (default behavior)
go test ./...

# Or explicitly skip with short flag
go test -short ./...
```

## How Integration Tests Work

### Test Database Setup

Each test creates a unique test database:

1. Connects to the main PostgreSQL instance
2. Creates a temporary database with timestamp suffix (e.g., `cairn_content_test_1234567890`)
3. Runs all migrations to set up the schema
4. Executes the test
5. Drops the temporary database on cleanup

This ensures:
- Complete isolation between test runs
- Clean state for each test
- No interference between parallel tests

### Test Helpers

Both services include a `testhelpers` package with utilities for:

- `SetupTestDatabase(t)` - Creates and migrates a test database
- `Cleanup()` - Drops the test database
- `TruncateAll()` - Clears all tables (faster than recreation for sequential tests)

### Example Test Structure

```go
func TestContentCreationIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    // Setup
    testDB := testhelpers.SetupTestDatabase(t)
    defer testDB.Cleanup()

    // Create test server
    dbWrapper := &database.DB{DB: testDB.DB}
    router := api.NewRouter(dbWrapper)
    server := httptest.NewServer(router)
    defer server.Close()

    // Run tests
    t.Run("SubTest", func(t *testing.T) {
        // Test implementation
    })
}
```

## Test Coverage Goals

According to Phase 5.5 of the implementation plan, integration tests are expected to contribute:

- **+10-15% coverage gain**
- Focus on end-to-end validation
- Test complete request flows
- Verify database state after operations

## Troubleshooting

### Database Connection Issues

If tests fail to connect to the database:

1. Ensure PostgreSQL is running:
   ```bash
   docker-compose ps
   ```

2. Check database logs:
   ```bash
   docker-compose logs content-db
   docker-compose logs rss-db
   ```

3. Verify connection parameters in test output

### Migration Errors

If migrations fail during test setup:

1. Ensure migration files exist in `services/*/migrations/`
2. Check migration file paths in test helper
3. Verify migrations work manually:
   ```bash
   migrate -path ./migrations -database "postgres://..." up
   ```

### Port Conflicts

If you get "address already in use" errors:

1. Check if databases are already running:
   ```bash
   lsof -i :5433
   lsof -i :5434
   ```

2. Stop conflicting services or use different ports via environment variables

## CI/CD Integration

For continuous integration, the tests can be run with:

```yaml
# Example GitHub Actions workflow
- name: Start PostgreSQL
  run: docker-compose up -d content-db rss-db

- name: Wait for databases
  run: |
    timeout 60 bash -c 'until docker exec cairn-content-db pg_isready -U cairn_content; do sleep 2; done'
    timeout 60 bash -c 'until docker exec cairn-rss-db pg_isready -U cairn_rss; do sleep 2; done'

- name: Run Integration Tests
  run: |
    cd services/content-service && go test -tags=integration -v ./...
    cd services/rss-fetcher-service && go test -tags=integration -v ./...

- name: Cleanup
  run: docker-compose down
```

## Notes

- Integration tests create and destroy databases, so they are slower than unit tests
- Each test run creates unique databases to allow parallel execution
- Tests use real HTTP servers via `httptest.NewServer`
- Database state is verified directly via SQL queries to ensure correctness
- All tests follow the Arrange-Act-Assert pattern for clarity

## Test Scenarios Covered

### Content Service Integration Tests

1. **TestContentCreationIntegration**
   - Create content from HTML
   - Retrieve content by ID
   - Bulk create multiple contents
   - Verify HTML sanitization (no script tags)

2. **TestUserContentIntegration**
   - Add content to user's reading list
   - List user's contents with pagination
   - Update user-content metadata (status, favorite, scroll position)
   - Delete content from user's list

3. **TestSearchIntegration**
   - Search by title using full-text search
   - Search by author
   - Verify no results for non-matching queries

4. **TestContentUpdatePropagation**
   - Update content preserves all user relationships
   - User metadata remains intact after content update
   - Multiple users can have different metadata for same content

5. **TestDeduplicationIntegration**
   - Content deduplication by hash and feed ID
   - Check-duplicates endpoint
   - Only one database entry for duplicate content

### RSS Fetcher Service Integration Tests

1. **TestFeedSubscriptionIntegration**
   - Subscribe to new feed
   - List user's feed subscriptions
   - Unsubscribe from feed
   - Enforce 100 feed limit per user

2. **TestOutboxPatternIntegration**
   - Create outbox entries
   - Get pending entries for delivery
   - Update delivery status
   - Retry logic with exponential backoff

3. **TestFeedItemProcessingIntegration**
   - Create feed items
   - Deduplication by GUID
   - Get pending items for processing
   - Update processing status

4. **TestFeedPollingIntegration**
   - Get feeds due for polling
   - Update feed after successful poll
   - Track consecutive errors
   - Auto-disable after 7 consecutive error days

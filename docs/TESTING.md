# Testing Guide

This document provides comprehensive testing guidelines for the Cairn backend services, including unit tests, integration tests, and end-to-end tests.

## Table of Contents

- [Testing Philosophy](#testing-philosophy)
- [Test Types Overview](#test-types-overview)
- [Prerequisites](#prerequisites)
- [Running Tests](#running-tests)
  - [Unit Tests](#unit-tests)
  - [Integration Tests](#integration-tests)
  - [End-to-End Tests](#end-to-end-tests)
- [Service-Specific Testing](#service-specific-testing)
  - [Explore Service](#explore-service)
  - [Read Service](#read-service)
  - [User Service](#user-service)
- [Test Helpers and Utilities](#test-helpers-and-utilities)
- [CI/CD Integration](#cicd-integration)
- [Troubleshooting](#troubleshooting)

---

## Testing Philosophy

Cairn follows a comprehensive testing strategy with three test layers:

1. **Unit Tests**: Fast, isolated tests for individual functions and components
2. **Integration Tests**: Test database interactions and service layer integration
3. **End-to-End Tests**: Validate complete workflows across multiple services

For detailed testing principles, see [Engineering Principles - Testing Philosophy](/docs/ENGINEERING_PRINCIPLES.md#testing-philosophy).

**Key Principles**:
- Tests should be independent and isolated
- Use real databases for integration tests (no mocks for database layer)
- Each test creates a unique test database to allow parallel execution
- Clean up resources properly (databases, connections, test data)

---

## Test Types Overview

### Unit Tests
- **Purpose**: Test individual functions, methods, and components in isolation
- **Speed**: Fast (milliseconds)
- **Dependencies**: None (mocked)
- **Coverage Target**: 60-70% of codebase

### Integration Tests
- **Purpose**: Test end-to-end flows from HTTP → Service → Repository → Database
- **Speed**: Moderate (seconds)
- **Dependencies**: Real PostgreSQL databases
- **Coverage Target**: +10-15% gain over unit tests

### End-to-End Tests
- **Purpose**: Validate complete workflows across multiple services
- **Speed**: Slow (minutes)
- **Dependencies**: All services running (Docker Compose)
- **Coverage Target**: Critical user journeys

---

## Prerequisites

### Database Requirements

All integration and E2E tests require PostgreSQL databases. You have two options:

#### Option 1: Docker Compose (Recommended)

Start all databases using the centralized Docker Compose setup:

```bash
cd infrastructure/docker
docker compose up -d

# Or start specific databases
docker compose up -d content-db rss-db explore-fetcher-db explore-recommender-db users-db
```

This starts:
- Content Service database (port 5433)
- RSS Fetcher Service database (port 5434)
- Explore Fetcher database (port 5435)
- Explore Recommender database (port 5432)
- User Service database (port 5436)

#### Option 2: Local PostgreSQL

If you have PostgreSQL running locally, create test databases and set environment variables:

```bash
# For Content Service
export TEST_DB_HOST=localhost
export TEST_DB_PORT=5433
export TEST_DB_USER=cairn_content
export TEST_DB_PASSWORD=cairn_content_pass
export TEST_DB_NAME=cairn_content_test

# For RSS Fetcher Service
export TEST_DB_HOST=localhost
export TEST_DB_PORT=5434
export TEST_DB_USER=cairn_rss
export TEST_DB_PASSWORD=cairn_rss_pass
export TEST_DB_NAME=cairn_rss_test

# For User Service
export TEST_DB_HOST=localhost
export TEST_DB_PORT=5436
export TEST_DB_USER=cairn_users
export TEST_DB_PASSWORD=cairn_users_pass
export TEST_DB_NAME=cairn_users_test
```

### Additional Tools

- **Docker**: For running databases and services
- **Make**: For running test commands
- **Go 1.21+**: For running Go tests
- **curl/jq**: For E2E testing

---

## Running Tests

### Unit Tests

Unit tests run without external dependencies and are the default test mode.

```bash
# Run all unit tests for a service
cd services/{service-name}
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests with coverage
go test -cover ./...

# Run specific test
go test -v -run TestFunctionName ./...

# Run tests matching a pattern
go test -v -run "TestContent.*" ./...
```

**Examples**:

```bash
# Explore service
cd services/explore
make test

# Read service
cd services/read
make test

# User service
cd services/users
make test
```

### Integration Tests

Integration tests are marked with the `integration` build tag and require databases to be running.

```bash
# Start databases
cd infrastructure/docker
docker compose up -d content-db rss-db users-db

# Run integration tests for a service
cd services/{service-name}
go test -tags=integration -v ./...

# Run specific integration test
go test -tags=integration -v -run TestContentCreationIntegration ./...

# Run with coverage
go test -tags=integration -coverprofile=coverage.out -covermode=atomic ./...

# View coverage in browser
go tool cover -html=coverage.out
```

**Examples**:

```bash
# Content Service integration tests
cd services/read/content-service
go test -tags=integration -v ./...

# RSS Fetcher Service integration tests
cd services/read/rss-fetcher-service
go test -tags=integration -v ./...
```

**Skip Integration Tests**:

```bash
# Skip with short flag
go test -short ./...

# Or just run unit tests (default)
go test ./...
```

### End-to-End Tests

E2E tests validate complete workflows across multiple services.

#### Explore Service E2E Tests

**Test Environment**:
- Fetcher Service: `http://localhost:8080`
- Recommender Service: `http://localhost:8081`
- Fetcher DB: PostgreSQL on port 5435
- Recommender DB: PostgreSQL on port 5432

**Setup**:

```bash
# Prerequisites
cd services/explore

# Clean slate
docker compose down
docker volume rm cairn-explore_postgres_data cairn-explore_fetcher_postgres_data

# Start services
docker compose up --build -d

# Wait for initialization
sleep 30

# Check service health
curl http://localhost:8080/health
curl http://localhost:8081/health
```

**Test Execution**:

Follow the test scenarios in the [E2E Test Plan](#explore-service-e2e-test-plan) below.

**Cleanup**:

```bash
docker compose down
docker volume rm cairn-explore_postgres_data cairn-explore_fetcher_postgres_data
```

---

## Service-Specific Testing

### Explore Service

The Explore service consists of two microservices that work together to fetch and recommend articles.

#### Architecture Under Test

```
Kagi Feed List → Fetcher DB ← Fetcher Service (8080) → Recommender Service (8081) → Recommender DB
                      ↓                                           ↓
                Feed Management                          Article Storage & Recommendations
```

#### Fetcher Test Suite

The fetcher service has comprehensive unit and integration tests covering all components.

**Test Statistics**:
- **Total Tests**: 39
- **Unit Tests**: 35
- **Integration Tests**: 4

**Test Files**:
- `internal/db/feed_repository_test.go` (16 tests) - Feed repository database operations
- `internal/sync/feed_sync_test.go` (7 tests) - Feed synchronization logic
- `internal/fetcher/fetcher_test.go` (12 tests) - RSS fetching and article filtering
- `integration_test.go` (4 tests) - End-to-end workflow tests

**Running Fetcher Tests**:

```bash
# From project root
cd services/explore

# All tests
go test ./fetcher/...

# With verbose output
go test -v ./fetcher/...

# With coverage
go test -cover ./fetcher/...
go test -coverprofile=coverage.out ./fetcher/...
go tool cover -html=coverage.out

# Specific test suites
go test -v ./fetcher/internal/db/        # Feed repository tests
go test -v ./fetcher/internal/sync/      # Feed syncer tests
go test -v ./fetcher/internal/fetcher/   # Fetcher logic tests
go test -v ./fetcher/                    # Integration tests

# Run individual test
go test -v ./fetcher/internal/db/ -run TestGetNextFeed_PrioritizesNeverFetched
```

**Coverage by Component**:

1. **Feed Repository (16 tests)**
   - GetNextFeed prioritization logic
   - UpdateFetchResult success/failure handling
   - Auto-disable after 10 failures
   - ImportFeeds with duplicate handling
   - RecordFetchHistory
   - ListFeeds with filtering

2. **Feed Syncer (7 tests)**
   - Sync from Kagi list
   - Comment and blank line handling
   - Metadata preservation on re-sync
   - HTTP error handling
   - Empty response handling
   - Duplicate URL handling

3. **Fetcher Logic (12 tests)**
   - Article filtering by publish date
   - First fetch vs. incremental fetch
   - Article conversion to models
   - ID generation consistency
   - RSS fetching success/failure
   - Recommender submission handling

4. **Integration Tests (4 tests)**
   - Complete end-to-end workflow
   - Feed health tracking and auto-disable
   - Feed prioritization (never-fetched > oldest)
   - Daily sync metadata preservation

**Test Scenarios**:

- **Feed List Management**: Initial sync, new feed addition, metadata preservation, duplicate handling
- **Feed Prioritization**: Never-fetched feeds first, then oldest-fetched, disabled feeds skipped
- **Article Collection**: RSS parsing, new article detection, model conversion, recommender submission
- **Failure Handling**: consecutive_failures counter, auto-disable after 10 failures, history recording
- **New Content Detection**: Article filtering by publish date, fallback to UpdatedParsed

**Database Setup for Fetcher Tests**:

```bash
# Start fetcher database
docker compose up -d fetcher_db

# Verify database is running
docker compose ps fetcher_db

# Reset database if needed
docker compose down
docker volume rm cairn-explore_fetcher_postgres_data
docker compose up -d fetcher_db
```

The fetcher test database connects to:
- **Host**: localhost
- **Port**: 5435
- **User**: fetcher
- **Password**: fetcher_password
- **Database**: fetcher_db

**Test Utilities**:

The fetcher service includes test helpers in `internal/testutil/helpers.go`:

```go
// Setup test database connection
database := testutil.SetupTestDB(t)
defer database.Close()
defer testutil.CleanupTestDB(t, database)

// Create repository
repo := db.NewFeedRepository(database)
ctx := context.Background()

// Test logic and assertions
```

**Best Practices**:
1. Always use `defer testutil.CleanupTestDB()` to clean up test data
2. Use table-driven tests for multiple similar test cases
3. Use descriptive test names: `TestFunction_Scenario_ExpectedBehavior`
4. Mock external dependencies (HTTP servers, recommender client)
5. Test both success and failure paths
6. Verify database state changes with helper functions

#### E2E Test Plan

**Test Objective**: Validate the complete integration of the Fetcher and Recommender services, ensuring all core functionality works correctly from feed fetching to article recommendations.

##### 1. Service Infrastructure Tests

**Objective**: Verify all services start correctly and are healthy

- [ ] Start Docker Compose stack (postgres DBs, fetcher, recommender)
- [ ] Verify fetcher health endpoints (`GET /health/live`, `GET /health/ready`)
- [ ] Verify recommender health endpoints (`GET /health/live`, `GET /health/ready`)
- [ ] Check database connectivity for both services
- [ ] Verify Docker containers are running without errors

**Expected Results**:
- All containers running and healthy
- Liveness endpoints return `{"status":"healthy"}`
- Readiness endpoints return `{"status":"healthy","checks":{"database":"ok"}}` (or `unhealthy` with 503 if DB unreachable)
- No connection errors in logs

##### 2. Feed Management Tests

**Objective**: Validate feed sync and storage in fetcher database

- [ ] Verify feed sync from Kagi Small Web collection
- [ ] Check feeds are stored in fetcher database
- [ ] Verify feeds table schema (url, enabled, last_fetched_at, consecutive_failures)
- [ ] Confirm initial state: enabled=true, consecutive_failures=0
- [ ] Verify no duplicate feed URLs

**Expected Results**:
- Feeds successfully synced and stored
- All feeds enabled by default
- Unique URLs enforced

**Validation Commands**:
```sql
SELECT COUNT(*) FROM feeds;
SELECT * FROM feeds WHERE enabled = false;
SELECT url, COUNT(*) FROM feeds GROUP BY url HAVING COUNT(*) > 1;
```

##### 3. Feed Fetching Tests

**Objective**: Validate one-feed-per-minute fetching logic

- [ ] Trigger manual fetch (`POST /api/v1/explore/feed/fetch` or wait for automatic fetch)
- [ ] Verify only 1 feed fetched per minute
- [ ] Confirm feed prioritization (never-fetched first, then oldest last_fetched_at)
- [ ] Verify last_fetched_at timestamp updated
- [ ] Check fetch_history table for fetch records
- [ ] Monitor consecutive_failures counter on fetch errors

**Expected Results**:
- Exactly 1 feed processed per minute
- Never-fetched feeds prioritized
- Timestamps updated correctly
- Fetch history recorded

**Validation Commands**:
```sql
SELECT * FROM feeds WHERE last_fetched_at IS NOT NULL ORDER BY last_fetched_at DESC LIMIT 5;
SELECT * FROM fetch_history ORDER BY fetched_at DESC LIMIT 10;
```

##### 4. Article Submission Tests

**Objective**: Verify articles are sent from Fetcher to Recommender

- [ ] Verify fetcher submits articles via `POST /api/v1/explore/article`
- [ ] Check article deduplication (SHA256 hash IDs)
- [ ] Verify article metadata stored (title, link, content, published_at, categories)
- [ ] Confirm feed_id reference set correctly
- [ ] Verify batch submission handling

**Expected Results**:
- Articles successfully submitted to recommender
- Duplicate articles handled with ON CONFLICT
- All metadata fields populated
- feed_id links back to fetcher database feed ID

**Validation Commands**:
```sql
SELECT COUNT(*) FROM articles;
SELECT id, title, feed_id, published_at FROM articles ORDER BY created_at DESC LIMIT 10;
SELECT COUNT(*) as article_count, feed_id FROM articles GROUP BY feed_id;
```

##### 5. Recommendation Engine Tests

**Objective**: Validate recommendation algorithm and user tracking

- [ ] Request recommendations for new user (`GET /api/v1/explore/recommendation`, requires JWT)
- [ ] Verify 5 articles returned
- [ ] Check recommendation scoring (quality score: `(upvotes + (downvotes * 3)) / recommends`)
- [ ] Verify algorithm selects 4 high-quality articles + 1 low-exposure article
- [ ] Request recommendations for same user again
- [ ] Verify different articles recommended (already-recommended articles excluded)

**Expected Results**:
- Exactly 5 articles per request
- Articles sorted by score
- Variety in recommendations across requests

**Test Commands**:
```bash
curl -H "Authorization: Bearer <user-001-JWT>" \
  http://localhost:8081/api/v1/explore/recommendation
curl -H "Authorization: Bearer <user-002-JWT>" \
  http://localhost:8081/api/v1/explore/recommendation
```

##### 6. User Interaction Tests

**Objective**: Test read status tracking and filtering

- [ ] Mark article as read (`POST /api/v1/explore/article/{article_id}/read`, requires JWT)
- [ ] Verify user_articles table entry created
- [ ] Request recommendations for same user
- [ ] Confirm read article NOT in recommendations
- [ ] Mark multiple articles as read
- [ ] Verify all read articles filtered from recommendations

**Expected Results**:
- Read status persisted correctly
- Read articles excluded from future recommendations
- User tracking works independently per user

**Test Commands**:
```bash
# Get recommendations (requires JWT)
RECS=$(curl -s -H "Authorization: Bearer <JWT>" \
  http://localhost:8081/api/v1/explore/recommendation)
ARTICLE_ID=$(echo $RECS | jq -r '.articles[0].id')

# Mark as read (requires JWT)
curl -X POST \
  -H "Authorization: Bearer <JWT>" \
  http://localhost:8081/api/v1/explore/article/$ARTICLE_ID/read

# Verify filtered out
curl -s -H "Authorization: Bearer <JWT>" \
  http://localhost:8081/api/v1/explore/recommendation | jq
```

**Validation SQL**:
```sql
SELECT * FROM user_articles WHERE user_id = 'test-user-001';
```

##### 7. Feed Health Tracking Tests

**Objective**: Validate automatic feed disabling on failures

- [ ] Identify feed with connection errors
- [ ] Verify consecutive_failures counter increments
- [ ] Simulate 10 consecutive failures
- [ ] Verify feed auto-disabled (enabled = false)
- [ ] Confirm disabled feeds not fetched
- [ ] Simulate successful fetch
- [ ] Verify consecutive_failures reset to 0

**Expected Results**:
- Failure counter increments on errors
- Feed disabled after 10 failures
- Disabled feeds skipped in fetch cycle
- Counter resets on successful fetch

**Validation Commands**:
```sql
SELECT url, enabled, consecutive_failures, last_fetched_at
FROM feeds
WHERE consecutive_failures > 0
ORDER BY consecutive_failures DESC
LIMIT 10;

SELECT COUNT(*) FROM feeds WHERE enabled = false;
```

##### 8. Multi-User Isolation Tests

**Objective**: Verify user data isolation

- [ ] Create multiple test users
- [ ] Mark different articles as read for each user
- [ ] Verify each user gets personalized recommendations
- [ ] Confirm no cross-user data leakage

**Expected Results**:
- User recommendations independent
- Read status isolated per user
- No data leakage between users

**Test Users**: test-user-001, test-user-002, test-user-003

##### 9. Performance & Load Tests

**Objective**: Validate system performance under normal conditions

- [ ] Monitor fetcher memory usage during fetch cycle
- [ ] Check recommender response times (should be < 100ms)
- [ ] Verify batch article submission performance
- [ ] Monitor database connection pooling
- [ ] Check for memory leaks over 10-minute period

**Expected Results**:
- Stable memory usage
- Fast recommendation response times
- No connection pool exhaustion

##### 10. Error Handling Tests

**Objective**: Validate graceful error handling

- [ ] Submit invalid article data to recommender
- [ ] Request recommendations for user with no available articles
- [ ] Mark non-existent article as read
- [ ] Test fetcher behavior with unreachable feeds
- [ ] Verify appropriate HTTP status codes (400, 404, 500)

**Expected Results**:
- Graceful error responses
- No service crashes
- Proper HTTP status codes
- Error messages in logs

#### Success Criteria

All tests must pass with:
- ✅ All services healthy
- ✅ Feeds successfully synced and fetched
- ✅ Articles submitted and stored correctly
- ✅ Recommendations returned with correct filtering
- ✅ User tracking working independently
- ✅ Feed health tracking operational
- ✅ No critical errors in logs
- ✅ Response times acceptable (< 100ms for recommendations)

#### Implementation Status

All previously planned features are now implemented:
- ✅ Voting system (`POST/DELETE/GET /api/v1/explore/article/{article_id}/vote`)
- ✅ Quality score algorithm: `(upvotes + (downvotes * 3)) / recommends`
- ✅ Enhanced recommendation algorithm (4 high-quality + 1 low-exposure)
- ✅ Article cleanup (90-day retention with two-phase soft/hard delete)
- ✅ Votes and recommendations tables (in recommender database)

---

### Read Service

The Read service consists of two microservices: Content Service and RSS Fetcher Service.

#### Integration Test Overview

Integration tests cover end-to-end flows from HTTP requests through the service layer to the database.

**Test Coverage**:
- Full content creation flow (HTTP → Service → Repository → Database)
- User-content relationship management
- Content update propagation (preserves user relationships)
- Search functionality with PostgreSQL full-text search
- Content deduplication by hash and feed ID
- RSS feed subscription and management
- Feed limit enforcement (100 feeds per user)
- Outbox pattern delivery flow
- RSS item processing and deduplication
- Feed polling logic
- Error tracking and auto-disable after 7 consecutive days

#### Content Service Integration Tests

##### 1. TestContentCreationIntegration

Tests the complete flow of creating and retrieving content.

**Scenarios**:
- Create content from HTML
- Retrieve content by ID
- Bulk create multiple contents
- Verify HTML sanitization (no script tags)

**Example**:
```go
func TestContentCreationIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    testDB := testhelpers.SetupTestDatabase(t)
    defer testDB.Cleanup()

    // Test implementation
}
```

##### 2. TestUserContentIntegration

Tests user-specific content management.

**Scenarios**:
- Add content to user's reading list
- List user's contents with pagination
- Update user-content metadata (status, favorite, scroll position)
- Delete content from user's list

##### 3. TestSearchIntegration

Tests full-text search functionality.

**Scenarios**:
- Search by title using full-text search
- Search by author
- Verify no results for non-matching queries

##### 4. TestContentUpdatePropagation

Tests content updates while preserving user relationships.

**Scenarios**:
- Update content preserves all user relationships
- User metadata remains intact after content update
- Multiple users can have different metadata for same content

##### 5. TestDeduplicationIntegration

Tests content deduplication logic.

**Scenarios**:
- Content deduplication by hash and feed ID
- Check-duplicates endpoint
- Only one database entry for duplicate content

#### RSS Fetcher Service Integration Tests

##### 1. TestFeedSubscriptionIntegration

Tests feed subscription management.

**Scenarios**:
- Subscribe to new feed
- List user's feed subscriptions
- Unsubscribe from feed
- Enforce 100 feed limit per user

##### 2. TestOutboxPatternIntegration

Tests reliable message delivery.

**Scenarios**:
- Create outbox entries
- Get pending entries for delivery
- Update delivery status
- Retry logic with exponential backoff

##### 3. TestFeedItemProcessingIntegration

Tests RSS item processing.

**Scenarios**:
- Create feed items
- Deduplication by GUID
- Get pending items for processing
- Update processing status

##### 4. TestFeedPollingIntegration

Tests feed polling logic.

**Scenarios**:
- Get feeds due for polling
- Update feed after successful poll
- Track consecutive errors
- Auto-disable after 7 consecutive error days

#### Running Read Service Integration Tests

```bash
# Start database (single PostgreSQL instance for all services)
cd infrastructure/docker/dev
docker compose up -d cairn-db

# Content Service integration tests
cd services/read/content
go test -tags=integration -v ./...

# RSS Fetcher Service integration tests
cd services/read/fetcher
go test -tags=integration -v ./...

# Run with coverage
go test -tags=integration -coverprofile=coverage.out -covermode=atomic ./...
go tool cover -html=coverage.out
```

---

### User Service

The User service handles authentication and user management.

#### Test Coverage

**Unit Tests**:
- JWT token generation and validation
- Refresh token rotation
- Password hashing and verification
- Device authentication

**Integration Tests**:
- Full authentication flow (register → login → refresh)
- Token validation with Vault integration
- Account upgrade (device-only → email/password)

#### Running User Service Tests

```bash
cd services/users

# Unit tests only
make test

# Integration tests (requires Vault and database)
cd infrastructure/docker/dev
docker compose up -d vault cairn-db

cd services/users
go test -tags=integration -v ./...

# All tests with coverage
make test-coverage
```

---

## Test Helpers and Utilities

### Database Test Helpers

Both Content Service and RSS Fetcher Service include a `testutil` package (`internal/testutil/`) with utilities for integration tests.

#### SetupTestDatabase

Creates a unique test database for each test run.

```go
func SetupTestDatabase(t *testing.T) *TestDatabase
```

**Process**:
1. Connects to the main PostgreSQL instance
2. Creates a temporary database with timestamp suffix (e.g., `cairn_content_test_1234567890`)
3. Runs all migrations to set up the schema
4. Returns a `TestDatabase` instance for use in tests

**Benefits**:
- Complete isolation between test runs
- Clean state for each test
- No interference between parallel tests

#### Cleanup

Drops the temporary test database.

```go
func (td *TestDatabase) Cleanup()
```

**Usage**:
```go
testDB := testutil.SetupTestDatabase(t)
defer testDB.Cleanup()
```

#### TruncateAll

Clears all tables in the database (faster than recreation for sequential tests).

```go
func (td *TestDatabase) TruncateAll()
```

### Example Test Structure

```go
func TestContentCreationIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    // Setup
    testDB := testutil.SetupTestDatabase(t)
    defer testDB.Cleanup()

    // Create test server
    dbWrapper := &database.DB{DB: testDB.DB}
    router := api.NewRouter(dbWrapper)
    server := httptest.NewServer(router)
    defer server.Close()

    // Run tests
    t.Run("CreateContent", func(t *testing.T) {
        // Arrange
        requestBody := []byte(`{"html":"<h1>Test</h1>"}`)

        // Act
        resp, err := http.Post(server.URL+"/api/v1/content", "application/json", bytes.NewBuffer(requestBody))

        // Assert
        assert.NoError(t, err)
        assert.Equal(t, http.StatusCreated, resp.StatusCode)
    })
}
```

---

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Integration Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest

    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_USER: cairn_test
          POSTGRES_PASSWORD: cairn_test_pass
          POSTGRES_DB: cairn_test
        ports:
          - 5432:5432
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
          go-version: '1.21'

      - name: Run Unit Tests
        run: |
          cd services/content-service
          go test -v ./...

      - name: Run Integration Tests
        env:
          TEST_DB_HOST: localhost
          TEST_DB_PORT: 5432
          TEST_DB_USER: cairn_test
          TEST_DB_PASSWORD: cairn_test_pass
          TEST_DB_NAME: cairn_test
        run: |
          cd services/content-service
          go test -tags=integration -v ./...
```

### Fetcher Service CI Example

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
          - 5435:5432
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
          PGPASSWORD=fetcher_password psql -h localhost -p 5435 -U fetcher -d fetcher_db -f services/explore/fetcher/migrations/001_init_schema.sql

      - name: Run tests
        run: |
          cd services/explore
          go test -v -cover ./fetcher/...
```

### Docker Compose CI Example

```yaml
name: E2E Tests

on: [push, pull_request]

jobs:
  e2e:
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v3

      - name: Start Services
        run: |
          cd infrastructure/docker
          docker compose up -d

      - name: Wait for Services
        run: |
          timeout 60 bash -c 'until curl -f http://localhost:8080/health; do sleep 2; done'
          timeout 60 bash -c 'until curl -f http://localhost:8081/health; do sleep 2; done'

      - name: Run E2E Tests
        run: |
          cd services/explore
          ./scripts/run-e2e-tests.sh

      - name: Collect Logs
        if: always()
        run: |
          cd infrastructure/docker
          docker compose logs > e2e-test-logs.txt

      - name: Upload Logs
        if: always()
        uses: actions/upload-artifact@v3
        with:
          name: e2e-logs
          path: infrastructure/docker/e2e-test-logs.txt

      - name: Cleanup
        if: always()
        run: |
          cd infrastructure/docker
          docker compose down
```

---

## Troubleshooting

### Database Connection Issues

**Problem**: Tests fail to connect to the database

**Solutions**:

1. Ensure PostgreSQL is running:
   ```bash
   docker compose ps
   ```

2. Check database logs:
   ```bash
   docker compose logs content-db
   docker compose logs rss-db
   ```

3. Verify connection parameters in test output

4. Test connection manually:
   ```bash
   psql -h localhost -p 5433 -U cairn_content -d cairn_content_test
   ```

### Migration Errors

**Problem**: Migrations fail during test setup

**Solutions**:

1. Ensure migration files exist in `services/*/migrations/`

2. Check migration file paths in test helper

3. Verify migrations work manually:
   ```bash
   migrate -path ./migrations -database "postgres://cairn_content:cairn_content_pass@localhost:5433/cairn_content_test?sslmode=disable" up
   ```

4. Check for syntax errors in migration SQL:
   ```bash
   cat services/content-service/migrations/*.sql
   ```

### Port Conflicts

**Problem**: "Address already in use" errors

**Solutions**:

1. Check if databases are already running:
   ```bash
   lsof -i :5432
   lsof -i :5433
   lsof -i :5434
   ```

2. Stop conflicting services:
   ```bash
   docker compose down
   ```

3. Use different ports via environment variables:
   ```bash
   export TEST_DB_PORT=5555
   ```

### Test Database Cleanup Issues

**Problem**: Test databases not being cleaned up

**Solutions**:

1. Manually drop test databases:
   ```bash
   psql -h localhost -p 5433 -U cairn_content -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname LIKE 'cairn_content_test%';"
   psql -h localhost -p 5433 -U cairn_content -c "DROP DATABASE IF EXISTS cairn_content_test_1234567890;"
   ```

2. Check for open connections:
   ```sql
   SELECT * FROM pg_stat_activity WHERE datname LIKE 'cairn_content_test%';
   ```

3. Ensure `defer testDB.Cleanup()` is called in tests

### Slow Integration Tests

**Problem**: Integration tests take too long to run

**Solutions**:

1. Run tests in parallel:
   ```bash
   go test -tags=integration -v -parallel=4 ./...
   ```

2. Use `TruncateAll()` instead of recreating databases for sequential tests within the same test function

3. Skip integration tests during development:
   ```bash
   go test -short ./...
   ```

### Vault Connection Issues (User Service)

**Problem**: User service tests fail with Vault connection errors

**Solutions**:

1. Ensure Vault is running:
   ```bash
   cd infrastructure/docker
   docker compose up -d vault
   ```

2. Wait for Vault initialization:
   ```bash
   sleep 5
   ```

3. Verify Vault health:
   ```bash
   curl http://localhost:8200/v1/sys/health
   ```

4. Check Vault token in environment:
   ```bash
   echo $VAULT_TOKEN
   ```

### Docker Compose Issues

**Problem**: Services not starting in Docker Compose

**Solutions**:

1. Check service logs:
   ```bash
   docker compose logs -f <service-name>
   ```

2. Rebuild containers:
   ```bash
   docker compose down
   docker compose up --build
   ```

3. Clean volumes:
   ```bash
   docker compose down -v
   ```

4. Check resource limits:
   ```bash
   docker stats
   ```

### Fetcher Test Database Issues

**Problem**: Tests fail with "relation 'feeds' does not exist"

**Solutions**:

1. Reset database to run migrations:
   ```bash
   docker compose down
   docker volume rm cairn-explore_fetcher_postgres_data
   docker compose up -d fetcher_db

   # Wait for database to initialize (check logs)
   docker compose logs -f fetcher_db
   ```

2. Verify schema was created:
   ```bash
   docker compose exec fetcher_db psql -U fetcher -d fetcher_db -c "\dt"
   ```

**Problem**: Test data cleanup issues

**Solutions**:

1. Tests should use `defer testutil.CleanupTestDB()` to clean up after each test

2. Manually clean test data if needed:
   ```bash
   docker compose exec fetcher_db psql -U fetcher -d fetcher_db

   # In psql:
   DELETE FROM fetch_history;
   DELETE FROM feeds;
   ```

3. Check for orphaned connections:
   ```sql
   SELECT * FROM pg_stat_activity WHERE datname = 'fetcher_db';
   ```

---

## Test Coverage Goals

According to the Engineering Principles, Cairn targets:

- **60-70% overall code coverage**
- **Unit tests**: Core business logic, utility functions, validation
- **Integration tests**: +10-15% coverage gain, focus on end-to-end validation
- **E2E tests**: Critical user journeys

**Coverage by Service**:

| Service | Unit Tests | Integration Tests | E2E Tests | Total Target |
|---------|------------|-------------------|-----------|--------------|
| Content Service | 50-60% | +10-15% | N/A | 60-75% |
| RSS Fetcher | 50-60% | +10-15% | N/A | 60-75% |
| User Service | 50-60% | +10% | N/A | 60-70% |
| Explore Fetcher | 50-60% | N/A | +5-10% | 55-70% |
| Explore Recommender | 50-60% | N/A | +5-10% | 55-70% |

**Check Coverage**:

```bash
# Generate coverage report
go test -coverprofile=coverage.out -covermode=atomic ./...

# View coverage by package
go tool cover -func=coverage.out

# View coverage in browser
go tool cover -html=coverage.out

# Check coverage percentage
go tool cover -func=coverage.out | grep total
```

---

## Notes

- Integration tests create and destroy databases, so they are slower than unit tests
- Each test run creates unique databases to allow parallel execution
- Tests use real HTTP servers via `httptest.NewServer`
- Database state is verified directly via SQL queries to ensure correctness
- All tests follow the Arrange-Act-Assert pattern for clarity
- Use `testing.Short()` to skip integration tests during rapid development
- Always clean up resources (`defer testDB.Cleanup()`, `defer server.Close()`)

---

## Summary

This guide provides comprehensive testing documentation for Cairn backend services:

- **Unit Tests**: Fast, isolated tests (default mode)
- **Integration Tests**: Real database tests (requires Docker)
- **E2E Tests**: Full workflow validation (requires all services)

**Quick Commands**:

```bash
# Unit tests only
go test ./...

# Integration tests
go test -tags=integration -v ./...

# E2E tests
cd services/explore
docker compose up --build -d
# Follow E2E test plan

# Coverage report
go test -cover ./...
go tool cover -html=coverage.out
```

For service-specific details, see:
- [Explore Service CLAUDE.md](/services/explore/CLAUDE.md)
- [Read Service CLAUDE.md](/services/read/CLAUDE.md)
- [User Service CLAUDE.md](/services/users/CLAUDE.md)

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

All integration and E2E tests require PostgreSQL. Each test run creates its own uniquely named database (for example `cairn_content_test_<timestamp>`) by connecting to the `postgres` database, so the configured user needs permission to create databases.

The dev stack (`infrastructure/docker/dev`) runs a single `cairn-db` service published on port 5432. Its per-service users (created by `infrastructure/docker/scripts/init-databases.sh`) cannot create databases, and the test defaults below do not match it, so use a throwaway PostgreSQL container instead.

#### Option 1: Throwaway PostgreSQL container (Recommended)

Each service's `internal/testutil/database.go` defaults to its own port, user and password (Content: 5433, Fetcher: 5434, Email: 5435). Start one container per service under test, using the defaults as the superuser credentials:

```bash
# Content Service (defaults: localhost:5433, cairn_content / cairn_content_pass)
docker run -d --name cairn-test-content -p 5433:5432 \
  -e POSTGRES_USER=cairn_content -e POSTGRES_PASSWORD=cairn_content_pass \
  postgres:18-alpine

# RSS Fetcher Service (defaults: localhost:5434, cairn_rss / cairn_rss_pass)
docker run -d --name cairn-test-rss -p 5434:5432 \
  -e POSTGRES_USER=cairn_rss -e POSTGRES_PASSWORD=cairn_rss_pass \
  postgres:18-alpine
```

Remove them with `docker rm -f cairn-test-content cairn-test-rss` when finished. The User Service integration tests read `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD` and `DB_NAME` (defaults: `localhost:5433`, `cairn_test` / `test_password`, `cairn_users_test`; see `services/users/test/integration/testutil/setup.go`).

#### Option 2: Local PostgreSQL

If you have PostgreSQL running locally, use a role that can create databases and set environment variables:

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
```

Alternatively, point the tests at the dev `cairn-db` (port 5432) with `TEST_DB_PORT=5432` and the `POSTGRES_ADMIN_USER` / `POSTGRES_ADMIN_PASSWORD` values from your dev `.env` as `TEST_DB_USER` / `TEST_DB_PASSWORD`.

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
# Start databases (see Prerequisites > Database Requirements)
docker start cairn-test-content cairn-test-rss

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

E2E tests validate complete workflows across multiple services. No automated cross-service E2E suite is currently maintained.

---

## Service-Specific Testing

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
   docker compose logs cairn-db
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

# Coverage report
go test -cover ./...
go tool cover -html=coverage.out
```

For service-specific details, see:
- [Read Service CLAUDE.md](/services/read/CLAUDE.md)
- [User Service CLAUDE.md](/services/users/CLAUDE.md)

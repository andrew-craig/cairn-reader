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
  - [Self-host binary and shared packages](#self-host-binary-and-shared-packages)
  - [Web and Mobile Apps](#web-and-mobile-apps)
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
- **Dependencies**: None (mocked), except the User Service `internal/database` and `internal/services` tests, which use a real PostgreSQL (`TEST_DB_*`) and skip when it is unreachable
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

Each Read service's `internal/testutil/database.go` defaults to its own port, user and password (Content: 5433, Fetcher: 5434, Email: 5435). Start one container per service under test, using the defaults as the superuser credentials:

```bash
# Content Service (defaults: localhost:5433, cairn_content / cairn_content_pass)
docker run -d --name cairn-test-content -p 5433:5432 \
  -e POSTGRES_USER=cairn_content -e POSTGRES_PASSWORD=cairn_content_pass \
  postgres:18-alpine

# RSS Fetcher Service (defaults: localhost:5434, cairn_rss / cairn_rss_pass)
docker run -d --name cairn-test-rss -p 5434:5432 \
  -e POSTGRES_USER=cairn_rss -e POSTGRES_PASSWORD=cairn_rss_pass \
  postgres:18-alpine

# Email Ingest Service (defaults: localhost:5435, cairn_email / cairn_email_pass)
docker run -d --name cairn-test-email -p 5435:5432 \
  -e POSTGRES_USER=cairn_email -e POSTGRES_PASSWORD=cairn_email_pass \
  postgres:18-alpine
```

Remove them with `docker rm -f cairn-test-content cairn-test-rss cairn-test-email` when finished. The User Service has two different test setups: its `internal/` tests that need a database read `TEST_DB_*` (defaults: `localhost:5432`, `postgres` / `postgres`, `cairn_test`, already migrated; they skip when the database is unreachable), while its Docker-based integration tests read `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD` and `DB_NAME` (defaults: `localhost:5433`, `cairn_test` / `test_password`, `cairn_users_test`; see `services/users/test/integration/testutil/setup.go` and the [integration README](/services/users/test/integration/README.md)).

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

# For Email Ingest Service
export TEST_DB_HOST=localhost
export TEST_DB_PORT=5435
export TEST_DB_USER=cairn_email
export TEST_DB_PASSWORD=cairn_email_pass
export TEST_DB_NAME=cairn_email_test
```

`TEST_DB_SSLMODE` (default `disable`) is also read. `TEST_DB_NAME` is only the prefix of the generated database name.

Alternatively, point the tests at the dev `cairn-db` (port 5432) with `TEST_DB_PORT=5432` and the `POSTGRES_ADMIN_USER` / `POSTGRES_ADMIN_PASSWORD` values from your dev `.env` as `TEST_DB_USER` / `TEST_DB_PASSWORD`.

### Additional Tools

- **Docker**: For running databases and services
- **Make**: For running test commands
- **Go 1.25+**: For running Go tests (see `go.mod`)
- **Node/npm**: For the web, mobile and shared package tests
- **curl/jq**: For E2E testing

---

## Running Tests

### Unit Tests

Unit tests run without external dependencies and are the default test mode. The Go code is split into separate modules (`services/read` containing `content` and `fetcher`, `services/read/email`, `services/users`, `cmd/selfhost`, and each `pkg/*` module), so run `go test` from inside the module you are testing.

```bash
# Run all unit tests for a module
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
# Read services (content, fetcher and email; make test-content / test-fetcher / test-email for one)
cd services/read
make test

# User service (go test -v ./internal/...)
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
go test -tags=integration -v -run TestFeedSubscriptionIntegration ./...

# Run with coverage
go test -tags=integration -coverprofile=coverage.out -covermode=atomic ./...

# View coverage in browser
go tool cover -html=coverage.out
```

**Examples**:

```bash
# Content Service integration tests
cd services/read/content
go test -tags=integration -v ./...

# RSS Fetcher Service integration tests
cd services/read/fetcher
go test -tags=integration -v ./...

# Email Ingest Service integration tests (separate Go module)
cd services/read/email
go test -tags=integration -v ./...
```

**Skip Integration Tests**:

```bash
# Just run unit tests (default; the integration build tag is off)
go test ./...

# With the tag on, -short skips tests that call testing.Short() (the fetcher and email lease tests, among others)
go test -tags=integration -short ./...

# Skip the tests that call real external sites (URL detection)
CAIRN_SKIP_NETWORK_TESTS=1 go test -tags=integration ./...
```

### End-to-End Tests

E2E tests validate complete workflows across multiple services. No automated cross-service E2E suite is currently maintained.

---

## Service-Specific Testing

### Read Service

The Read service consists of three microservices: Content Service (`services/read/content`), RSS Fetcher Service (`services/read/fetcher`) and Email Ingest Service (`services/read/email`, its own Go module). Integration tests carry the `integration` build tag and create their database with the service's `internal/testutil` package (see [Test Helpers and Utilities](#test-helpers-and-utilities)).

#### Integration Test Overview

Integration tests cover the database-backed repository, service and handler layers, plus a few end-to-end flows.

**Test Coverage**:
- Feed/Reads routing: list resolved from `source_routes`, re-routing affects new items only, list filter, moving items between lists, bulk ingest routing by user and source
- Feed retention (deletion of expired non-favorited Feed items) and orphaned-content cleanup batching
- Content deduplication under concurrent create and bulk create, and repeat email delivery
- Migration tests (Feed/Reads routing up/down, dedup backfill)
- RSS feed subscription and management, including the 100 feeds per user limit
- Outbox pattern delivery flow (fetcher and email)
- RSS item processing and deduplication
- Feed polling logic and error tracking
- Lease behaviour of the pending-work queries (crash recovery, exclusivity, retry reselection) for feeds, feed items, outbox entries and raw emails
- JWT auth and Vault JWT key loading in the Content Service (the Vault test needs `VAULT_ADDR`, and skips without it)

#### Content Service Integration Tests

Located under `services/read/content`:

- `integration_test.go`: `TestJWTAuthIntegration`, `TestVaultJWTIntegration`
- `internal/repository/*_integration_test.go`: routing (`routing_integration_test.go`), dedup (`content_dedup_integration_test.go`), cleanup batching, Feed retention
- `internal/service/*_integration_test.go`: bulk create and email redelivery, plus URL detection (`url_detector_integration_test.go`)
- `internal/api/handlers/*_integration_test.go`: bulk ingest routing and URL detection
- `migrations/*_integration_test.go`: routing migration up/down and the dedup backfill

The URL-detection tests call real external sites; set `CAIRN_SKIP_NETWORK_TESTS=1` to skip them (CI does).

#### RSS Fetcher Service Integration Tests

Located in `services/read/fetcher/integration_test.go` (the scenarios below) plus the lease tests in `internal/repository/*_lease_integration_test.go`. Two further tests there, `TestContentServiceReceivesRSSTitleAndAuthor` and `TestRawHTMLPipelineIntegration`, exercise the fetcher-to-content hand-off using `httptest` servers.

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
- Retry count increment and next-retry scheduling

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
- Track consecutive error days (repository-level counter; auto-disable lives in the service layer and is not covered here)

#### Running Read Service Integration Tests

Start the throwaway PostgreSQL containers from [Prerequisites](#database-requirements) first (the dev `cairn-db` users cannot create databases).

```bash
# Content Service integration tests
cd services/read/content
go test -tags=integration -v ./...

# RSS Fetcher Service integration tests
cd services/read/fetcher
go test -tags=integration -v ./...

# Email Ingest Service integration tests
cd services/read/email
go test -tags=integration -v ./...

# Run with coverage
go test -tags=integration -coverprofile=coverage.out -covermode=atomic ./...
go tool cover -html=coverage.out
```

---

### User Service

The User service handles authentication and user management.

#### Test Coverage

**Unit Tests** (`internal/...`, `cmd/user-service`, `selfhost`):
- JWT token generation and validation
- Refresh token rotation
- Password hashing and verification
- Device authentication

The `internal/database` and `internal/services` tests (and parts of `internal/handlers`) need a migrated PostgreSQL reachable via `TEST_DB_*` (defaults `localhost:5432`, `postgres` / `postgres`, `cairn_test`) and skip when it is not. CI provides one and runs `go run cmd/migrate/main.go -command=up` first.

**Integration Tests** (`test/integration`, `integration` build tag, own Docker Compose environment with PostgreSQL and Vault):
- Full authentication flow (register → login → refresh)
- Logout, token rotation and reuse
- Account upgrade (device-only → email/password)
- User get/update/delete

**Security scripts** (`test/security`, bash against a running service): see the [security testing README](/services/users/test/security/README.md).

Details: [integration README](/services/users/test/integration/README.md).

#### Running User Service Tests

```bash
cd services/users

# Unit tests only (go test -v ./internal/...)
make test

# Integration tests: start the test environment (PostgreSQL on 5433, Vault on 8201), then run with the tag
make test-integration-up
cd test/integration && go test -tags=integration -v -timeout 5m ./...
cd ../.. && make test-integration-down

# Unit tests with coverage (coverage.html)
make test-coverage

# Security scripts (needs the service running at USER_SERVICE_URL)
make test-security
```

---

### Self-host binary and shared packages

```bash
# Consolidated self-host binary (config, health, router recovery tests)
cd cmd/selfhost
go test ./...

# Its integration tests (tag `integration`; TEST_DB_HOST/PORT/USER/PASSWORD/SSLMODE,
# defaults localhost:5432 postgres/postgres; creates and drops its own databases)
go test -tags=integration -v ./...

# Shared Go modules, each with its own go.mod
cd pkg/auth && go test ./...        # also pkg/config, pkg/logging, pkg/middleware, pkg/rss
```

`services/users/selfhost` and the other `services/*/selfhost` packages carry their own unit tests and run with their service module.

### Web and Mobile Apps

The npm workspace root (`package.json`) contains `apps/web`, `apps/shared` and `apps/mobile`; run `npm ci` once from the repo root.

```bash
# Web app: Vitest + jsdom + Testing Library (apps/web/vitest.config.ts, vitest.setup.ts)
cd apps/web && npm test            # vitest run

# Shared package: Vitest in plain Node (apps/shared/vitest.config.ts)
cd apps/shared && npm test

# Mobile app: Jest with the jest-expo preset (apps/mobile/jest.config.js, jest.setup.ts)
cd apps/mobile && npm test         # npm run test:ci for jest --ci --coverage
```

Tests are colocated as `*.test.ts(x)` next to the source. There is no Playwright or other automated browser/device end-to-end suite in the repo. See [Web App](/apps/web/CLAUDE.md#testing) and [Mobile App](/apps/mobile/CLAUDE.md#testing) for conventions and patterns.

---

## Test Helpers and Utilities

### Database Test Helpers

The Content, RSS Fetcher and Email Ingest services each include a `testutil` package (`internal/testutil/`) with the same utilities for integration tests. The User Service uses its own `test/integration/testutil` package instead (see its README).

#### SetupTestDatabase

Creates a unique test database for each test run.

```go
func SetupTestDatabase(t *testing.T) *TestDatabase
```

**Process**:
1. Connects to the main PostgreSQL instance
2. Creates a temporary database with timestamp suffix (e.g., `cairn_content_test_1234567890`)
3. Applies every `*up.sql` file found in the service's `migrations/` directory (a missing directory only logs a warning)
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

Truncates the service's tables (a fixed list per service, with `CASCADE`); faster than recreation for sequential tests.

```go
func (td *TestDatabase) TruncateAll()
```

### Example Test Structure

Integration test files start with `//go:build integration`. From `services/read/fetcher/integration_test.go`:

```go
func TestFeedSubscriptionIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    // Setup
    testDB := testutil.SetupTestDatabase(t)
    defer testDB.Cleanup()

    feedRepo := repository.NewFeedRepository(testDB.DB)
    subscriptionRepo := repository.NewFeedSubscriptionRepository(testDB.DB)
    feedService := service.NewFeedService(feedRepo, subscriptionRepo)
    router := setupTestRouter(handlers.NewSubscriptionHandler(feedService))
    server := httptest.NewServer(router)
    defer server.Close()

    // Run tests
    t.Run("ListUserFeeds", func(t *testing.T) {
        resp, err := http.Get(fmt.Sprintf("%s/api/v1/users/%s/feeds", server.URL, userID))
        require.NoError(t, err)
        defer resp.Body.Close()
        assert.Equal(t, http.StatusOK, resp.StatusCode)
    })
}
```

---

## CI/CD Integration

Workflows live in `.github/workflows/`; test-related ones are:

| Workflow | What it runs |
|----------|--------------|
| `go-checks.yml` | Per module (path-filtered): gofmt, golangci-lint, `go vet`, `go build`, and `go test -race -count=1` with coverage for `services/read` (content, fetcher), `services/read/email`, `services/users` (`./internal/...` with `-p 1` against a PostgreSQL service container and migrations applied first), `cmd/selfhost` and the `pkg/*` modules |
| `go-checks.yml` (integration jobs) | `go test -race -tags=integration` for content, fetcher and email (PostgreSQL 16 service, `TEST_DB_*` pointing at it, `CAIRN_SKIP_NETWORK_TESTS=1`) and for `cmd/selfhost` |
| `web-checks.yml` | `tsc --noEmit`, ESLint, `npx vitest run` for `apps/web`, and `tsc` + `vitest run` for `apps/shared` |
| `mobile-checks.yml` | `tsc --noEmit`, ESLint, `npx jest --ci --coverage`, an Expo iOS bundle export, and `knip` (dead code) |
| `selfhost-compose-smoke.yml` | Builds the self-host image, starts `infrastructure/docker/selfhost` with Docker Compose and checks `/health/live` |
| `docker-test.yml` | Builds each service image (no push) as a smoke test |

The User Service Docker-based integration tests (`services/users/test/integration`) and the bash security scripts are not run in CI.

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

1. Ensure migration files exist in `services/read/{content,fetcher,email}/migrations/` (or `services/users/migrations/`)

2. Check migration file paths in test helper

3. Verify migrations work manually:
   ```bash
   migrate -path ./migrations -database "postgres://cairn_content:cairn_content_pass@localhost:5433/cairn_content_test?sslmode=disable" up
   ```

4. Check for syntax errors in migration SQL:
   ```bash
   cat services/read/content/migrations/*.up.sql
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

**Problem**: User service integration tests fail with Vault connection errors

**Solutions**:

Note: the User Service integration environment has its own Vault on port 8201 (token `test-root-token`); see the [integration README](/services/users/test/integration/README.md). The commands below are for the dev stack Vault on 8200.

1. Ensure Vault is running:
   ```bash
   cd infrastructure/docker/dev
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

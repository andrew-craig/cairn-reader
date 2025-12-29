# Cairn Backend - Implementation Plan

## Overview

This document outlines the implementation plan for the initial scope of Cairn, a read-it-later backend system. The implementation is organized into phases to ensure systematic development and testing.

## Architecture Summary

- **Content Service**: Stores and serves article content with user-specific metadata
- **RSS Fetcher Service**: Polls RSS feeds and delivers content to subscribed users
- **Communication**: REST APIs exclusively (no direct database access between services)
- **Database**: Separate PostgreSQL databases for each service
- **Deployment**: Docker Compose for containerized deployment

## Implementation Phases

### Phase 0: Project Setup & Infrastructure

**Objective**: Set up the foundational project structure and development environment.

#### Tasks

1. **Repository Structure**
   - Create Go module structure for monorepo
   - Set up directory layout:
     ```
     /services/content-service/
     /services/rss-fetcher-service/
     /pkg/shared/ (for common utilities if needed)
     /deployments/docker/
     /scripts/
     /docs/
     ```

2. **Database Setup & Migrations**
   - Set up `golang-migrate/migrate` for database migrations
   - Create migration directory structure:
     ```
     /services/content-service/migrations/
       000001_initial_schema.up.sql
       000001_initial_schema.down.sql
     /services/rss-fetcher-service/migrations/
       000001_initial_schema.up.sql
       000001_initial_schema.down.sql
     ```
   - Create initial migration files from schema definitions
   - Add migration commands to Makefile:
     - `make migrate-up` - Apply all pending migrations
     - `make migrate-down` - Rollback last migration
     - `make migrate-create name=<name>` - Create new migration files
     - `make migrate-status` - Show current migration status
   - Configure migration to run on service startup (optional, configurable)
   - Document migration workflow in README

3. **Docker Configuration**
   - Create Dockerfile for Content Service
   - Create Dockerfile for RSS Fetcher Service
   - Create docker-compose.yml with:
     - Content Service container
     - RSS Fetcher Service container
     - PostgreSQL container for Content Service
     - PostgreSQL container for RSS Fetcher Service
     - Proper networking between services

4. **Development Tools**
   - Set up Go modules (`go.mod`) for each service
   - Configure linting (golangci-lint)
   - Set up basic Makefile for common tasks
   - Create `.env.example` files for configuration

**Dependencies**:
```go
github.com/golang-migrate/migrate/v4
```

**Deliverables**:
- Working Docker Compose environment
- Database migrations with golang-migrate
- Makefile with migration commands
- Build and deployment scripts

---

### Phase 1: Content Service - Core Functionality

**Objective**: Build the core Content Service with content storage, retrieval, and user-content relationship management.

#### 1.1 Database Layer

**Tasks**:
1. Implement database connection pooling
2. Create data models (structs) for:
   - `Content`
   - `UserContent`
3. Implement repository pattern for database operations:
   - `ContentRepository` interface and implementation
   - `UserContentRepository` interface and implementation
4. Add database helper functions for transactions
5. Deploy schema with triggers for orphaned content tracking

**Key Files**:
- `services/content-service/internal/repository/content.go`
- `services/content-service/internal/repository/user_content.go`
- `services/content-service/internal/models/models.go`
- `services/content-service/internal/database/connection.go`

#### 1.2 Content Processing

**Tasks**:
1. Integrate `go-shiori/go-readability` for HTML cleaning
2. Integrate `bluemonday` for HTML sanitization
3. Implement content processing pipeline:
   - Fetch URL content (with timeout and error handling)
   - Apply readability parsing (with fallback to raw HTML)
   - Sanitize HTML output
   - Generate SHA-256 content hash
   - Validate content size (5MB limit)
4. Create service layer for content operations:
   - `ContentService` interface and implementation
   - Content creation/update logic
   - Deduplication logic (content_hash + source_feed_id)
5. Implement URL canonicalization utility (for future use):
   - Normalize scheme (lowercase, prefer https)
   - Normalize host (lowercase)
   - Remove common tracking parameters (utm_*, fbclid, etc.)
   - Consistent trailing slash handling
   - Store result in `canonical_url` column

**Key Files**:
- `services/content-service/internal/processor/readability.go`
- `services/content-service/internal/processor/sanitizer.go`
- `services/content-service/internal/processor/content.go`
- `services/content-service/internal/processor/url_canonicalizer.go`
- `services/content-service/internal/service/content_service.go`

**Dependencies**:
```go
github.com/go-shiori/go-readability
github.com/microcosm-cc/bluemonday
```

#### 1.3 REST API - Basic Operations

**Tasks**:
1. Set up HTTP server with routing (use `chi` or `gorilla/mux`)
2. Implement API endpoints:
   - `POST /api/v1/contents` - Create content
   - `PUT /api/v1/contents/:id` - Update content (preserves user-content relationships)
   - `GET /api/v1/contents/:id` - Get content by ID
   - `GET /api/v1/users/:user_id/contents` - List user's contents
     - Query params: status, is_favorite, limit, cursor
     - Cursor-based pagination (20 items per page)
   - `POST /api/v1/users/:user_id/contents` - Add content to user's list
   - `PATCH /api/v1/users/:user_id/contents/:content_id` - Update user-content metadata
     - Update status, scroll_position, is_favorite
   - `DELETE /api/v1/users/:user_id/contents/:content_id` - Remove content from user's list
   - `GET /api/v1/users/:user_id/contents/search` - Search user's contents
     - Query param: q (search in title and author)
     - Uses PostgreSQL full-text search with GIN index for performance
3. Implement request/response DTOs
4. Add input validation middleware
5. Add error handling middleware
6. Add logging middleware

**Key Files**:
- `services/content-service/internal/api/handlers/content_handler.go`
- `services/content-service/internal/api/handlers/user_content_handler.go`
- `services/content-service/internal/api/middleware/`
- `services/content-service/internal/api/dto/`
- `services/content-service/cmd/server/main.go`

#### 1.4 REST API - Bulk Operations

**Tasks**:
1. Implement bulk endpoints for RSS Fetcher:
   - `POST /api/v1/contents/bulk` - Batch create/update contents (max 100 items)
   - `POST /api/v1/contents/check-duplicates` - Check for existing content (by feed_id + content_hash)
   - `POST /api/v1/users/bulk/contents` - Batch add contents to multiple users
2. Implement efficient batch processing with transactions
3. Add rate limiting considerations

**Key Files**:
- `services/content-service/internal/api/handlers/bulk_handler.go`

**Deliverables**:
- Functional Content Service with all required APIs
- Content processing with readability and sanitization
- Deduplication working correctly
- Comprehensive unit tests for service layer
- Integration tests for API endpoints

---

### Phase 2: RSS Fetcher Service - Feed Management

**Objective**: Build the RSS Fetcher Service with feed subscription management and polling infrastructure.

#### 2.1 Database Layer

**Tasks**:
1. Implement database connection pooling
2. Create data models (structs) for:
   - `Feed`
   - `FeedSubscription`
   - `FeedItem`
3. Implement repository pattern:
   - `FeedRepository` interface and implementation
   - `FeedSubscriptionRepository` interface and implementation
   - `FeedItemRepository` interface and implementation
4. Deploy schema with feed limit trigger

**Key Files**:
- `services/rss-fetcher-service/internal/repository/feed.go`
- `services/rss-fetcher-service/internal/repository/subscription.go`
- `services/rss-fetcher-service/internal/repository/feed_item.go`
- `services/rss-fetcher-service/internal/models/models.go`
- `services/rss-fetcher-service/internal/database/connection.go`

#### 2.2 Feed Subscription API

**Tasks**:
1. Set up HTTP server with routing
2. Implement feed subscription endpoints:
   - `POST /api/v1/users/:user_id/feeds/subscribe` - Subscribe to a feed
     - Validate feed URL
     - Check 100 feed limit per user
     - Create feed if new, or link to existing feed
   - `DELETE /api/v1/users/:user_id/feeds/:feed_id` - Unsubscribe from feed
   - `GET /api/v1/users/:user_id/feeds` - List user's feed subscriptions
   - `PATCH /api/v1/feeds/:feed_id/enable` - Re-enable a disabled feed
3. Implement feed validation:
   - Fetch feed URL and verify it's valid RSS/Atom
   - Extract feed metadata (title, description, site_url)
4. Add error handling and validation

**Key Files**:
- `services/rss-fetcher-service/internal/api/handlers/subscription_handler.go`
- `services/rss-fetcher-service/internal/service/feed_service.go`
- `services/rss-fetcher-service/cmd/server/main.go`

**Dependencies**:
```go
github.com/mmcdole/gofeed (RSS/Atom parser)
```

#### 2.3 Content Service Client

**Tasks**:
1. Create HTTP client for Content Service API
2. Implement client methods:
   - `CreateContent(content)` - Create single content
   - `UpdateContent(contentID, content)` - Update existing content (preserves user relationships)
   - `BulkCreateContent(contents)` - Batch create up to 100 items
   - `CheckDuplicates(feedID, contentHashes)` - Check for existing content
   - `AddContentToUsers(contentID, userIDs)` - Add content to multiple users
3. Add retry logic with exponential backoff
4. Add timeout configuration
5. Add circuit breaker pattern for resilience

**Key Files**:
- `services/rss-fetcher-service/internal/client/content_service_client.go`

**Deliverables**:
- Feed subscription management working
- Integration with Content Service via REST API
- Feed validation and metadata extraction
- Unit tests for subscription logic

---

### Phase 3: RSS Fetcher Service - Polling & Processing

**Objective**: Implement the feed polling scheduler and content extraction pipeline.

#### 3.1 Feed Polling Scheduler

**Tasks**:
1. Implement tiered polling strategy:
   - Tier 1 (Active): Poll every 1 hour
   - Tier 2 (Moderate): Poll every 6 hours
   - Tier 3 (Quiet): Poll every 24 hours
2. Create background job to:
   - Query feeds where `next_poll_at <= NOW()` and `status = 'active'`
   - Sort by `next_poll_at` ASC
   - Process feeds in batches
3. Implement tier management:
   - Daily background job to evaluate and update tiers based on `last_published_at`
   - Promote tiers when new content is published
   - Demote tiers based on inactivity
4. Update `next_poll_at` after each fetch based on current tier
5. Implement concurrent processing with worker pool pattern

**Key Files**:
- `services/rss-fetcher-service/internal/scheduler/poll_scheduler.go`
- `services/rss-fetcher-service/internal/scheduler/tier_manager.go`
- `services/rss-fetcher-service/internal/worker/feed_worker.go`

#### 3.2 RSS Parsing & Item Processing

**Tasks**:
1. Implement feed fetching:
   - HTTP client with timeout
   - Handle redirects
   - Verify SSL certificates
   - Parse RSS/Atom XML
2. Implement error handling:
   - Track consecutive error days
   - Auto-disable feed after 7 consecutive days of errors
   - Update `last_error_at` and `last_error_message`
3. Extract feed items:
   - Parse item metadata (title, author, published_at, description, URL, GUID)
   - Check for duplicates using `feed_id + item_guid`
   - Store new items in `feed_items` table with status 'pending'
4. Update feed metadata after successful fetch:
   - Update `last_fetched_at`
   - Update `last_published_at` if new items found
   - Reset `consecutive_error_days` to 0

**Key Files**:
- `services/rss-fetcher-service/internal/fetcher/feed_fetcher.go`
- `services/rss-fetcher-service/internal/fetcher/parser.go`
- `services/rss-fetcher-service/internal/processor/item_processor.go`

#### 3.2.1 Content Update Detection

**Tasks**:
1. Track HTTP caching headers during article fetch:
   - Store `Last-Modified` header in `feed_items.http_last_modified`
   - Store `ETag` header in `feed_items.http_etag`
   - Update `last_checked_at` timestamp
2. Implement conditional requests for existing items:
   - Send `If-Modified-Since` header with stored `last_modified` value
   - Send `If-None-Match` header with stored `etag` value
   - Handle 304 Not Modified responses (skip processing)
3. Detect content changes:
   - Re-fetch article content when headers indicate update
   - Re-process through readability and sanitization
   - Generate new content hash
   - Compare with stored hash to confirm actual change
4. Update content via Content Service:
   - Call `PUT /api/v1/contents/:id` with updated content
   - Include new content_hash, cleaned_html, metadata
   - Content Service preserves all user-content relationships
5. Update tracking data:
   - Store new `http_last_modified` and `http_etag`
   - Update `content_hash` in `feed_items`
   - Log content updates for monitoring

**Key Files**:
- `services/rss-fetcher-service/internal/processor/update_detector.go`
- `services/rss-fetcher-service/internal/fetcher/conditional_fetcher.go`

#### 3.3 Content Extraction & Outbox

**Tasks**:
1. Create background job to process pending feed items:
   - Query `feed_items` where `processing_status = 'pending'`
   - Process in batches
2. For each pending item:
   - Fetch full article content from source URL
     - Handle timeouts (30 second timeout)
     - Handle paywalls and errors (fallback to RSS description)
   - Apply readability parsing
   - Apply HTML sanitization
   - Generate content hash (SHA-256)
   - Update `feed_items` with hash and status 'processing'
3. Write to Outbox (instead of direct API call):
   - Get list of all users subscribed to the feed
   - Create `content_outbox` entry with:
     - Full content payload (ready for Content Service API)
     - Array of user IDs to deliver to
     - Initial status 'pending'
   - Update `feed_items` status to 'completed'
4. Implement retry logic for content extraction:
   - Retry failed extractions up to 3 times with exponential backoff
   - Mark as 'failed' after 3 retries

**Key Files**:
- `services/rss-fetcher-service/internal/processor/content_extractor.go`
- `services/rss-fetcher-service/internal/repository/outbox.go`
- `services/rss-fetcher-service/internal/models/outbox.go`

#### 3.4 Outbox Delivery Worker

**Tasks**:
1. Create background worker to process outbox entries:
   - Query `content_outbox` where `delivery_status = 'pending'` and `next_retry_at <= NOW()`
   - Process entries in order of `next_retry_at`
   - Use worker pool pattern for concurrent delivery
2. For each outbox entry:
   - Set status to 'sending'
   - Call Content Service bulk API to create content
   - Check for duplicates (same feed_id + content_hash)
   - Add content to all users' reading lists (status = 'unread')
3. Handle delivery results:
   - On success: Set status to 'delivered', store `content_service_id`, set `delivered_at`
   - On failure: Increment `retry_count`, calculate `next_retry_at` with exponential backoff
   - On max retries exceeded: Set status to 'failed'
4. Implement exponential backoff schedule:
   - Retry 1: 1 minute
   - Retry 2: 5 minutes
   - Retry 3: 15 minutes
   - Retry 4: 1 hour
   - Retry 5: 4 hours
   - Retry 6: 12 hours
   - After 6 retries: Mark as 'failed'
5. Add circuit breaker for Content Service:
   - Open after 5 consecutive failures
   - Half-open after 30 seconds
   - Prevents overwhelming a failing Content Service

**Key Files**:
- `services/rss-fetcher-service/internal/worker/outbox_worker.go`
- `services/rss-fetcher-service/internal/client/content_service_client.go`

**Deliverables**:
- Automatic feed polling with tiered strategy
- RSS parsing and item extraction
- Content processing with readability
- Content update detection via HTTP caching headers
- Outbox pattern for reliable content delivery
- Auto-delivery to subscribed users via outbox worker
- Circuit breaker for Content Service resilience
- Error handling and feed auto-disable
- Comprehensive logging
- Unit and integration tests

---

### Phase 4: Background Jobs & Maintenance

**Objective**: Implement cleanup jobs and maintenance tasks.

#### 4.1 Orphaned Content Cleanup

**Tasks**:
1. Create scheduled job for Content Service:
   - Run daily (e.g., at 2 AM)
   - Query contents where `orphaned_at < NOW() - INTERVAL '90 days'`
   - Delete in batches to avoid long locks
   - Log deleted content IDs
2. Add monitoring/logging for cleanup operations

**Key Files**:
- `services/content-service/internal/jobs/cleanup_job.go`
- `services/content-service/cmd/worker/main.go` (separate worker process)

#### 4.2 Feed Tier Management Job

**Tasks**:
1. Create scheduled job for RSS Fetcher Service:
   - Run daily
   - Update polling tiers based on `last_published_at`:
     - Active: Last published within 7 days
     - Moderate: Last published within 30 days (but not 7 days)
     - Quiet: Last published over 30 days ago
   - Update `polling_tier` and `next_poll_at` accordingly
2. Log tier changes for monitoring

**Key Files**:
- `services/rss-fetcher-service/internal/jobs/tier_update_job.go`

#### 4.3 Outbox Cleanup Job

**Tasks**:
1. Create scheduled job for RSS Fetcher Service:
   - Run daily (e.g., at 3 AM)
   - Delete delivered outbox entries older than 7 days
   - Archive or log failed outbox entries for investigation
   - Clean up in batches to avoid long locks
2. Add monitoring/alerting for:
   - Number of failed deliveries
   - Outbox queue depth
   - Average delivery latency

**Key Files**:
- `services/rss-fetcher-service/internal/jobs/outbox_cleanup_job.go`

#### 4.4 Feed Items Cleanup Job

**Tasks**:
1. Create scheduled job for RSS Fetcher Service:
   - Run daily (e.g., at 4 AM)
   - Delete completed `feed_items` older than 7 days
   - Delete failed `feed_items` older than 30 days
   - Clean up in batches to avoid long locks
2. Add monitoring metrics:
   - Number of items cleaned up per run
   - Current table size
   - Items by processing status

**Key Files**:
- `services/rss-fetcher-service/internal/jobs/feed_items_cleanup_job.go`

#### 4.5 Job Scheduling Infrastructure

**Tasks**:
1. Implement cron-like scheduler using `robfig/cron/v3`
2. Create worker processes separate from API servers
3. Add graceful shutdown handling
4. Add health check endpoints for worker processes

**Dependencies**:
```go
github.com/robfig/cron/v3
```

**Deliverables**:
- Automated orphaned content cleanup
- Automated feed tier updates
- Automated outbox cleanup
- Automated feed items table cleanup
- Reliable job scheduling infrastructure
- Monitoring and logging for background jobs

---

### Phase 5: Comprehensive Testing

**Objective**: Implement comprehensive test coverage across all layers of both services to ensure code quality, reliability, and maintainability.

**Status**: ✅ Completed

#### 5.1 Service Layer Tests

**Implementation**: Both Content Service and RSS Fetcher Service have complete service layer test coverage using mock dependencies.

**Content Service** (`services/content-service/internal/service/content_service_test.go`):
- Content creation from HTML and URL with various input scenarios
- Content updates with preservation of user-content relationships
- Duplicate detection logic by content hash and feed ID
- Bulk operations with partial failure handling
- Duplicate checking across multiple items
- Uses `testify/mock` to mock processor and repository dependencies

**RSS Fetcher Service** (`services/rss-fetcher-service/internal/service/feed_service_test.go`):
- Feed subscription with 100-feed limit enforcement
- Duplicate subscription prevention
- Feed URL validation and format checking
- Feed enable/disable logic with error reset
- Unsubscribe operations with proper cleanup
- Uses mocked feed and subscription repositories

**Testing Approach**: Pure unit tests with mocked dependencies to test business logic in isolation.

#### 5.2 Repository Layer Tests

**Implementation**: Complete database layer testing using `go-sqlmock` for fast, isolated unit tests.

**Test Files**:
- Content Service:
  - `services/content-service/internal/repository/content_test.go`
  - `services/content-service/internal/repository/user_content_test.go`
- RSS Fetcher Service:
  - `services/rss-fetcher-service/internal/repository/feed_test.go`
  - `services/rss-fetcher-service/internal/repository/subscription_test.go`
  - `services/rss-fetcher-service/internal/repository/feed_item_test.go`
  - `services/rss-fetcher-service/internal/repository/outbox_test.go`

**Coverage**:
- CRUD operations (Create, Read, Update, Delete)
- Complex queries (GetByContentHashAndFeedID, GetFeedsDueForPolling)
- Bulk operations (BulkCreate, GetByContentHashesAndFeedID)
- Pagination and filtering with limit/offset
- Transaction handling (CreateWithTx, UpdateWithTx)
- Error cases (constraint violations, not found errors)
- Cascade deletes and orphaned content tracking

**Testing Approach**: Using `DATA-DOG/go-sqlmock` to mock SQL queries and responses without requiring a real database.

#### 5.3 API Handler Tests

**Implementation**: HTTP endpoint testing using `httptest` with mocked service layer dependencies.

**Test Files**:
- Content Service:
  - `services/content-service/internal/api/handlers/content_handler_test.go`
  - `services/content-service/internal/api/handlers/user_content_handler_test.go`
  - `services/content-service/internal/api/handlers/bulk_handler_test.go`
- RSS Fetcher Service:
  - `services/rss-fetcher-service/internal/api/handlers/subscription_handler_test.go`

**Coverage**:
- Request validation (valid/invalid JSON, missing fields, type errors)
- Success responses with correct status codes
- Error handling (not found, validation errors, server errors)
- URL parameter parsing (UUIDs, path parameters)
- Request body parsing and DTO conversion
- Response formatting and serialization

**Key Test Scenarios**:
- Content Service: Create, update, get, list, search, bulk operations, duplicate checking
- RSS Fetcher Service: Subscribe, unsubscribe, list feeds, enable/disable feeds

**Testing Approach**: Mock the service layer and test HTTP request/response handling in isolation.

#### 5.4 Worker & Background Job Tests

**Implementation**: Comprehensive testing of background processing, scheduled jobs, and worker pools.

**Test Files**:
- Content Service:
  - `services/content-service/internal/jobs/cleanup_job_test.go`
- RSS Fetcher Service:
  - `services/rss-fetcher-service/internal/jobs/content_extraction_job_test.go`
  - `services/rss-fetcher-service/internal/jobs/feed_items_cleanup_job_test.go`
  - `services/rss-fetcher-service/internal/jobs/outbox_cleanup_job_test.go`
  - `services/rss-fetcher-service/internal/worker/outbox_worker_test.go`
  - `services/rss-fetcher-service/internal/worker/feed_worker_test.go`

**Coverage**:
- Job configuration and initialization
- Batch processing logic
- Exponential backoff retry calculations
- Worker pool concurrency patterns
- Cleanup operations (orphaned content, old outbox entries, feed items)
- Content extraction from pending feed items
- Outbox delivery with circuit breaker pattern

**Testing Approach**: Test worker logic and job execution with mocked repositories and clients.

#### 5.5 Middleware Tests

**Implementation**: Testing of HTTP middleware for validation, error handling, logging, and recovery.

**Test Files**:
- Content Service:
  - `services/content-service/internal/api/middleware/validation_test.go`
  - `services/content-service/internal/api/middleware/error_handler_test.go`
  - `services/content-service/internal/api/middleware/logging_test.go`
- RSS Fetcher Service:
  - `services/rss-fetcher-service/internal/api/middleware/logging_test.go`
  - `services/rss-fetcher-service/internal/api/middleware/recovery_test.go`

**Coverage**:
- Request validation (JSON parsing, required fields, type validation)
- Error response formatting and status code mapping
- Request/response logging with timing
- Panic recovery with stack trace logging

**Testing Approach**: Test middleware in isolation using `httptest` and mock handlers.

#### 5.6 Integration Tests

**Implementation**: End-to-end testing of complete request flows from HTTP to database.

**Test Files**:
- `services/content-service/integration_test.go`
- `services/rss-fetcher-service/integration_test.go`

**Coverage**:
- Full content creation flow (HTTP → Handler → Service → Repository → Database)
- Content retrieval and filtering operations
- User-content relationship management
- Feed subscription and management flows
- Database state verification
- Content sanitization validation
- Search functionality with PostgreSQL full-text search

**Testing Approach**:
- Uses `testhelpers.SetupTestDatabase()` for test database setup
- Runs migrations before tests
- Creates real HTTP test server with `httptest`
- Verifies database state after operations
- Tagged with `// +build integration` for selective execution
- Can be skipped with `go test -short`

#### 5.7 RSS Fetcher & Parser Tests

**Implementation**: Comprehensive testing of RSS/Atom feed fetching, parsing, and content extraction.

**Test Files**:
- `services/rss-fetcher-service/internal/fetcher/feed_fetcher_test.go`
- `services/rss-fetcher-service/internal/fetcher/conditional_fetcher_test.go`
- `services/rss-fetcher-service/internal/fetcher/parser_test.go`

**Coverage**:
- Feed fetching with various success and error scenarios
- HTTP redirect handling (up to 10 redirects)
- Timeout handling with configurable duration
- Error tracking and consecutive error day increments
- Feed metadata updates (last_fetched_at, last_published_at)
- Conditional requests (If-Modified-Since, If-None-Match headers)
- 304 Not Modified response handling
- Content update detection via ETag comparison
- RSS/Atom XML parsing with sample feeds
- Feed item extraction and metadata parsing

**Testing Approach**: Uses HTTP test servers and mock responses to simulate various feed scenarios.

#### 5.8 Content Processing Tests

**Implementation**: Testing of HTML processing, sanitization, and canonicalization.

**Test Files**:
- `services/content-service/internal/processor/content_test.go`
- `services/content-service/internal/processor/sanitizer_test.go`
- `services/content-service/internal/processor/url_canonicalizer_test.go`

**Coverage**:
- Readability extraction from HTML content
- HTML sanitization with bluemonday (script removal, safe tags)
- Content hash generation (SHA-256)
- URL canonicalization (scheme normalization, tracking parameter removal)
- Content size validation (5MB limit)
- Fallback handling for processing failures

**Testing Approach**: Unit tests with sample HTML and URL inputs.

#### 5.9 Content Service Client Tests

**Implementation**: Testing of HTTP client used by RSS Fetcher to communicate with Content Service.

**Test Files**:
- `services/rss-fetcher-service/internal/client/content_service_client_test.go`

**Coverage**:
- Content creation and update operations
- Bulk content creation with batching
- Duplicate checking across multiple items
- Retry logic with exponential backoff
- Circuit breaker for service resilience
- Timeout handling
- Error response parsing

**Testing Approach**: Mock HTTP server to simulate Content Service responses.

**Tools & Dependencies**:

```go
// Testing libraries
github.com/stretchr/testify/assert      // Assertions
github.com/stretchr/testify/mock        // Mocking
github.com/stretchr/testify/require     // Required assertions
github.com/DATA-DOG/go-sqlmock         // SQL mocking
```

**Testing Best Practices Implemented**:

1. **Test Naming**: `TestComponentName_MethodName_Scenario` format
2. **Table-Driven Tests**: Used for testing multiple input variations
3. **Arrange-Act-Assert**: Clear test structure
4. **Mock Assertions**: Consistent use of `AssertExpectations(t)`
5. **Test Isolation**: Independent and idempotent tests
6. **Parallel Execution**: `t.Parallel()` used where appropriate
7. **Integration Tags**: Build tags for selective test execution

**Coverage Achievement**:
- Service layer: Comprehensive coverage of business logic
- Repository layer: Complete CRUD and query operation coverage
- API handlers: Full request/response flow coverage
- Workers & jobs: Background processing logic covered
- Middleware: Security and validation coverage
- Integration tests: End-to-end flow validation
- **Target**: 80%+ code coverage across critical paths ✅

**Deliverables**:
- ✅ 32 test files with comprehensive coverage
- ✅ Service layer tests with mocked dependencies
- ✅ Repository tests with sqlmock
- ✅ API handler tests with httptest
- ✅ Worker and background job tests
- ✅ Middleware tests for security and validation
- ✅ Integration tests for end-to-end validation
- ✅ RSS fetcher and parser tests
- ✅ Content processing tests
- ✅ HTTP client tests with retry logic

---

### Phase 6: API & Deployment Documentation

**Objective**: Create comprehensive documentation for APIs, deployment, and operations.

**Status**: 🔲 Not Started

#### 6.1 API Documentation

**Tasks**:
1. Document all REST API endpoints:
   - Request/response schemas
   - Error responses
   - Example requests
2. Create OpenAPI/Swagger specifications:
   - `services/content-service/api/openapi.yaml`
   - `services/rss-fetcher-service/api/openapi.yaml`
3. Add API documentation UI (Swagger UI or similar)

**Tools**:
- `swaggo/swag` for generating Swagger docs from code comments

**Key Files to Document**:

Content Service Endpoints:
- `POST /api/v1/contents` - Create content
- `PUT /api/v1/contents/:id` - Update content
- `GET /api/v1/contents/:id` - Get content by ID
- `GET /api/v1/users/:user_id/contents` - List user's contents
- `POST /api/v1/users/:user_id/contents` - Add content to user's list
- `PATCH /api/v1/users/:user_id/contents/:content_id` - Update user-content metadata
- `DELETE /api/v1/users/:user_id/contents/:content_id` - Remove content from user's list
- `GET /api/v1/users/:user_id/contents/search` - Search user's contents
- `POST /api/v1/contents/bulk` - Batch create/update contents
- `POST /api/v1/contents/check-duplicates` - Check for existing content

RSS Fetcher Service Endpoints:
- `POST /api/v1/users/:user_id/feeds/subscribe` - Subscribe to a feed
- `DELETE /api/v1/users/:user_id/feeds/:feed_id` - Unsubscribe from feed
- `GET /api/v1/users/:user_id/feeds` - List user's feed subscriptions
- `PATCH /api/v1/feeds/:feed_id/enable` - Re-enable a disabled feed

#### 6.2 Deployment Documentation

**Tasks**:
1. Create comprehensive README.md:
   - Project overview
   - Architecture diagram
   - Prerequisites
   - Installation steps
   - Running with Docker Compose
   - Configuration reference
2. Create deployment guide:
   - Environment variables reference
   - Database setup and migration steps
   - Scaling considerations
   - Monitoring and logging setup
3. Create troubleshooting guide:
   - Common issues and solutions
   - Debugging tips
   - Log analysis
4. Document background jobs and workers:
   - Job schedules and purposes
   - Worker pool configurations
   - Monitoring job health

**Documentation Files**:
- `README.md` - Main project documentation
- `docs/DEPLOYMENT.md` - Deployment guide
- `docs/CONFIGURATION.md` - Configuration reference
- `docs/TROUBLESHOOTING.md` - Troubleshooting guide
- `docs/ARCHITECTURE.md` - Architecture overview

**Content to Include**:

Environment Variables:
- Database connection strings
- Service URLs
- Worker configurations
- Logging levels
- Migration settings

Database Management:
- Running migrations: `make migrate-up`
- Rolling back: `make migrate-down`
- Creating new migrations: `make migrate-create name=<name>`
- Checking migration status: `make migrate-status`

Docker Compose:
- Starting services: `docker-compose up -d`
- Viewing logs: `docker-compose logs -f`
- Stopping services: `docker-compose down`
- Database access: `docker-compose exec postgres-content psql -U cairn`

**Deliverables**:
- Complete OpenAPI/Swagger specifications for both services
- API documentation UI accessible via HTTP
- Comprehensive README with getting started guide
- Deployment documentation with step-by-step instructions
- Configuration reference for all environment variables
- Troubleshooting guide for common issues
- Architecture documentation with diagrams

---

### Phase 7: Polish & Production Readiness

**Objective**: Final polish, performance optimization, and production preparation.

**Status**: 🔲 Not Started

#### 7.1 Performance Optimization

**Tasks**:
1. Database query optimization:
   - Analyze slow queries with `EXPLAIN ANALYZE`
   - Add missing indexes if needed
   - Optimize pagination queries
2. API response optimization:
   - Implement connection pooling tuning
   - Review and optimize JSON serialization
   - Add database query result caching where appropriate
3. RSS Fetcher optimization:
   - Tune worker pool sizes
   - Optimize batch sizes for bulk operations
   - Review memory usage for large feeds

#### 7.2 Observability

**Tasks**:
1. Structured logging:
   - Use `zap` or `logrus` for structured logging
   - Add request IDs for tracing
   - Log levels: DEBUG, INFO, WARN, ERROR
2. Metrics collection:
   - HTTP request metrics (duration, status codes)
   - Database query metrics
   - Feed polling metrics (success/failure rates)
   - Content processing metrics
   - Use Prometheus format
3. Health check endpoints:
   - `/health/live` - Liveness check
   - `/health/ready` - Readiness check (includes DB connection)

**Dependencies**:
```go
go.uber.org/zap
github.com/prometheus/client_golang
```

#### 7.3 Configuration Management

**Tasks**:
1. Centralize configuration using environment variables
2. Create configuration structs with validation
3. Support configuration files (YAML/JSON) for local development
4. Add sensible defaults
5. Document all configuration options

**Dependencies**:
```go
github.com/spf13/viper
```

#### 7.4 Security Hardening

**Tasks**:
1. Add rate limiting middleware (prevent abuse)
2. Add request timeout middleware
3. Validate all user inputs
4. Review HTML sanitization rules
5. Add CORS configuration
6. Add security headers
7. Review and document security considerations

#### 7.5 Docker & Deployment

**Tasks**:
1. Optimize Dockerfiles:
   - Multi-stage builds
   - Minimize image sizes
   - Non-root user
2. Update docker-compose.yml:
   - Add health checks
   - Add restart policies
   - Configure resource limits
   - Add volumes for persistence
3. Create production-ready docker-compose:
   - `docker-compose.prod.yml`
   - Environment-specific configuration
4. Add database backup strategy

**Deliverables**:
- Optimized performance
- Production-ready logging and metrics
- Secure configuration management
- Production-ready Docker images
- Deployment ready for single-server VPS

---

## Success Verification Checklist

Use this checklist to verify the implementation meets all requirements:

- [ ] Users can subscribe to RSS feeds via API
- [ ] RSS Fetcher automatically polls feeds based on tiered strategy
- [ ] New RSS items are auto-delivered to all subscribed users
- [ ] Content is deduplicated by feed_id + content_hash
- [ ] Content updates are detected via HTTP caching headers (Last-Modified, ETag)
- [ ] Updated content is re-processed and synced to Content Service
- [ ] Multiple users can save the same content with individual metadata
- [ ] Users can retrieve their content list with filtering (status, favorite, date)
- [ ] Users can update status, scroll position, and favorite flag
- [ ] Users can delete content from their list (hard delete of user-content)
- [ ] Content is stored as cleaned HTML using readability parser
- [ ] HTML content is sanitized using bluemonday
- [ ] Content size limit of 5MB is enforced
- [ ] Basic search works using PostgreSQL full-text search (title and author fields)
- [ ] Cursor-based pagination works (20 items per page)
- [ ] Feed items table cleanup job removes old completed/failed items
- [ ] All inter-service communication uses REST APIs only
- [ ] Outbox pattern ensures reliable content delivery to Content Service
- [ ] Circuit breaker protects against Content Service failures
- [ ] Services have separate PostgreSQL databases
- [ ] Orphaned content is deleted after 90 days
- [ ] Feeds are auto-disabled after 7 consecutive days of errors
- [ ] 100 feed limit per user is enforced
- [ ] Docker Compose deployment works
- [ ] Database migrations apply successfully with golang-migrate
- [ ] All database triggers function correctly
- [ ] Comprehensive tests pass (unit + integration)
- [ ] API documentation is complete

---

## Technology Stack Summary

### Go Libraries

**Content Service**:
- `github.com/go-shiori/go-readability` - HTML readability extraction
- `github.com/microcosm-cc/bluemonday` - HTML sanitization
- `github.com/lib/pq` - PostgreSQL driver
- `github.com/go-chi/chi` or `github.com/gorilla/mux` - HTTP routing
- `go.uber.org/zap` - Structured logging
- `github.com/prometheus/client_golang` - Metrics
- `github.com/testify/testify` - Testing

**RSS Fetcher Service**:
- `github.com/mmcdole/gofeed` - RSS/Atom parsing
- `github.com/lib/pq` - PostgreSQL driver
- `github.com/go-chi/chi` or `github.com/gorilla/mux` - HTTP routing
- `github.com/robfig/cron/v3` - Job scheduling
- `github.com/sony/gobreaker` - Circuit breaker for Content Service resilience
- `go.uber.org/zap` - Structured logging
- `github.com/prometheus/client_golang` - Metrics

**Shared**:
- `github.com/golang-migrate/migrate/v4` - Database migrations
- `github.com/spf13/viper` - Configuration management
- `github.com/testcontainers/testcontainers-go` - Integration testing

### Infrastructure
- **Database**: PostgreSQL 15+
- **Containerization**: Docker & Docker Compose
- **Language**: Go 1.21+

---

## Development Timeline Estimates

This is a rough estimate for development effort, assuming one developer:

- **Phase 0**: Project Setup - 2-3 days ✅ Completed
- **Phase 1**: Content Service Core - 5-7 days ✅ Completed
- **Phase 2**: RSS Fetcher Feed Management - 3-4 days ✅ Completed
- **Phase 3**: RSS Fetcher Polling & Processing - 5-7 days ✅ Completed
- **Phase 4**: Background Jobs - 2-3 days ✅ Completed
- **Phase 5**: Comprehensive Testing - 5-7 days ✅ Completed
- **Phase 6**: Documentation - 3-4 days 🔲 Not Started
- **Phase 7**: Production Readiness - 3-4 days 🔲 Not Started

**Total Estimate**: 28-39 days of focused development

**Current Progress**: Phases 0-5 completed (19-31 days), Phases 6-7 remaining (6-8 days)

Note: Estimates assume familiarity with Go and the technology stack. Add buffer for learning, debugging, and unexpected issues.

---

## Next Steps

After completing the initial implementation:

1. **Deploy to staging environment**
   - Test on a real VPS
   - Monitor performance and resource usage
   - Validate all functionality end-to-end

2. **Mobile app integration**
   - Provide API documentation to mobile team
   - Support mobile app development with API adjustments
   - Test mobile app integration

3. **Production deployment**
   - Set up production environment
   - Configure monitoring and alerting
   - Implement backup and disaster recovery
   - Add authentication/authorization (JWT)

4. **Iterative improvements**
   - Gather usage metrics
   - Identify performance bottlenecks
   - Implement future enhancements from requirements.md

---

## Risk Assessment & Mitigations

### Technical Risks

1. **Large feed processing performance**
   - Risk: Feeds with thousands of items may overwhelm the system
   - Mitigation: Implement batching, rate limiting, and processing limits

2. **Content extraction failures**
   - Risk: Paywalls and anti-scraping measures may block content
   - Mitigation: Fallback to RSS description, implement retry logic with backoff

3. **Database performance at scale**
   - Risk: Slow queries as data grows
   - Mitigation: Proper indexing, query optimization, pagination

4. **Memory usage with HTML content**
   - Risk: Large HTML content may cause OOM errors
   - Mitigation: 5MB content limit, streaming where possible, connection pooling

### Operational Risks

1. **Feed reliability**
   - Risk: Many feeds may be unreliable or go offline
   - Mitigation: Auto-disable after 7 days, allow manual re-enable

2. **Data integrity**
   - Risk: Orphaned content cleanup may accidentally delete active content
   - Mitigation: Thorough testing of triggers, 90-day grace period

3. **Service dependencies**
   - Risk: Content Service downtime blocks RSS Fetcher
   - Mitigation: Retry logic, circuit breaker, queue pending items

---

## Conclusion

This implementation plan provides a comprehensive roadmap for building the Cairn backend system. The phased approach ensures systematic development with clear milestones and deliverables. Each phase builds upon the previous one, allowing for incremental testing and validation.

The plan prioritizes core functionality first (Phases 1-3), followed by maintenance features (Phase 4), quality assurance (Phase 5), and production readiness (Phase 6). This approach allows for early demonstrations of working software while ensuring production quality by the end.

Key success factors:
- Strict adherence to service separation (REST APIs only)
- Comprehensive testing at each phase
- Security-first approach (HTML sanitization, input validation)
- Performance consideration from the start
- Clear documentation throughout

Following this plan will result in a robust, scalable, and maintainable backend system ready for production deployment.

# Cairn Backend Architecture

This document provides a detailed overview of Cairn's architecture, design decisions, data flows, and technical implementation.

## Table of Contents

- [System Overview](#system-overview)
- [Service Architecture](#service-architecture)
- [Data Models](#data-models)
- [Communication Patterns](#communication-patterns)
- [Content Processing Pipeline](#content-processing-pipeline)
- [Feed Polling Strategy](#feed-polling-strategy)
- [Reliability Patterns](#reliability-patterns)
- [Database Design](#database-design)
- [Security Considerations](#security-considerations)

## System Overview

Cairn is a microservices-based read-it-later backend system designed for scalability, reliability, and maintainability. The system consists of two primary services that communicate exclusively via REST APIs.

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         Client Layer                         │
│                    (Mobile App / Web App)                    │
└────────────────────────────┬────────────────────────────────┘
                             │
                   ┌─────────┴─────────┐
                   │                   │
         ┌─────────▼──────────┐  ┌────▼──────────────────┐
         │  Content Service   │  │  RSS Fetcher Service  │
         │                    │  │                        │
         │  HTTP API (8080)   │◄─┤  HTTP API (8081)      │
         │                    │  │                        │
         │  - Content CRUD    │  │  - Subscription Mgmt   │
         │  - User Lists      │  │  - Feed Polling        │
         │  - Search          │  │  - Content Extraction  │
         │  - Metadata Mgmt   │  │  - Outbox Delivery     │
         └────────┬───────────┘  └────────┬───────────────┘
                  │                       │
         ┌────────▼───────────┐  ┌───────▼────────────────┐
         │  PostgreSQL DB     │  │  PostgreSQL DB         │
         │  (content_service) │  │  (rss_fetcher_service) │
         │                    │  │                         │
         │  - contents        │  │  - feeds                │
         │  - user_contents   │  │  - feed_subscriptions   │
         └────────────────────┘  │  - feed_items           │
                                 │  - content_outbox       │
                                 └─────────────────────────┘
```

### Design Principles

1. **Service Isolation**: Each service owns its data; no direct database access across services
2. **REST-Only Communication**: Services communicate exclusively via HTTP REST APIs
3. **Eventual Consistency**: Acceptable for content delivery; outbox pattern ensures reliability
4. **Idempotency**: Operations designed to be safely retried
5. **Fail-Safe Defaults**: Graceful degradation when dependencies are unavailable
6. **Single Responsibility**: Each service has a clear, focused purpose

## Service Architecture

### Content Service

**Purpose**: Store and serve article content with user-specific metadata

**Core Responsibilities**:
- Store cleaned and sanitized article content
- Manage user-content relationships (one content → many users)
- Provide CRUD operations for content
- Implement full-text search across user's content
- Support bulk operations for RSS Fetcher integration
- Handle content deduplication

**Technology Stack**:
- Go 1.21+
- Chi HTTP router
- PostgreSQL with full-text search (GIN indexes)
- go-readability for content extraction
- bluemonday for HTML sanitization

**Endpoints**:
```
POST   /api/v1/contents                              # Create content
PUT    /api/v1/contents/:id                          # Update content
GET    /api/v1/contents/:id                          # Get content
GET    /api/v1/users/:user_id/contents               # List user's contents
POST   /api/v1/users/:user_id/contents               # Add content to user
PATCH  /api/v1/users/:user_id/contents/:content_id   # Update user-content metadata
DELETE /api/v1/users/:user_id/contents/:content_id   # Remove from user's list
GET    /api/v1/users/:user_id/contents/search        # Search user's contents
POST   /api/v1/contents/bulk                         # Bulk create/update
POST   /api/v1/contents/check-duplicates             # Check for duplicates
```

### RSS Fetcher Service

**Purpose**: Manage RSS feed subscriptions and deliver content to users

**Core Responsibilities**:
- Subscribe/unsubscribe users to RSS feeds
- Poll feeds using intelligent tiered strategy
- Extract full article content from feed items
- Detect content updates via HTTP caching headers
- Deliver content to Content Service via outbox pattern
- Auto-disable failing feeds
- Manage feed polling tiers based on activity

**Technology Stack**:
- Go 1.21+
- Chi HTTP router
- PostgreSQL for feed metadata and outbox
- gofeed for RSS/Atom parsing
- robfig/cron for job scheduling
- sony/gobreaker for circuit breaker

**Endpoints**:
```
POST   /api/v1/users/:user_id/feeds/subscribe   # Subscribe to feed
DELETE /api/v1/users/:user_id/feeds/:feed_id    # Unsubscribe
GET    /api/v1/users/:user_id/feeds             # List subscriptions
PATCH  /api/v1/feeds/:feed_id/enable            # Re-enable disabled feed
```

**Background Workers**:
1. **Feed Polling Worker**: Continuously polls active feeds
2. **Content Extraction Worker**: Extracts full content from feed items
3. **Outbox Delivery Worker**: Delivers content to Content Service
4. **Tier Management Job**: Daily job to adjust feed tiers
5. **Cleanup Jobs**: Remove old outbox entries and feed items

## Data Models

### Content Service Data Models

#### Contents Table

```sql
CREATE TABLE contents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title TEXT NOT NULL,
    author TEXT,
    source_url TEXT NOT NULL,
    canonical_url TEXT,
    cleaned_html TEXT NOT NULL,
    content_hash TEXT NOT NULL,
    published_at TIMESTAMP WITH TIME ZONE,
    source_feed_id UUID,  -- NULL for manually added content
    orphaned_at TIMESTAMP WITH TIME ZONE,  -- Set when last user removes it
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(content_hash, source_feed_id)
);

CREATE INDEX idx_contents_orphaned_at ON contents(orphaned_at) WHERE orphaned_at IS NOT NULL;
CREATE INDEX idx_contents_source_feed_id ON contents(source_feed_id);
```

**Key Fields**:
- `content_hash`: SHA-256 hash of cleaned_html for deduplication
- `canonical_url`: Normalized URL (tracking params removed)
- `orphaned_at`: Timestamp when last user removed content (for cleanup)
- `source_feed_id`: NULL for manually saved content, UUID for RSS content

#### User Contents Table

```sql
CREATE TABLE user_contents (
    user_id UUID NOT NULL,
    content_id UUID NOT NULL REFERENCES contents(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'unread' CHECK (status IN ('unread', 'reading', 'completed', 'archived')),
    scroll_position INTEGER DEFAULT 0,
    is_favorite BOOLEAN DEFAULT FALSE,
    added_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_accessed_at TIMESTAMP WITH TIME ZONE,

    PRIMARY KEY (user_id, content_id)
);

CREATE INDEX idx_user_contents_user_status ON user_contents(user_id, status);
CREATE INDEX idx_user_contents_user_favorite ON user_contents(user_id, is_favorite) WHERE is_favorite = TRUE;
CREATE INDEX idx_user_contents_added_at ON user_contents(user_id, added_at DESC);
```

**Triggers**:
- **After INSERT**: Clear `orphaned_at` on contents when user adds it
- **After DELETE**: Set `orphaned_at` on contents when last user removes it

### RSS Fetcher Service Data Models

#### Feeds Table

```sql
CREATE TABLE feeds (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    feed_url TEXT NOT NULL UNIQUE,
    title TEXT,
    description TEXT,
    site_url TEXT,
    polling_tier TEXT NOT NULL DEFAULT 'active' CHECK (polling_tier IN ('active', 'moderate', 'quiet')),
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled', 'error')),
    last_fetched_at TIMESTAMP WITH TIME ZONE,
    last_published_at TIMESTAMP WITH TIME ZONE,
    next_poll_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    consecutive_error_days INTEGER DEFAULT 0,
    last_error_at TIMESTAMP WITH TIME ZONE,
    last_error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_feeds_next_poll ON feeds(next_poll_at) WHERE status = 'active';
CREATE INDEX idx_feeds_polling_tier ON feeds(polling_tier);
```

**Polling Tiers**:
- `active`: Last published within 7 days → poll every 1 hour
- `moderate`: Last published within 30 days → poll every 6 hours
- `quiet`: Last published over 30 days ago → poll every 24 hours

#### Feed Subscriptions Table

```sql
CREATE TABLE feed_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    feed_id UUID NOT NULL REFERENCES feeds(id) ON DELETE CASCADE,
    subscribed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(user_id, feed_id)
);

CREATE INDEX idx_feed_subscriptions_user ON feed_subscriptions(user_id);
CREATE INDEX idx_feed_subscriptions_feed ON feed_subscriptions(feed_id);
```

**Trigger**: Enforce 100 feed limit per user

#### Feed Items Table

```sql
CREATE TABLE feed_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    feed_id UUID NOT NULL REFERENCES feeds(id) ON DELETE CASCADE,
    item_guid TEXT NOT NULL,
    title TEXT NOT NULL,
    author TEXT,
    source_url TEXT NOT NULL,
    description TEXT,
    published_at TIMESTAMP WITH TIME ZONE,
    content_hash TEXT,
    http_last_modified TEXT,  -- HTTP Last-Modified header
    http_etag TEXT,            -- HTTP ETag header
    last_checked_at TIMESTAMP WITH TIME ZONE,
    processing_status TEXT NOT NULL DEFAULT 'pending' CHECK (processing_status IN ('pending', 'processing', 'completed', 'failed')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(feed_id, item_guid)
);

CREATE INDEX idx_feed_items_processing_status ON feed_items(processing_status) WHERE processing_status = 'pending';
CREATE INDEX idx_feed_items_feed_id ON feed_items(feed_id);
```

#### Content Outbox Table

```sql
CREATE TABLE content_outbox (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    feed_item_id UUID NOT NULL REFERENCES feed_items(id),
    content_payload JSONB NOT NULL,  -- Full content ready for Content Service API
    user_ids UUID[] NOT NULL,        -- Array of user IDs to deliver to
    delivery_status TEXT NOT NULL DEFAULT 'pending' CHECK (delivery_status IN ('pending', 'sending', 'delivered', 'failed')),
    content_service_id UUID,         -- ID returned from Content Service
    retry_count INTEGER DEFAULT 0,
    next_retry_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    delivered_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_content_outbox_delivery_status ON content_outbox(delivery_status, next_retry_at) WHERE delivery_status IN ('pending', 'failed');
```

## Communication Patterns

### 1. RSS Fetcher → Content Service

**Pattern**: REST API calls with outbox pattern for reliability

**Flow**:
1. RSS Fetcher extracts content from feed item
2. RSS Fetcher writes to local `content_outbox` table
3. Outbox worker picks up pending entries
4. Worker calls Content Service REST API
5. On success: Mark as delivered, store content ID
6. On failure: Increment retry counter, schedule next retry

**Benefits**:
- Decouples content extraction from delivery
- Survives Content Service downtime
- Enables retry with exponential backoff
- Provides audit trail

**Retry Schedule**:
```
Retry 1: 1 minute
Retry 2: 5 minutes
Retry 3: 15 minutes
Retry 4: 1 hour
Retry 5: 4 hours
Retry 6: 12 hours
After 6 retries: Mark as failed
```

### 2. Circuit Breaker Pattern

**Purpose**: Prevent overwhelming a failing Content Service

**Implementation**: Using `sony/gobreaker`

**States**:
- **Closed**: Normal operation, requests flow through
- **Open**: After 5 consecutive failures, reject requests immediately
- **Half-Open**: After 30 seconds, allow test request

**Benefits**:
- Prevents cascade failures
- Allows failing service to recover
- Provides fast-fail behavior

## Content Processing Pipeline

### Content Creation Flow

```
┌─────────────────────────────────────────────────────────────┐
│                    Content Processing                        │
└─────────────────────────────────────────────────────────────┘

1. Receive HTML or URL
   │
   ├─ If URL provided → Fetch HTML (30s timeout)
   └─ If HTML provided → Use directly
   │
2. Apply Readability Parsing (go-readability)
   │ ├─ Extract title, author, published_at
   │ ├─ Extract main content
   │ └─ Remove ads, navigation, footers
   │
3. Apply HTML Sanitization (bluemonday)
   │ ├─ Remove <script> tags
   │ ├─ Remove <iframe> tags
   │ ├─ Allow safe HTML tags only
   │ └─ Remove dangerous attributes (onclick, etc.)
   │
4. Generate Content Hash (SHA-256)
   │ └─ Hash of cleaned_html for deduplication
   │
5. Canonicalize URL
   │ ├─ Normalize scheme (https)
   │ ├─ Lowercase host
   │ └─ Remove tracking parameters (utm_*, fbclid, etc.)
   │
6. Validate Size (max 5MB)
   │
7. Check for Duplicates (content_hash + source_feed_id)
   │ ├─ If exists → Return existing content
   │ └─ If new → Store in database
   │
8. Store in Database
   └─ Return content ID
```

### Content Update Detection

```
┌─────────────────────────────────────────────────────────────┐
│                   Update Detection Flow                      │
└─────────────────────────────────────────────────────────────┘

1. Feed item already exists in feed_items table
   │
2. Make conditional HTTP request
   │ ├─ Include "If-Modified-Since: {http_last_modified}"
   │ └─ Include "If-None-Match: {http_etag}"
   │
3. Check HTTP response
   │
   ├─ 304 Not Modified
   │  └─ Update last_checked_at, skip reprocessing
   │
   └─ 200 OK (content changed)
      │
      ├─ Re-fetch and re-process content
      ├─ Generate new content_hash
      ├─ Compare with stored hash
      │
      └─ If hash changed
         │
         ├─ Call Content Service PUT /api/v1/contents/:id
         ├─ Update cleaned_html, metadata, content_hash
         └─ Content Service preserves all user-content relationships
```

## Feed Polling Strategy

### Tiered Polling

**Objective**: Reduce load on inactive feeds while keeping active feeds fresh

**Tier Assignment Logic**:

```
IF last_published_at > NOW() - INTERVAL '7 days'
    THEN tier = 'active' (poll every 1 hour)
ELSE IF last_published_at > NOW() - INTERVAL '30 days'
    THEN tier = 'moderate' (poll every 6 hours)
ELSE
    THEN tier = 'quiet' (poll every 24 hours)
```

**Tier Management**:
- Daily background job evaluates all feeds
- Updates `polling_tier` based on `last_published_at`
- Updates `next_poll_at` based on new tier
- New content promotes feed to 'active' tier immediately

### Polling Worker Flow

```
┌─────────────────────────────────────────────────────────────┐
│                    Feed Polling Worker                       │
└─────────────────────────────────────────────────────────────┘

LOOP (continuous):
    1. Query feeds WHERE next_poll_at <= NOW() AND status = 'active'
       ORDER BY next_poll_at ASC
       LIMIT 10

    2. FOR EACH feed (concurrent, max 5 workers):
       │
       ├─ Fetch RSS/Atom XML (30s timeout)
       │
       ├─ Parse feed metadata and items
       │
       ├─ FOR EACH item:
       │  │
       │  ├─ Check if exists (feed_id + item_guid)
       │  │
       │  ├─ If new:
       │  │  └─ Insert into feed_items with status='pending'
       │  │
       │  └─ If exists:
       │     └─ Check for updates (see Update Detection)
       │
       ├─ Update feed metadata:
       │  ├─ last_fetched_at = NOW()
       │  ├─ last_published_at = MAX(item.published_at)
       │  ├─ consecutive_error_days = 0
       │  └─ next_poll_at = NOW() + tier_interval
       │
       └─ IF error:
          ├─ Increment consecutive_error_days
          ├─ Store last_error_at, last_error_message
          └─ IF consecutive_error_days >= 7:
             └─ Set status = 'disabled'

    3. Sleep 10 seconds
```

### Content Extraction Worker Flow

```
┌─────────────────────────────────────────────────────────────┐
│                 Content Extraction Worker                    │
└─────────────────────────────────────────────────────────────┘

LOOP (continuous):
    1. Query feed_items WHERE processing_status = 'pending'
       LIMIT 20

    2. FOR EACH feed_item (concurrent, max 5 workers):
       │
       ├─ Update status = 'processing'
       │
       ├─ Fetch full article HTML (30s timeout)
       │  └─ On timeout/error: Fallback to RSS description
       │
       ├─ Apply readability + sanitization
       │
       ├─ Generate content_hash
       │
       ├─ Get all subscribed users for this feed
       │
       ├─ Write to content_outbox:
       │  ├─ content_payload = {full content JSON}
       │  ├─ user_ids = [array of subscribed user IDs]
       │  └─ delivery_status = 'pending'
       │
       └─ Update feed_item status = 'completed'

    3. Sleep 10 seconds
```

## Reliability Patterns

### 1. Outbox Pattern

**Purpose**: Ensure content delivery even if Content Service is temporarily unavailable

**Implementation**:
- RSS Fetcher writes to local `content_outbox` table (ACID transaction)
- Separate worker processes outbox entries
- Worker retries failed deliveries with exponential backoff
- Provides at-least-once delivery guarantee

**Benefits**:
- Decouples content extraction from delivery
- No content loss during Content Service downtime
- Audit trail of all deliveries

### 2. Circuit Breaker

**Purpose**: Prevent overwhelming a failing Content Service

**Implementation**: Using `sony/gobreaker`

**Configuration**:
- Open after 5 consecutive failures
- Half-open after 30 seconds
- Test with single request before closing

**Benefits**:
- Fast-fail when service is down
- Prevents resource exhaustion
- Automatic recovery detection

### 3. Idempotency

**Content Service**:
- Duplicate check on `content_hash + source_feed_id`
- Returns existing content if duplicate
- Safe to retry creation

**RSS Fetcher**:
- Feed items deduplicated by `feed_id + item_guid`
- Outbox entries retried safely

### 4. Graceful Degradation

**Strategies**:
- Fetch timeout → Use RSS description instead of full article
- Content Service unavailable → Queue in outbox for later delivery
- Feed fetch failure → Increment error counter, retry next poll

## Database Design

### Indexes

**Content Service**:
```sql
-- Fast lookups by hash and feed for deduplication
CREATE UNIQUE INDEX idx_contents_hash_feed ON contents(content_hash, source_feed_id);

-- Fast orphaned content cleanup queries
CREATE INDEX idx_contents_orphaned_at ON contents(orphaned_at) WHERE orphaned_at IS NOT NULL;

-- Fast user content list queries
CREATE INDEX idx_user_contents_added_at ON user_contents(user_id, added_at DESC);

-- Fast filtering by status and favorites
CREATE INDEX idx_user_contents_user_status ON user_contents(user_id, status);
CREATE INDEX idx_user_contents_user_favorite ON user_contents(user_id, is_favorite) WHERE is_favorite = TRUE;

-- Full-text search
CREATE INDEX idx_contents_search ON contents USING GIN (to_tsvector('english', title || ' ' || COALESCE(author, '')));
```

**RSS Fetcher Service**:
```sql
-- Fast feed polling queries
CREATE INDEX idx_feeds_next_poll ON feeds(next_poll_at) WHERE status = 'active';

-- Fast pending feed items queries
CREATE INDEX idx_feed_items_processing_status ON feed_items(processing_status) WHERE processing_status = 'pending';

-- Fast outbox delivery queries
CREATE INDEX idx_content_outbox_delivery_status ON content_outbox(delivery_status, next_retry_at) WHERE delivery_status IN ('pending', 'failed');
```

### Triggers

**Content Service**:
```sql
-- Clear orphaned_at when user adds content
CREATE TRIGGER clear_orphaned_at
    AFTER INSERT ON user_contents
    FOR EACH ROW
    EXECUTE FUNCTION clear_content_orphaned_at();

-- Set orphaned_at when last user removes content
CREATE TRIGGER mark_orphaned_content
    AFTER DELETE ON user_contents
    FOR EACH ROW
    EXECUTE FUNCTION mark_content_as_orphaned();
```

**RSS Fetcher Service**:
```sql
-- Enforce 100 feed limit per user
CREATE TRIGGER enforce_feed_limit
    BEFORE INSERT ON feed_subscriptions
    FOR EACH ROW
    EXECUTE FUNCTION check_user_feed_limit();
```

## Security Considerations

### HTML Sanitization

**Library**: bluemonday (industry-standard Go HTML sanitizer)

**Allowed Tags**:
- Text: `<p>`, `<h1>`-`<h6>`, `<br>`, `<hr>`
- Formatting: `<strong>`, `<em>`, `<u>`, `<s>`
- Lists: `<ul>`, `<ol>`, `<li>`
- Links: `<a>` (with `href` attribute only)
- Media: `<img>` (with `src`, `alt` attributes only)
- Code: `<code>`, `<pre>`
- Tables: `<table>`, `<tr>`, `<td>`, `<th>`

**Blocked/Removed**:
- `<script>` tags and JavaScript
- `<iframe>` and embedded content
- Event handlers (`onclick`, `onerror`, etc.)
- `<form>` elements
- `<style>` tags (inline styles allowed on safe attributes)

### Input Validation

**Content Service**:
- Content size limit: 5MB
- URL format validation
- UUID validation for IDs
- Status enum validation
- Scroll position >= 0

**RSS Fetcher Service**:
- Feed URL format validation
- 100 feed limit per user (enforced by trigger)
- UUID validation for IDs

### Database Security

- No raw SQL queries (use parameterized queries)
- Prepared statements prevent SQL injection
- Foreign key constraints enforce referential integrity
- Check constraints enforce valid enum values

## Performance Considerations

### Connection Pooling

Both services use connection pooling:
```go
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(5 * time.Minute)
```

### Pagination

- Cursor-based pagination for user content lists (20 items/page)
- Offset-based pagination for feed subscriptions (50 items/page)

### Caching

Future enhancement:
- Cache feed metadata (1 hour TTL)
- Cache user content counts
- HTTP caching headers for API responses

### Batch Processing

- Bulk content creation (max 100 items)
- Batch feed polling (10 feeds at a time)
- Batch outbox delivery (concurrent workers)

## Monitoring & Observability

### Health Checks

Both services expose:
- `GET /health/live` - Liveness check (is process running?)
- `GET /health/ready` - Readiness check (can handle traffic? DB connected?)

### Logging

Structured logging includes:
- Request ID for tracing
- HTTP method, path, status code
- Response time
- User ID (when available)
- Error stack traces

### Metrics (Future)

Planned Prometheus metrics:
- HTTP request duration (histogram)
- HTTP request count by status code
- Database query duration
- Feed polling success/failure rates
- Outbox queue depth
- Circuit breaker state changes

## Scalability Considerations

### Horizontal Scaling

**Content Service**:
- Stateless HTTP server → scale by adding instances
- Database connection pooling limits concurrent connections
- Load balancer distributes requests

**RSS Fetcher Service**:
- API server: Stateless → scale by adding instances
- Workers: Use distributed locks for job coordination (future)

### Database Scaling

**Current**: Single PostgreSQL instance per service

**Future**:
- Read replicas for read-heavy workloads
- Connection pooler (PgBouncer) for connection management
- Partitioning for large tables (contents, feed_items)

### Worker Scaling

**Current**: Single worker instance per service

**Future**:
- Multiple worker instances with job queue (Redis/RabbitMQ)
- Distributed job scheduling
- Worker-specific scaling based on queue depth

## Future Enhancements

### Authentication & Authorization

- JWT-based authentication
- User service for user management
- API key authentication for service-to-service

### Caching Layer

- Redis for feed metadata caching
- API response caching with ETags
- User content count caching

### Message Queue

- Replace outbox polling with message queue (Redis Streams, RabbitMQ)
- Event-driven architecture for real-time updates

### Observability

- Distributed tracing (OpenTelemetry)
- Prometheus metrics
- Grafana dashboards
- Alert rules for critical failures

### Content Features

- Thumbnail extraction and storage
- PDF support
- Video/podcast content
- Multi-language support with language detection

## Conclusion

Cairn's architecture prioritizes:
- **Reliability**: Outbox pattern, circuit breaker, retry logic
- **Scalability**: Stateless services, connection pooling, batch processing
- **Maintainability**: Clean separation of concerns, comprehensive testing
- **Performance**: Indexes, pagination, caching strategy
- **Security**: HTML sanitization, input validation, parameterized queries

The microservices architecture allows independent scaling and deployment of Content Service and RSS Fetcher Service while maintaining loose coupling through REST APIs.

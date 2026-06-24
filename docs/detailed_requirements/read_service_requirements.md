# Cairn Content Storage - Requirements

## Project Overview

Cairn is a simple read-it-later mobile app that allows users to read and discover long-form content. This repository contains the backend services responsible for fetching, storing, and serving user-selected content.

The backend consists of modular services built in Go:
- **Content Service**: Core service responsible for storing and serving content
- **RSS Fetcher Service**: Lightweight service for fetching and parsing RSS feeds
- Future fetcher services can be added (web scraper, API integrations, etc.)

## Core Principles

1. **Modular Architecture**: Services are independent and communicate via REST APIs
2. **Shared Content Model**: The same content can be saved by multiple users; the system must handle deduplication and user-content relationships efficiently
3. **Scalability**: Design for growth - additional fetcher services can be added over time

## Content Service Requirements

### Content Storage

The Content Service must store and manage article content with the following specifications:

#### Content Format
- **Cleaned HTML**: Pass raw HTML through a readability module (e.g., [go-shiori/go-readability](https://github.com/go-shiori/go-readability)) to extract clean, readable content
  - If readability parsing fails, fall back to storing the raw HTML content
- **Original URL**: Store the source URL for reference
- **Content Hash**: Generate SHA-256 hash of the cleaned HTML content for deduplication purposes
  - **Design Note**: The hash is generated from the processed content (cleaned HTML or raw HTML fallback). This means the same article could theoretically produce different hashes if readability parsing succeeds in one instance but fails in another. This is an accepted trade-off for simplicity.

#### Metadata Storage
Store the following metadata for each piece of content:
- Title
- Author
- Publication date
- Description/excerpt
- Image references (URL only - no image downloading/hosting initially)
- Source type (RSS, web, etc.)
- Content-type specific metadata (e.g., feed_id for RSS content)

### User-Content Relationships

Track user-specific data for each saved content item:

#### Required Fields
- **Status**: One of `unread`, `read`, or `archived`
- **Scroll Position**: Fraction in `[0, 1]` representing how far the reader has scrolled through the content
  - Stores `offsetY / contentHeight` (e.g., `0.5` means halfway through the article)
  - The mobile and web apps compute this from the current scroll offset relative to total content height
  - A relative fraction is stable across different viewports, layouts, font sizes, and devices (unlike an absolute pixel/character offset, which shifts when content reflows)
- **Favorite**: Boolean flag for favorited items
- **Added Timestamp**: When the user saved/received this content

#### Future Enhancements
The following features are planned for future iterations:
- User tags
- Highlights and annotations
- Personal notes
- Reading time estimates
- Custom folders/collections

### Storage Limits

- **Content per User**: No limits enforced initially. System will monitor usage patterns to determine if limits are needed in the future.
- **Content Size**: Maximum 5MB per article. Articles exceeding this limit should be rejected with an appropriate error.

### Content Security

All stored HTML content must be sanitized to prevent security vulnerabilities:

#### HTML Sanitization
- **Sanitization Library**: Use a well-established HTML sanitization library (e.g., [bluemonday](https://github.com/microcosm-cc/bluemonday) for Go)
- **When to Sanitize**: Sanitize HTML content before storing in the database
- **Allowed Elements**: Permit safe content elements only:
  - Text formatting: `<p>`, `<br>`, `<strong>`, `<em>`, `<u>`, `<h1>`-`<h6>`, `<blockquote>`
  - Lists: `<ul>`, `<ol>`, `<li>`
  - Links: `<a>` (with href validation - http/https only)
  - Media: `<img>` (src validation - http/https only), `<figure>`, `<figcaption>`
  - Code: `<pre>`, `<code>`
  - Tables: `<table>`, `<thead>`, `<tbody>`, `<tr>`, `<td>`, `<th>`
- **Stripped Elements**: Remove all potentially dangerous elements:
  - Scripts: `<script>`, `<noscript>`
  - Event handlers: `onclick`, `onerror`, `onload`, etc.
  - Embedded content: `<iframe>`, `<embed>`, `<object>`
  - Forms: `<form>`, `<input>`, `<button>`
  - Style: `<style>` tags and inline `style` attributes (preserve safe CSS via allowlist if needed)
- **Attribute Filtering**: Remove dangerous attributes like `javascript:` URLs, `data:` URLs, etc.

#### XSS Prevention
- **Defense in Depth**: Even though content is sanitized at storage, the mobile app should implement Content Security Policy (CSP) when rendering HTML in WebViews
- **URL Validation**: Validate all URLs (source URLs, image URLs, link hrefs) are well-formed http/https before storage
- **Output Encoding**: Mobile app must properly encode HTML content when rendering

**Note**: Sanitization applies to both:
- Cleaned HTML from the readability parser
- Raw HTML fallback content

### Content Deduplication

Deduplication is handled on a **per-fetcher basis** to accommodate different fetcher strategies:

#### RSS Fetcher Deduplication
- Deduplicate based on: `feed_id` + `content_hash`
- If the same article appears in the same feed, store only once
- If different users subscribe to the same feed, link to the same content record
- **Cross-feed behavior**: If the same article appears in different feeds (different `feed_id`), create separate content records. Users subscribed to multiple feeds may see the same article multiple times in their reading list.
  - **Design Note**: This cross-feed duplication is intentional and accepted as an edge case to maintain feed independence and simplify the deduplication logic.

#### Future Fetchers
- Each fetcher type can implement its own deduplication strategy
- Examples: URL canonicalization for web scraper, API-specific IDs, etc.

### Content Retrieval API

Provide REST APIs for the mobile app to:
- Retrieve user's content list with filtering (by status, favorite, date range)
- Get full content for a specific article
- Update user-content metadata (status, scroll position, favorite)
- Delete content from user's list (hard delete of user-content relationship - content record is kept if other users have it saved)
- Search user's saved content (basic text matching within title and author fields)

**Response Format**: JSON with HTML content embedded in a designated field

**Pagination**: Cursor-based pagination with 20 items per page

### Content Update API

Provide REST APIs for the RSS Fetcher to update content when source articles change:
- Update content body and metadata while preserving all user-content relationships
- Update endpoint: `PUT /api/v1/contents/:id`
  - Accepts updated cleaned_html, title, author, description, image_urls, content_hash
  - Preserves all existing user-content records (status, scroll_position, is_favorite)
  - Updates `updated_at` timestamp on the content record
- The content hash is updated to reflect the new content
- Deduplication is re-evaluated with the new hash

### Data Model Considerations

The system should maintain:
1. **Content Table**: Stores unique content items (shared across users)
2. **User-Content Junction Table**: Maps users to content with user-specific metadata
3. **Feed Subscriptions Table**: Tracks which users subscribe to which RSS feeds

**Note**: The Content Service does not maintain a Users table. User IDs are provided in API requests and validated by a separate authentication service.

**Initial Development**: For the initial implementation before production deployment, user ID validation is not enforced. Any user_id can be sent in API requests. This will be replaced with proper JWT-based authentication before production deployment.

### Content Retention Policy

- **User-Content Deletion**: When a user deletes content from their list, the user-content junction record is immediately hard deleted
  - The content record itself is preserved if other users have it saved
  - No restoration of deleted user-content relationships is supported
- **Orphaned Content**: Content that is no longer saved by any user will be automatically deleted after 90 days
  - Content becomes "orphaned" when the last user-content relationship pointing to it is deleted
  - The 90-day grace period allows for potential re-discovery via RSS feeds
- **Cleanup Process**: A periodic background job will identify and remove orphaned content past the 90-day retention period

## RSS Fetcher Service Requirements

### Feed Management

- **User Subscriptions**: Individual users can subscribe to RSS feeds via the mobile app API
- **Feed Limit**: Maximum of 100 feeds per user (hard limit - API returns error on attempt to exceed)
- **Feed Registration**: When a user subscribes to a feed, the system should:
  - Check if the user has reached the 100 feed limit; return error if exceeded
  - Check if the feed already exists in the database
  - If new, validate the feed URL and add it to the monitoring list
  - Subscribe the user to that feed

### Feed Processing

- **Tiered Polling Strategy**: Optimize polling frequency based on feed activity to balance freshness with resource efficiency
  - **Tier 1 - Active Feeds**: Poll every 1 hour
    - Feeds that have published new content in the last 7 days
    - Newly added feeds (first 7 days after subscription)
  - **Tier 2 - Moderate Feeds**: Poll every 6 hours
    - Feeds that have published content in the last 30 days but not in the last 7 days
  - **Tier 3 - Quiet Feeds**: Poll every 24 hours
    - Feeds that haven't published content in over 30 days
  - **Tier Promotion**: When a quiet/moderate feed publishes new content, immediately promote it to the appropriate tier
  - **Tier Demotion**: Automatically demote feeds to lower tiers based on the time since last publication
  - **Implementation**: A background scheduler evaluates feed tiers daily and adjusts polling schedules accordingly
- **Content Extraction**: For each new item in a feed:
  1. Fetch the full article content from the source URL
     - If fetching fails (paywall, error, blocked, timeout), fall back to storing just the RSS feed description/summary
  2. Pass through readability parser to extract clean HTML
     - If parsing fails, store the raw HTML
  3. Extract metadata (title, author, date, description, images)
  4. Generate content hash for deduplication
- **Error Handling**: When a feed returns errors (404, 5xx, timeouts, invalid XML, SSL errors, etc.):
  - Continue retrying the feed with normal polling schedule
  - After 7 consecutive days of any type of error, automatically disable the feed
  - Disabled feeds remain in the user's feed list with a "disabled" state
  - Users are not notified when feeds are auto-disabled
  - Users can manually re-enable or unsubscribe from disabled feeds via the API

### Auto-Delivery

When new content is discovered in an RSS feed:
- Automatically add it to **all subscribed users'** reading lists
- Set initial status as `unread`
- Deduplicate: If content already exists (same feed_id + content_hash), link to existing content record

### Content Update Detection

The RSS Fetcher can detect when an article has been updated at its source:

- **Update Tracking**: Store `last_modified` header and `etag` from HTTP responses when fetching article content
- **Change Detection**: On subsequent fetches (during normal feed polling), compare stored values with new HTTP headers
- **Update Processing**: When an update is detected:
  1. Re-fetch and re-process the article content
  2. Generate new content hash
  3. If hash differs from stored content, update the content record via Content Service API
  4. User reading positions are preserved (scroll_position is not reset)
  5. User statuses are preserved (read/unread status is not changed)
- **Update API**: Content Service provides an update endpoint that preserves user-content relationships
- **Limitations**:
  - Not all sources provide `last_modified` or `etag` headers
  - Updates are only detected during normal feed polling cycles
  - Major rewrites may result in new content records rather than updates (if URL changes)

### Communication with Content Service

- Use **REST API exclusively** to communicate with the Content Service
- The RSS Fetcher **must not** directly access the Content Service database
- The RSS Fetcher can maintain an **internal queue** for processing feeds and items
- API endpoints needed from Content Service:
  - Add/update content
  - Check for existing content (deduplication)
  - Add content to user's reading list
  - Bulk operations for efficiency (batch size: 100 items per request)
    - **Handling Large Feeds**: If a feed contains more than 100 new items, the RSS Fetcher should make multiple sequential requests, splitting items into batches of 100
    - Example: A feed with 350 new items requires 4 requests (100 + 100 + 100 + 50)

### Outbox Pattern for Resilience

To ensure reliable content delivery when the Content Service is temporarily unavailable, the RSS Fetcher implements the **Outbox Pattern**:

- **Content Outbox Table**: Processed content is first written to a local `content_outbox` table before being sent to the Content Service
- **Delivery States**: Outbox entries have states: `pending`, `sending`, `delivered`, `failed`
- **Background Delivery Worker**: A background job continuously processes pending outbox entries and sends them to the Content Service
- **Retry Logic**: Failed deliveries are retried with exponential backoff (1 min, 5 min, 15 min, 1 hour, max 6 retries)
- **Idempotency**: The Content Service deduplication ensures that retried deliveries don't create duplicate content
- **Benefits**:
  - Prevents content loss during Content Service outages
  - Decouples content processing from content delivery
  - Provides visibility into delivery status and failures

## Technical Stack

### Database
- **PostgreSQL**: Primary data store for all services
- **Architectural Requirement**: Each service MUST have its own separate database/schema
  - Services communicate exclusively via REST APIs
  - No direct database access between services
  - This ensures true service independence and scalability

### Communication
- **REST APIs**: All inter-service communication uses REST
- Services may maintain internal queues for async processing (in-memory or persistent)
- Standard HTTP status codes and JSON responses

### Response Format
- **JSON**: All API responses in JSON format
- **HTML Content**: Cleaned HTML content embedded in JSON response body
- Example:
  ```json
  {
    "id": "uuid",
    "title": "Article Title",
    "author": "Author Name",
    "content": "<article><p>Cleaned HTML content...</p></article>",
    "url": "https://example.com/article",
    "metadata": {
      "publishedAt": "2025-01-15T10:00:00Z",
      "images": ["https://example.com/image.jpg"]
    }
  }
  ```

### Deployment

- **Initial Environment**: Docker Compose for local development and initial deployment
  - PostgreSQL included as a service in docker-compose.yml
  - All backend services (Content Service, RSS Fetcher) as separate containers
- **Target Architecture**: Single-server deployment with Docker containers
- **Future Scaling**: Plan to migrate to Kubernetes when scaling requirements emerge
- **Infrastructure**: Suitable for deployment on cloud VPS providers (DigitalOcean, AWS EC2, etc.)

## Database Schema

Since the Content Service and RSS Fetcher Service must have separate databases, the following schemas define the tables for each service.

### Content Service Database

#### 1. `contents` table

Stores unique content items shared across all users.

```sql
CREATE TABLE contents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    content_hash VARCHAR(64) NOT NULL, -- SHA-256 hash
    cleaned_html TEXT NOT NULL, -- Max 5MB enforced at application level
    original_url TEXT NOT NULL,
    canonical_url TEXT, -- Normalized URL for deduplication (future use)
    title TEXT NOT NULL,
    author TEXT,
    published_at TIMESTAMP WITH TIME ZONE,
    description TEXT,
    image_urls TEXT[], -- Array of image URLs
    source_type VARCHAR(50) NOT NULL, -- 'rss', 'web', etc.

    -- Source-specific identifiers (nullable, depends on source_type)
    source_feed_id UUID, -- Feed ID for RSS content (more efficient than JSONB extraction)

    -- Content-type specific metadata stored as JSONB for flexibility
    -- Use for additional metadata that doesn't need indexing
    metadata JSONB,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    orphaned_at TIMESTAMP WITH TIME ZONE, -- When last user-content relationship was deleted

    -- Constraints
    CONSTRAINT chk_html_size CHECK (octet_length(cleaned_html) <= 5242880) -- 5MB
);

-- Composite index for RSS deduplication (using dedicated column for efficiency)
CREATE INDEX idx_contents_rss_dedup ON contents(content_hash, source_feed_id)
    WHERE source_type = 'rss';
CREATE INDEX idx_contents_orphaned ON contents(orphaned_at)
    WHERE orphaned_at IS NOT NULL;
CREATE INDEX idx_contents_url ON contents(original_url);
CREATE INDEX idx_contents_canonical_url ON contents(canonical_url)
    WHERE canonical_url IS NOT NULL;

-- Full-text search index for title and author (supports basic search feature)
CREATE INDEX idx_contents_search ON contents
    USING gin(to_tsvector('english', coalesce(title, '') || ' ' || coalesce(author, '')));
```

#### 2. `user_contents` table

Junction table mapping users to content with user-specific metadata.

```sql
CREATE TABLE user_contents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL, -- External user ID, not validated initially
    content_id UUID NOT NULL REFERENCES contents(id) ON DELETE CASCADE,

    -- User-specific metadata
    status VARCHAR(20) NOT NULL DEFAULT 'unread', -- 'unread', 'read', 'archived'
    scroll_position NUMERIC(5,4) NOT NULL DEFAULT 0.0, -- Reading progress as a fraction in [0,1]
    is_favorite BOOLEAN NOT NULL DEFAULT false,

    -- Timestamps
    added_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- Constraints
    CONSTRAINT chk_status CHECK (status IN ('unread', 'read', 'archived')),
    CONSTRAINT chk_scroll_position CHECK (scroll_position >= 0 AND scroll_position <= 1),
    CONSTRAINT unique_user_content UNIQUE(user_id, content_id)
);

-- Indexes for efficient querying
CREATE INDEX idx_user_contents_user ON user_contents(user_id, added_at DESC);
CREATE INDEX idx_user_contents_status ON user_contents(user_id, status);
CREATE INDEX idx_user_contents_favorite ON user_contents(user_id, is_favorite)
    WHERE is_favorite = true;
CREATE INDEX idx_user_contents_content ON user_contents(content_id);
```

#### 3. Triggers for orphaned content tracking

Automatically manage orphaned content lifecycle.

```sql
-- Trigger to mark content as orphaned when last user removes it
CREATE OR REPLACE FUNCTION mark_content_orphaned()
RETURNS TRIGGER AS $$
BEGIN
    -- Check if this was the last user-content relationship
    IF NOT EXISTS (
        SELECT 1 FROM user_contents WHERE content_id = OLD.content_id
    ) THEN
        UPDATE contents
        SET orphaned_at = NOW()
        WHERE id = OLD.content_id;
    END IF;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_mark_orphaned
AFTER DELETE ON user_contents
FOR EACH ROW
EXECUTE FUNCTION mark_content_orphaned();

-- Trigger to clear orphaned status when content is re-saved
CREATE OR REPLACE FUNCTION clear_orphaned_status()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE contents
    SET orphaned_at = NULL
    WHERE id = NEW.content_id AND orphaned_at IS NOT NULL;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_clear_orphaned
AFTER INSERT ON user_contents
FOR EACH ROW
EXECUTE FUNCTION clear_orphaned_status();
```

### RSS Fetcher Service Database

#### 1. `feeds` table

Stores RSS feed information and polling management.

```sql
CREATE TABLE feeds (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    feed_url TEXT NOT NULL UNIQUE,

    -- Feed metadata
    title TEXT,
    description TEXT,
    site_url TEXT,

    -- Polling management
    polling_tier VARCHAR(20) NOT NULL DEFAULT 'active', -- 'active', 'moderate', 'quiet'
    last_fetched_at TIMESTAMP WITH TIME ZONE,
    last_published_at TIMESTAMP WITH TIME ZONE,
    next_poll_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- Status and error tracking
    status VARCHAR(20) NOT NULL DEFAULT 'active', -- 'active', 'disabled'
    consecutive_error_days INTEGER NOT NULL DEFAULT 0,
    last_error_at TIMESTAMP WITH TIME ZONE,
    last_error_message TEXT,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- Constraints
    CONSTRAINT chk_polling_tier CHECK (polling_tier IN ('active', 'moderate', 'quiet')),
    CONSTRAINT chk_status CHECK (status IN ('active', 'disabled'))
);

-- Indexes
CREATE INDEX idx_feeds_next_poll ON feeds(next_poll_at) WHERE status = 'active';
CREATE INDEX idx_feeds_polling_tier ON feeds(polling_tier);
CREATE INDEX idx_feeds_last_published ON feeds(last_published_at);
```

#### 2. `feed_subscriptions` table

Tracks which users subscribe to which feeds.

```sql
CREATE TABLE feed_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL, -- External user ID
    feed_id UUID NOT NULL REFERENCES feeds(id) ON DELETE CASCADE,

    subscribed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    CONSTRAINT unique_user_feed UNIQUE(user_id, feed_id)
);

-- Indexes
CREATE INDEX idx_feed_subs_user ON feed_subscriptions(user_id);
CREATE INDEX idx_feed_subs_feed ON feed_subscriptions(feed_id);

-- Function to enforce 100 feed limit per user
CREATE OR REPLACE FUNCTION check_feed_limit()
RETURNS TRIGGER AS $$
DECLARE
    feed_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO feed_count
    FROM feed_subscriptions
    WHERE user_id = NEW.user_id;

    IF feed_count >= 100 THEN
        RAISE EXCEPTION 'User has reached maximum feed limit of 100';
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_check_feed_limit
BEFORE INSERT ON feed_subscriptions
FOR EACH ROW
EXECUTE FUNCTION check_feed_limit();
```

#### 3. `feed_items` table

Internal queue for tracking RSS items during processing.

```sql
CREATE TABLE feed_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    feed_id UUID NOT NULL REFERENCES feeds(id) ON DELETE CASCADE,

    -- Item identifiers
    item_url TEXT NOT NULL,
    item_guid TEXT, -- RSS GUID if available

    -- Processing status
    processing_status VARCHAR(20) NOT NULL DEFAULT 'pending', -- 'pending', 'processing', 'completed', 'failed'
    content_hash VARCHAR(64), -- SHA-256 hash after processing
    content_service_id UUID, -- ID returned from Content Service

    -- Item metadata (raw from RSS)
    title TEXT,
    author TEXT,
    published_at TIMESTAMP WITH TIME ZONE,
    description TEXT,

    -- Content update tracking
    http_last_modified TEXT, -- Last-Modified header from article fetch
    http_etag TEXT, -- ETag header from article fetch
    last_checked_at TIMESTAMP WITH TIME ZONE, -- When we last checked for updates

    -- Error tracking
    retry_count INTEGER NOT NULL DEFAULT 0,
    last_error TEXT,

    -- Timestamps
    discovered_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    processed_at TIMESTAMP WITH TIME ZONE,

    -- Constraints
    CONSTRAINT chk_processing_status CHECK (
        processing_status IN ('pending', 'processing', 'completed', 'failed')
    ),
    CONSTRAINT unique_feed_item UNIQUE(feed_id, item_guid)
);

-- Indexes
CREATE INDEX idx_feed_items_status ON feed_items(processing_status, discovered_at);
CREATE INDEX idx_feed_items_feed ON feed_items(feed_id, discovered_at DESC);
CREATE INDEX idx_feed_items_hash ON feed_items(feed_id, content_hash)
    WHERE processing_status = 'completed';
```

#### 4. `content_outbox` table

Implements the Outbox Pattern for reliable content delivery to the Content Service.

```sql
CREATE TABLE content_outbox (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    feed_item_id UUID NOT NULL REFERENCES feed_items(id) ON DELETE CASCADE,

    -- Processed content ready for delivery
    content_payload JSONB NOT NULL, -- Full content payload for Content Service API
    user_ids UUID[] NOT NULL, -- Array of user IDs to deliver to

    -- Delivery status tracking
    delivery_status VARCHAR(20) NOT NULL DEFAULT 'pending', -- 'pending', 'sending', 'delivered', 'failed'
    retry_count INTEGER NOT NULL DEFAULT 0,
    max_retries INTEGER NOT NULL DEFAULT 6,
    next_retry_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_error TEXT,

    -- Response tracking
    content_service_id UUID, -- ID returned from Content Service on success

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    delivered_at TIMESTAMP WITH TIME ZONE,

    -- Constraints
    CONSTRAINT chk_delivery_status CHECK (
        delivery_status IN ('pending', 'sending', 'delivered', 'failed')
    ),
    CONSTRAINT chk_retry_count CHECK (retry_count >= 0)
);

-- Indexes for efficient delivery worker queries
CREATE INDEX idx_outbox_pending ON content_outbox(next_retry_at)
    WHERE delivery_status IN ('pending', 'sending');
CREATE INDEX idx_outbox_status ON content_outbox(delivery_status, created_at);
CREATE INDEX idx_outbox_feed_item ON content_outbox(feed_item_id);
```

### Schema Design Notes

**Service Separation**
- Content Service database contains: `contents`, `user_contents`
- RSS Fetcher database contains: `feeds`, `feed_subscriptions`, `feed_items`, `content_outbox`
- Services communicate exclusively via REST APIs, never direct database access

**Deduplication Implementation**
- Content Service uses composite index on `content_hash` + `source_feed_id` for RSS deduplication
- Dedicated `source_feed_id` column is more efficient than JSONB extraction for queries
- RSS Fetcher tracks items by `feed_id` + `item_guid` to avoid re-processing the same item
- Future scrapers can use `canonical_url` for URL-based deduplication

**Orphaned Content Lifecycle**
- `orphaned_at` timestamp automatically set via trigger when last user removes content
- Background cleanup job deletes content where `orphaned_at < NOW() - INTERVAL '90 days'`
- Trigger automatically clears `orphaned_at` if content is re-saved by any user

**Feed Polling Management**
- `polling_tier` field determines fetch frequency (active/moderate/quiet)
- `next_poll_at` enables efficient scheduling queries
- Background job updates tiers based on `last_published_at` timestamps

**Metadata Flexibility**
- `metadata` JSONB column in `contents` stores source-specific data (e.g., `feed_id` for RSS content)
- Allows future fetcher services to add custom metadata without schema changes
- Indexed for efficient RSS deduplication queries

**Full-Text Search**
- GIN index on `contents` table enables efficient text search on title and author fields
- Uses PostgreSQL `to_tsvector` with English language configuration
- Supports the basic search API requirement without external search engine

**URL Canonicalization** (Future Enhancement)
- `canonical_url` column stores normalized URLs for deduplication
- Normalization includes: lowercase scheme/host, remove tracking params, consistent trailing slashes
- Enables cross-source deduplication for future web scraper fetcher

**Feed Items Cleanup**
- Completed `feed_items` older than 7 days are deleted by background job
- Failed `feed_items` older than 30 days are archived/deleted
- Prevents unbounded table growth

**Performance Optimizations**
- Strategic indexes for common query patterns (user content lists, feed polling schedules)
- Partial indexes for selective filtering (favorites, orphaned content, active feeds)
- Dedicated `source_feed_id` column avoids JSONB extraction in deduplication queries
- Cursor-based pagination supported via `added_at` timestamp ordering

## Authentication & Authorization

**Note**: Authentication is **NOT** part of this initial implementation.

The system integrates with a separate authentication service that uses JWT tokens. Authentication requirements will be added in a separate specification.

For initial development:
- APIs can be unprotected or use placeholder auth
- Design with auth in mind (user IDs in requests, etc.)

## Out of Scope (Initial Release)

The following features are explicitly out of scope for the initial implementation:

- JWT token validation and user authentication
- Image downloading and hosting
- Full-text search (basic filtering only)
- Content recommendations or discovery algorithms
- Social features (sharing, comments, etc.)
- Mobile app development (backend only)
- Web interface
- Analytics and usage tracking
- Email notifications
- Import from other read-it-later services

## Success Criteria

The initial implementation is considered successful when:

1. ✅ Users can subscribe to RSS feeds
2. ✅ RSS Fetcher automatically polls feeds and extracts content
3. ✅ New RSS items are auto-delivered to subscribed users
4. ✅ Content is deduplicated (same feed + same content = one record)
5. ✅ Multiple users can save the same content with individual metadata
6. ✅ Users can retrieve their content list with proper metadata
7. ✅ Users can update status, scroll position, and favorite flag
8. ✅ Content is stored as cleaned HTML with metadata
9. ✅ All services communicate via REST APIs
10. ✅ PostgreSQL is used for persistence

## Future Enhancements

Potential additions for future iterations:

### Content Features
- Additional fetcher services (web scraper, Pocket import, email forwarding)
- Offline content caching
- Reading time estimation
- Related content suggestions
- Full-text search with PostgreSQL FTS or external search engine

### User Features
- Tags and collections
- Highlights and annotations
- Reading statistics and streaks
- Export functionality (Markdown, PDF, EPUB)
- Content sharing between users

### Technical Improvements
- Tiered feed polling (more frequent for active feeds, less for inactive)
- Image downloading and CDN hosting
- Content archiving (snapshot at save time)
- Rate limiting and abuse prevention
- Monitoring and observability
- Horizontal scaling and load balancing
- Caching layer (Redis)
- Distributed locking for feed polling (required when scaling RSS Fetcher to multiple instances)
  - Prevents multiple instances from polling the same feed simultaneously
  - Options: PostgreSQL advisory locks, Redis-based distributed locks, or optimistic locking with `locked_by`/`locked_until` columns

### Resilience Enhancements
- Enhanced circuit breaker configuration
  - Configurable thresholds (e.g., open after 5 consecutive failures)
  - Half-open state after configurable timeout (e.g., 30 seconds)
  - Per-endpoint circuit breakers for fine-grained control
  - Metrics and alerting for circuit breaker state changes
- Dead letter queue for failed RSS items
  - Move permanently failed `feed_items` to a `feed_items_dlq` table after max retries
  - Store failure reason, timestamps, and original payload for investigation
  - Admin API to view, retry, or discard dead letter items
  - Enables root cause analysis of persistent failures
- Graceful degradation chain for content extraction
  - Level 1: Full readability parsing with all metadata
  - Level 2: Raw HTML extraction (if readability fails)
  - Level 3: RSS description/summary only (if HTML fetch fails or exceeds size limit)
  - Level 4: Title-only placeholder with link to original (if all else fails)
  - Each level logs degradation reason for monitoring

### Security Enhancements
- Comprehensive rate limiting
  - API endpoints: 100 requests/minute per user
  - Feed subscriptions: 10 per minute per user
  - Bulk operations: 5 per minute per user
  - Configurable limits via environment variables
  - Return `429 Too Many Requests` with `Retry-After` header
- Input validation limits
  - Maximum URL length: 2048 characters
  - Maximum title length: 1000 characters
  - Maximum description length: 10000 characters
  - Maximum feeds per request: 10
  - Maximum search query length: 500 characters
- Content Security Policy recommendations
  - Return recommended CSP headers in API documentation for mobile WebView rendering
  - Suggested policy: `default-src 'self'; img-src https: data:; style-src 'unsafe-inline'`
  - Document CSP best practices for rendering stored HTML content

### Additional Features
- Content versioning
  - Store content revision history in `content_versions` table
  - Track what changed between versions (diff storage)
  - Allow users to view previous versions of updated content
  - Configurable retention (e.g., keep last 5 versions)
- Feed discovery and validation caching
  - Cache feed validation results for 1 hour
  - Avoid redundant HTTP requests when multiple users subscribe to same feed simultaneously
  - Store feed metadata (title, description, icon) on first validation
  - Refresh cached metadata periodically (e.g., weekly)

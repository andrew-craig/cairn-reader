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
- **Original URL**: Store the source URL for reference
- **Content Hash**: Generate hash for deduplication purposes

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
- **Scroll Position**: Integer representing reading progress (pixel or percentage)
- **Favorite**: Boolean flag for favorited items
- **Added Timestamp**: When the user saved/received this content

#### Future Enhancements
The following features are planned for future iterations:
- User tags
- Highlights and annotations
- Personal notes
- Reading time estimates
- Custom folders/collections

### Content Deduplication

Deduplication is handled on a **per-fetcher basis** to accommodate different fetcher strategies:

#### RSS Fetcher Deduplication
- Deduplicate based on: `feed_id` + `content_hash`
- If the same article appears in the same feed, store only once
- If different users subscribe to the same feed, link to the same content record

#### Future Fetchers
- Each fetcher type can implement its own deduplication strategy
- Examples: URL canonicalization for web scraper, API-specific IDs, etc.

### Content Retrieval API

Provide REST APIs for the mobile app to:
- Retrieve user's content list with filtering (by status, favorite, date range)
- Get full content for a specific article
- Update user-content metadata (status, scroll position, favorite)
- Delete content from user's list (soft delete - keep content record if other users have it)
- Search user's saved content

**Response Format**: JSON with HTML content embedded in a designated field

### Data Model Considerations

The system should maintain:
1. **Content Table**: Stores unique content items (shared across users)
2. **User-Content Junction Table**: Maps users to content with user-specific metadata
3. **Feed Subscriptions Table**: Tracks which users subscribe to which RSS feeds

## RSS Fetcher Service Requirements

### Feed Management

- **User Subscriptions**: Individual users can subscribe to RSS feeds via the mobile app API
- **Feed Registration**: When a user subscribes to a feed, the system should:
  - Check if the feed already exists in the database
  - If new, validate the feed URL and add it to the monitoring list
  - Subscribe the user to that feed

### Feed Processing

- **Periodic Polling**: Fetch RSS feeds on a regular schedule (exact interval TBD)
- **Content Extraction**: For each new item in a feed:
  1. Fetch the full article content from the source URL
  2. Pass through readability parser to extract clean HTML
  3. Extract metadata (title, author, date, description, images)
  4. Generate content hash for deduplication

### Auto-Delivery

When new content is discovered in an RSS feed:
- Automatically add it to **all subscribed users'** reading lists
- Set initial status as `unread`
- Deduplicate: If content already exists (same feed_id + content_hash), link to existing content record

### Communication with Content Service

- Use **REST API** to communicate with the Content Service
- The RSS Fetcher can maintain an **internal queue** for processing feeds and items
- API endpoints needed from Content Service:
  - Add/update content
  - Check for existing content (deduplication)
  - Add content to user's reading list
  - Bulk operations for efficiency

## Technical Stack

### Database
- **PostgreSQL**: Primary data store for all services
- Each service may have its own database/schema, or share a database with clear separation

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
- Content expiration or cleanup policies
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
- Image downloading and CDN hosting
- Content archiving (snapshot at save time)
- Rate limiting and abuse prevention
- Monitoring and observability
- Horizontal scaling and load balancing
- Caching layer (Redis)

## Open Questions

1. **Feed Polling Frequency**: How often should RSS feeds be checked? (Every 15 minutes, hourly, daily?)
2. **Pagination**: What are the pagination requirements for content lists? (Page size, cursor vs offset)
3. **Content Limits**: Are there limits on content per user or content storage size?
4. **Feed Limits**: Max number of feeds per user?
5. **Error Handling**: What happens when an RSS feed is permanently unavailable?
6. **Retention**: How long should content be retained if no users have it saved?
7. **Deployment**: What's the target deployment environment? (Docker, Kubernetes, VMs?)
# RSS Fetcher Service - API Reference

## Phase 2.2: Feed Subscription API

### Base URL
```
http://localhost:8081/api/v1
```

### Endpoints

#### 1. Subscribe to a Feed
Subscribe a user to an RSS/Atom feed.

**Endpoint:** `POST /users/:user_id/feeds/subscribe`

**Request Body:**
```json
{
  "feed_url": "https://example.com/rss"
}
```

**Response (201 Created):**
```json
{
  "subscription_id": "uuid",
  "feed_id": "uuid",
  "feed_url": "https://example.com/rss",
  "feed_title": "Example Feed",
  "is_new_feed": true,
  "subscribed_at": "2025-01-15T10:00:00Z"
}
```

**Error Responses:**
- `400 Bad Request` - Invalid feed URL or feed limit reached (100 max)
- `409 Conflict` - Already subscribed to this feed
- `500 Internal Server Error` - Server error

**Example:**
```bash
curl -X POST http://localhost:8081/api/v1/users/123e4567-e89b-12d3-a456-426614174000/feeds/subscribe \
  -H "Content-Type: application/json" \
  -d '{"feed_url": "https://example.com/rss"}'
```

---

#### 2. List User's Feed Subscriptions
Get all feeds a user is subscribed to.

**Endpoint:** `GET /users/:user_id/feeds`

**Response (200 OK):**
```json
{
  "subscriptions": [
    {
      "subscription_id": "uuid",
      "feed_id": "uuid",
      "feed_url": "https://example.com/rss",
      "feed_title": "Example Feed",
      "feed_status": "active",
      "polling_tier": "active",
      "last_fetched_at": "2025-01-15T10:00:00Z",
      "subscribed_at": "2025-01-15T09:00:00Z"
    }
  ],
  "count": 1
}
```

**Example:**
```bash
curl http://localhost:8081/api/v1/users/123e4567-e89b-12d3-a456-426614174000/feeds
```

---

#### 3. Unsubscribe from a Feed
Remove a user's subscription to a feed.

**Endpoint:** `DELETE /users/:user_id/feeds/:feed_id`

**Response (200 OK):**
```json
{
  "message": "Successfully unsubscribed from feed",
  "success": true
}
```

**Error Responses:**
- `404 Not Found` - Subscription not found
- `500 Internal Server Error` - Server error

**Example:**
```bash
curl -X DELETE http://localhost:8081/api/v1/users/123e4567-e89b-12d3-a456-426614174000/feeds/456e7890-e89b-12d3-a456-426614174000
```

---

#### 4. Re-enable a Disabled Feed
Re-activate a feed that was automatically disabled due to errors.

**Endpoint:** `PATCH /feeds/:feed_id/enable`

**Response (200 OK):**
```json
{
  "message": "Feed successfully enabled",
  "success": true
}
```

**Error Responses:**
- `400 Bad Request` - Feed is already active
- `404 Not Found` - Feed not found
- `500 Internal Server Error` - Server error

**Example:**
```bash
curl -X PATCH http://localhost:8081/api/v1/feeds/456e7890-e89b-12d3-a456-426614174000/enable
```

---

### Health Check

**Endpoint:** `GET /health`

**Response (200 OK):**
```json
{
  "status": "ok",
  "service": "rss-fetcher-service",
  "phase": "2.2"
}
```

**Example:**
```bash
curl http://localhost:8081/health
```

---

## Features Implemented

1. **Feed Validation** - Validates RSS/Atom feeds before subscription
2. **Feed Metadata Extraction** - Automatically extracts title, description, and site URL
3. **Feed Limit Enforcement** - Maximum 100 feeds per user
4. **Duplicate Detection** - Prevents duplicate subscriptions to the same feed
5. **Feed Reuse** - Multiple users can subscribe to the same feed
6. **Error Handling** - Comprehensive error responses with clear messages
7. **Logging Middleware** - All requests are logged with duration and status
8. **Recovery Middleware** - Panic recovery prevents server crashes

## Environment Variables

- `PORT` - Server port (default: 8081)
- `DB_HOST` - Database host (default: localhost)
- `DB_PORT` - Database port (default: 5433)
- `DB_USER` - Database user (default: postgres)
- `DB_PASSWORD` - Database password (default: postgres)
- `DB_NAME` - Database name (default: rss_fetcher)

## Running the Service

```bash
# Build the service
go build -o bin/rss-fetcher-service ./cmd/server

# Run with default settings
./bin/rss-fetcher-service

# Or run with custom environment
DB_HOST=localhost DB_PORT=5434 DB_USER=cairn_rss DB_PASSWORD=cairn_rss_pass DB_NAME=cairn_rss ./bin/rss-fetcher-service
```

## Next Steps (Phase 2.3)

The next phase will implement the Content Service Client to enable communication between RSS Fetcher Service and Content Service.

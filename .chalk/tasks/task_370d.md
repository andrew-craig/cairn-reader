---
id: task_370d
title: Backend: Add POST /api/v1/content/discover-feed endpoint
type: task
status: in_progress
priority: 2
labels: []
blocked_by: []
parent: feature_c95a
remote_task_url: null
created_at: 2026-04-10T08:08:06Z
updated_at: 2026-04-12T22:32:35Z
---
Add a new unprotected endpoint to the Content Service for feed discovery.

## Endpoint

```
POST /api/v1/content/discover-feed
Content-Type: application/json

Request:  {"url": "https://example.com"}
Response: {"data": {"feeds": [{"url": "https://example.com/feed.xml", "title": "Example Blog"}]}}
```

## Implementation

### Files to modify

- `services/read/content/internal/api/dto/detection.go` — add `DiscoverFeedRequest` and `DiscoverFeedResponse` DTOs
- `services/read/content/internal/api/handlers/detection_handler.go` — add `DiscoverFeed` handler method
- `services/read/content/internal/api/router.go` — register `r.Post("/discover-feed", detectionHandler.DiscoverFeed)` next to the existing `/detect` route
- `services/read/content/api/openapi.yaml` — document the new endpoint

### Handler logic

1. Decode and validate request (URL required)
2. Call `urlDetector.DiscoverFeeds(ctx, url)` from task_59f7
3. Return the list of discovered feeds (empty array if none found, never an error to the client)

### Tests

- Handler unit tests in `detection_handler_test.go`:
  - Valid request returns discovered feeds
  - Valid request with no feeds returns empty array
  - Missing URL returns 400
  - Invalid JSON returns 400

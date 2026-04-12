---
id: task_31b1
title: Mobile: Add discoverFeed service method and types
type: task
status: closed
priority: 2
labels: []
blocked_by: []
parent: feature_c95a
remote_task_url: null
created_at: 2026-04-10T08:08:35Z
updated_at: 2026-04-12T22:58:43Z
---
Add the mobile service method and TypeScript types for calling the new discover-feed backend endpoint.

## Files to modify

### `apps/mobile/src/types/read.ts`

Add types:

```typescript
export interface DiscoveredFeed {
  url: string;
  title: string;
}

export interface DiscoverFeedResponse {
  feeds: DiscoveredFeed[];
}
```

### `apps/mobile/src/services/read.ts`

Add method:

```typescript
/**
 * Discover RSS/Atom feeds associated with a page URL.
 * Returns an array of discovered feeds (may be empty).
 */
static async discoverFeed(url: string): Promise<DiscoverFeedResponse> {
  // POST to /api/v1/content/discover-feed (no auth required)
  // 15s timeout (matching backend discovery timeout)
  // On error, return { feeds: [] } rather than throwing
}
```

## Notes

- No authentication required (matches `/detect` pattern)
- Use `AbortController` with 15s timeout (same pattern as `detectURL`)
- Graceful failure: return empty feeds array on network error

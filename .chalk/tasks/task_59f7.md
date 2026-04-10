---
id: task_59f7
title: Backend: Add feed discovery service to url_detector.go
type: task
status: open
priority: 2
labels: []
blocked_by: []
parent: feature_c95a
remote_task_url: null
created_at: 2026-04-10T08:07:54Z
updated_at: 2026-04-10T08:07:54Z
---
Add a `DiscoverFeeds` method to the `URLDetector` interface in `services/read/content/internal/service/url_detector.go`.

## Input/Output

```go
type DiscoveredFeed struct {
    URL   string `json:"url"`
    Title string `json:"title"`
}

DiscoverFeeds(ctx context.Context, pageURL string) ([]DiscoveredFeed, error)
```

## Discovery Strategy (ordered)

1. **Direct parse**: Try parsing the URL itself as a feed (it may already be one)
2. **HTML `<link>` tags**: Fetch the page, parse HTML for `<link rel="alternate" type="application/rss+xml">` and `<link rel="alternate" type="application/atom+xml">` tags. Extract `href` and `title` attributes
3. **Common path probing**: Resolve these paths against the page's base URL and try fetching + parsing each:
   - `/feed`
   - `/rss`
   - `/rss.xml`
   - `/atom.xml`
   - `/feed.xml`
   - `/index.xml`
   - `/feed/rss`
   - `/feed/atom`
   - `.rss` (appended to path)

## Implementation Notes

- Reuse the existing `gofeed.Parser` and `httpClient` already in `urlDetectorImpl`
- Limit total discovery time (15s context timeout)
- Probe common paths concurrently with a bounded worker pool
- Deduplicate results by final URL (after redirects)
- Return empty slice (not error) when no feeds found
- Limit response body reads to prevent memory issues (reuse existing 10MB limit)

## Tests

- Unit tests in `url_detector_test.go` using httptest servers:
  - Page with `<link rel="alternate">` tags → discovers feed
  - Page with no feeds → returns empty
  - URL that is itself a feed → returns it
  - Common path `/feed` works when HTML has no link tags
  - Deduplication when HTML link and common path point to same URL

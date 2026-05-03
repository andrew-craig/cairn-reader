---
id: task_69fd
title: Migrate Read fetcher and Content Service to pkg/rss/
type: task
status: open
priority: 2
labels: []
blocked_by: [task_5ee6]
parent: epic_6d46
remote_task_url: null
created_at: 2026-05-02T04:18:35Z
updated_at: 2026-05-03T04:21:03Z
---
# Replace Read's RSS/sanitize/readability internals with pkg/rss/

## Why

Once pkg/rss/ is proven by the Explore migration (task_5ee6), Read can adopt it. Read is the harder migration because of conditional fetch, outbox, tiered scheduling, and the (now consolidated) single readability pass.

## Requirements

### services/read/fetcher/

- Replace services/read/fetcher/internal/fetcher/parser.go with pkg/rss/parse.
- Replace services/read/fetcher/internal/fetcher/feed_fetcher.go HTTP client + services/read/fetcher/internal/fetcher/conditional_fetcher.go with pkg/rss/fetch (which now provides ETag/Last-Modified). Keep the ETag/Last-Modified persistence in the feeds table.
- Drop services/read/fetcher/internal/processor/item_processor.go's sanitizer and readability calls (already removed in task_7479) and any remaining gofeed/bluemonday imports.
- Switch content-hash computation to pkg/rss/hash.

### services/read/content/

- Replace services/read/content/internal/processor/readability.go with pkg/rss/readability.
- Replace services/read/content/internal/processor/sanitizer.go with pkg/rss/sanitize.
- Keep services/read/content/internal/processor/url_canonicalizer.go - it's not RSS-specific.
- services/read/content/internal/processor/content.go becomes a thin orchestrator that calls pkg/rss/readability.Extract -> pkg/rss/sanitize.Sanitize -> pkg/rss/hash.ContentHash.

### Out of scope

- No changes to services/read/fetcher/internal/scheduler/, internal/worker/, internal/repository/, internal/jobs/.
- No changes to services/read/content/ user_content_handler, JWT middleware, or DB schema.

## Tests

- All existing unit and integration tests under services/read/fetcher/ and services/read/content/ pass after refactor without behavioral changes.
- Integration test: full RSS poll -> outbox -> Content Service -> mobile-shaped response, with title/author preserved.

## Acceptance

- grep -r 'gofeed\|bluemonday\|go-readability' services/read/ returns no matches outside _test.go and pkg/rss/ imports.
- Polling cadence (tier1/tier2/tier3 timings) unchanged.
- Outbox throughput in staging unchanged within noise after one full polling cycle.
- Mobile app continues to display title/author correctly for RSS-ingested articles.

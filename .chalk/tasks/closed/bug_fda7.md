---
id: bug_fda7
title: Plumb RSS title and author through fetcher -> outbox -> Content Service
type: bug
status: closed
priority: 0
labels: []
blocked_by: []
parent: epic_6d46
remote_task_url: null
created_at: 2026-05-02T04:17:10Z
updated_at: 2026-05-02T04:38:22Z
---
# Stop dropping RSS title/author on the way to the Content Service

## Why

The Read fetcher's item processor already stores RSS-parsed title and author in the outbox payload (services/read/fetcher/internal/processor/item_processor.go:154-163), but they never reach the Content Service. The mobile app then renders 'Unknown' for author and a blank title.

Three layers silently drop them:
1. services/read/fetcher/internal/worker/outbox_worker.go:271-308 (buildContentItem) only reads source_url, cleaned_html, source_feed_id, published_at from the payload.
2. services/read/fetcher/internal/client/content_service_client.go:52-58 (BulkContentItem) has no Title/Author fields.
3. services/read/content/internal/api/dto/bulk.go:14-20 (BulkContentItem) has no Title/Author fields.

## Requirements

- Add Title *string and Author *string fields to BulkContentItem in:
  - services/read/fetcher/internal/client/content_service_client.go
  - services/read/content/internal/api/dto/bulk.go (with validation that title <= 1000 chars, author <= 255 chars)
  - services/read/content/internal/service/content_service.go (service-layer struct)
- Update services/read/fetcher/internal/worker/outbox_worker.go buildContentItem() to extract title/author from entry.ContentPayload.
- Update services/read/content/internal/api/handlers/bulk_handler.go BulkCreateContent to pass them through to the service.
- Update services/read/content/internal/service/content_service.go BulkCreateFromHTML to prefer caller-supplied Title/Author over readability-extracted values; fall back to readability only when caller did not supply them.

## Tests

- Unit test in outbox_worker_test.go: payload with title/author round-trips into BulkContentItem.
- Unit test in content_service_test.go: caller-supplied title wins over readability output; missing title falls back to readability.
- Integration test (services/read/fetcher/integration_test.go) updated to assert Content Service receives the RSS title/author.

## Acceptance

- After deploying both services, a fresh RSS-ingested article appears in mobile with non-empty title and the feed-supplied author (not 'Unknown'), without changing any other behavior.
- No DB migration needed (contents.title is already TEXT NOT NULL, author is VARCHAR(255) nullable).

## Branch

claude/fix-rss-article-metadata-RWpre (already created).

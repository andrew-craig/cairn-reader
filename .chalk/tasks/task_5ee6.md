---
id: task_5ee6
title: Migrate Explore fetcher to pkg/rss/
type: task
status: open
priority: 2
labels: []
blocked_by: []
parent: epic_6d46
remote_task_url: null
created_at: 2026-05-02T04:18:21Z
updated_at: 2026-05-03T04:21:03Z
---
# Replace Explore fetcher's RSS internals with pkg/rss/

## Why

Explore is the simpler caller of the two RSS pipelines (no conditional fetch, no outbox, fixed-interval scheduling), so migrating it first validates the pkg/rss/ API before we touch Read.

## Requirements

- Delete services/explore/fetcher/internal/sanitizer/ and replace usages with pkg/rss/sanitize.
- Delete the gofeed parsing inside services/explore/fetcher/internal/fetcher/fetcher.go and replace with pkg/rss/parse + pkg/rss/fetch.
- Replace the ad-hoc HTTP fetch (gofeed default) with pkg/rss/fetch using the canonical User-Agent. Adding ETag/Last-Modified support to Explore is in scope - record the values somewhere appropriate (existing fetch_history table or new columns on feeds).
- Switch content-hash computation to pkg/rss/hash. Note: Explore currently hashes the link; this task changes it to hash the cleaned content for consistency with Read. Add a one-time DB migration to invalidate stale hashes if dedup is keyed on them.
- Keep services/explore/fetcher/internal/sync/ and the recommender_client untouched - those are Explore-specific.

## Tests

- Existing fetcher tests pass after refactor (services/explore/fetcher/internal/fetcher/fetcher_test.go).
- Add a test that confirms ETag/Last-Modified are sent on subsequent fetches.

## Acceptance

- grep -r 'gofeed\|bluemonday' services/explore/ returns no matches outside _test.go and pkg/rss/ imports.
- Recommender ingestion volume in staging matches pre-migration baseline (within 5%) for one full polling cycle.

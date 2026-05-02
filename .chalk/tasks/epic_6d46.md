---
id: epic_6d46
title: Consolidate RSS feed handling into shared pkg/rss/ and fix metadata loss
type: epic
status: open
priority: 1
labels: []
blocked_by: []
parent: null
remote_task_url: null
created_at: 2026-05-02T04:16:55Z
updated_at: 2026-05-02T04:16:55Z
---
# Re-architect RSS ingestion to share code and stop dropping article metadata

## Background

We currently run two independent RSS fetcher services and a third readability+sanitize pass in the Content Service. Investigation of the "no title / Unknown author" mobile bug surfaced two related problems:

1. **Metadata loss between Read fetcher and Content Service.** The fetcher already has the RSS-parsed `title` and `author` (in `feed_items` and the outbox payload), but the outbox worker and the bulk DTOs don't carry them. The Content Service then tries to re-derive title/author from HTML that was already sanitized upstream, so readability finds nothing and the mobile app renders "Unknown".

2. **Code duplication and drift between Explore fetcher, Read fetcher, and Content Service.** All three reach for gofeed, bluemonday, and go-readability, but with three different User-Agents, divergent bluemonday policies (audio/video/class/id keep-vs-strip), different content-hash inputs (Explore hashes link, Read hashes content), and conditional GET only implemented in Read.

## Goal

One canonical RSS pipeline shared between Explore and Read; readability runs exactly once per article; RSS-supplied metadata (title, author, published_at) is preserved end-to-end and trusted over re-derived values.

## Scope

In scope:
- Plumb title/author through fetcher -> outbox -> Content Service (immediate bug fix)
- Collapse Read's double-readability pipeline to a single pass
- Extract pkg/rss/ with parse/, sanitize/, readability/, fetch/ subpackages
- Migrate Explore fetcher to pkg/rss/
- Migrate Read fetcher + Content Service to pkg/rss/
- Pick one canonical sanitizer policy and one User-Agent

Out of scope:
- Scheduler logic (tiered polling stays in Read; fixed-interval stays in Explore)
- Outbox pattern (Read-specific)
- Subscription / user-feed model (Read-specific)
- Recommender HTTP client (Explore-specific)
- New feed sources (RSS only)

## Success criteria

- Mobile app displays the correct title and author for RSS-ingested articles (no more "Unknown" except where the feed legitimately omits the author).
- Readability is invoked at most once per article ingest.
- Explore and Read share a single bluemonday policy and a single User-Agent constant.
- pkg/rss/ has unit tests covering parse, sanitize, readability, and conditional fetch.
- No regressions in feed polling cadence, dedup behavior, or recommender ingestion volume.

## Phasing

See sub-tasks. Order matters: the title/author plumbing fix lands first (unblocks the user-visible bug), then the readability consolidation, then the pkg/rss/ extraction, then the per-service migrations.

---
id: task_8bc5
title: [409→500] Client matches error codes the server never sends → re-subscribe returns 500 instead of 409
type: task
status: closed
priority: 2
labels: [quality,wave2,contracts]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-09T06:46:25Z
updated_at: 2026-08-16T05:40:56Z
---
Read docs/QUALITY_REMEDIATION_STRATEGY.md §0 (rules of engagement) and §2.6 (definition of done) before starting. Read the full finding text in docs/CODE_QUALITY_REVIEW.md. One finding, one branch, one PR. Re-verify on main first — cited line numbers are from 2026-07-05 and drift.

**Finding:** 409→500 mistranslation (Part 1 read-M6, confirmed end-to-end in Part 2b) | **Wave 2** | **Recipe:** R8 (strategy §2.5) | **Test level:** real server handler via httptest, real client pointed at it
**Touches:** services/read/content/internal/service/ingest_rss_client.go, user_content_handler.go
**Blocked by:** the P2-C8 task — same file, same client; land that first to avoid a conflicting diff.

## Problem
The client matches error **codes** (`"already_subscribed"`) that the server never sends — ingest-rss sends generic `"conflict"` / `"bad_request"` with the human text only in `message`. A downstream exact-string `switch` misses too, so re-subscribing to a feed you already have returns **500 instead of 409**. Ordinary user conditions surface as server errors.

## What to do
1. Reproduce at the seam: real server handler via `httptest`, real client, trigger the duplicate-subscribe condition; assert the app sees 409. Fails on main.
2. Fix by branching on HTTP **status codes**, not error-code strings. Reference implementation: `Unsubscribe` in the same ingest-rss client — it already does this correctly.
3. Check the other error branches in the same client for the same string-matching pattern.

## Done when
- 409-shaped server responses surface as 409 to the app, proven by a seam test; no error-string matching remains in the client.

## Review

Re-verified on `main`: the client (`ingest_rss_client.go:118-131`, pre-fix) matched
`apiError.Error == "already_subscribed"`/`"feed_limit_reached"`/`"invalid_feed"`, but the real
`fetcher` subscribe handler sends the generic `pkg/api` codes `"conflict"`/`"bad_request"` with
the specific reason only in `message`. Every branch missed, so all three conditions fell through
to the generic wrapped error `"ingest RSS service error: <message>"`. Downstream,
`user_content_handler.go`'s `handleFeedSubmission` then switched on the *exact* string
`"already subscribed to this feed"` etc., which never matched the wrapped prefix either — so
re-subscribing to an existing feed (and the feed-limit/invalid-feed cases) all returned 500.

Fix: `SubscribeUserToFeed` now branches on HTTP status only (mirroring `Unsubscribe`'s pattern in
the same client) and returns sentinel errors — `ErrAlreadySubscribed` for 409,
`ErrInvalidSubscribeRequest` (wrapping the server's message) for 400. `handleFeedSubmission` now
uses `errors.Is` against those sentinels instead of string-matching.

Tests added (fail before the fix, pass after):
- `ingest_rss_client_test.go: TestSubscribeUserToFeed_AlreadySubscribedSurfacesAsSentinel` —
  client seam test against a fake server reproducing the real 409 response shape.
- `user_content_handler_test.go: TestAddContentToUser_FeedAlreadySubscribed` — full handler test
  with the real `IngestRSSClient` pointed at the fake server; asserts the app sees 409.

Verified: `gofmt -l .` clean, `go vet ./...` clean, `golangci-lint run ./...` clean,
`go test -race -count=1 ./...` green (content service). No OpenAPI/CLAUDE.md changes needed — the
409 response was already documented correctly; only the client's internal code-matching was wrong.

PR: https://github.com/andrew-craig/cairn-reader/pull/331


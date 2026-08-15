---
id: task_8bc5
title: [409→500] Client matches error codes the server never sends → re-subscribe returns 500 instead of 409
type: task
status: open
priority: 2
labels: [quality,wave2,contracts]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-09T06:46:25Z
updated_at: 2026-08-15T10:45:21Z
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


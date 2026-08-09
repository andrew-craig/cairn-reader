---
id: task_963d
title: [P2-C8] Feed-subscribe client unmarshals the wrong response shape → app gets empty data with HTTP 201
type: task
status: open
priority: 1
labels: [quality,wave2,contracts]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-09T06:46:24Z
updated_at: 2026-08-09T06:46:24Z
---
Read docs/QUALITY_REMEDIATION_STRATEGY.md §0 (rules of engagement) and §2.6 (definition of done) before starting. Read the full finding text in docs/CODE_QUALITY_REVIEW.md. One finding, one branch, one PR. Re-verify on main first — cited line numbers are from 2026-07-05 and drift.

**Finding:** P2-C8 | **Wave 2** | **Recipe:** R8 (strategy §2.5) | **Test level:** real server handler via httptest, real client pointed at it
**Touches:** services/read/content/internal/service/ingest_rss_client.go, user_content_handler.go

## Problem
`ingest_rss_client.go:144-151` unmarshals the subscribe response into a **nested** `{subscription{…}, feed{…}}` struct, but ingest-rss actually returns a **flat** DTO (`subscription_id` / `feed_id` / `feed_url` / `feed_title` / …) wrapped in the `pkg/api.WriteSuccess` `{data,meta}` envelope, which the client never unwraps here. `json.Unmarshal` does not error on missing keys, so it "succeeds" with a zero-valued struct that propagates through `user_content_handler.go:366-380` to the app: the feed *is* created, but the client receives `feed_id:""`, `title:""`, `subscribed_at:0001-01-01` with a real 201 and no error anywhere.

The sibling `ListUserSubscriptions` in the same file unwraps correctly — this is an isolated regression. Classic bug that unit tests miss because nothing fails.

## What to do
1. Reproduce at the seam: run the real ingest-rss server handler via `httptest`, point the real client at it, assert the decoded struct is **fully populated**. This must fail on current main.
2. Fix the client to unwrap the `pkg/api` `{data,meta}` envelope, matching `ListUserSubscriptions`.
3. Note the hand-maintained response struct as a Wave-4/5 candidate for the shared-type pattern (`pkg/models.Article` is the reference) — do **not** restructure in this bugfix PR.

## Done when
- Seam test fails before the fix and passes after; the app receives a populated subscription on 201.


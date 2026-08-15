---
id: task_7fbe
title: [C5] Outbox client's internal 1m→12h retry loop blocks the whole worker for up to ~17h
type: task
status: closed
priority: 1
labels: [quality,wave2,ops]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-09T06:46:25Z
updated_at: 2026-08-15T11:21:21Z
---
Read docs/QUALITY_REMEDIATION_STRATEGY.md §0 (rules of engagement) and §2.6 (definition of done) before starting. Read the full finding text in docs/CODE_QUALITY_REVIEW.md. One finding, one branch, one PR. Re-verify on main first — cited line numbers are from 2026-07-05 and drift.

**Finding:** C5 | **Wave 2** | **Recipe:** R6 (strategy §2.5) | **Test level:** inject a clock / derive delays from config; assert on the *schedule decision*, not wall time
**Touches:** services/read/email/internal/client/content_service_client.go, internal/worker/outbox_worker.go

## Problem
`content_service_client.go:21-28,146-169` — `DeliverContent` runs its **own** blocking retry loop (`time.After` over 1m / 5m / 15m / 1h / 4h / 12h) *inside* `deliverBatch`, which processes entries sequentially in a single ticker loop (`outbox_worker.go:70-88`). This duplicates and conflicts with the DB-level backoff that already exists and is correct. If the content service is degraded, the first failing entry stalls delivery **for every user** for up to roughly 17 hours.

The existing test patches `retryDelays` to millisecond values, so production timing is never exercised — that is exactly why this survived.

## What to do
1. Test first: first entry's downstream fails, assert entry 2 is attempted **within the same batch pass**. Do not patch delays to zero — that is the trap that hid the bug; assert on the schedule decision instead.
2. Delete the client's internal sleep-retry loop entirely. On failure: mark the entry, move on. The DB-level backoff handles the retry.

## Done when
- A failing downstream no longer blocks subsequent entries in a batch, proven by a test that fails before the fix.

## Review

Re-verified on main: `content_service_client.go` still ran its own `time.After` retry loop
(1m/5m/15m/1h/4h/12h) inside `DeliverContent`, on top of the DB-level backoff already applied
in `outbox_worker.go`'s `recordFailure`/`outboxBackoff`. Finding confirmed as described.

Added `TestOutboxWorker_DeliverBatch_FailedEntryDoesNotBlockSubsequentEntries` in
`internal/worker/outbox_worker_test.go`, which drives a real (unpatched) `ContentServiceClient`
against an `httptest` server where entry 1 always 500s and entry 2 succeeds, then asserts entry 2
is attempted within 2s of `deliverBatch` starting. This fails on unfixed code (times out — entry 1
blocks inside `DeliverContent` for a full minute before deliverBatch can move on) and passes after
the fix.

Fix: deleted `retryDelays`, the retry loop, and the now-unused `isNonRetryable` helper from
`DeliverContent` — it now makes exactly one attempt and returns the error immediately, letting the
outbox's DB-level backoff own retry timing. Updated
`TestContentServiceClient_DeliverContent_RetryOnServerError` (renamed to
`..._NoInternalRetryOnServerError`, now asserts exactly one call) and removed the `retryDelays`
patching from the circuit-breaker test, since there's no internal retry left to patch around.

Verification: `go test ./...` and `go vet ./...` green in `services/read/email`.


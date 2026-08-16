---
id: task_d1eb
title: [Cleanup batching] RunWithBatching does not batch — unbounded DELETE in one transaction
type: task
status: closed
priority: 2
labels: [quality,wave2,database]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-09T06:46:25Z
updated_at: 2026-08-16T11:44:33Z
---
Read docs/QUALITY_REMEDIATION_STRATEGY.md §0 (rules of engagement) and §2.6 (definition of done) before starting. Read the full finding text in docs/CODE_QUALITY_REVIEW.md. One finding, one branch, one PR. Re-verify on main first — cited line numbers are from 2026-07-05 and drift.

**Finding:** Cleanup batching (critical in the DB pass) | **Wave 2** | **Recipe:** R7 (strategy §2.5) | **Test level:** `//go:build integration` — seed rows above the batch size, assert multiple bounded passes
**Touches:** services/read/content/internal/repository/content.go, internal/jobs/cleanup_job.go, services/read/email/internal/repository/outbox.go, raw_email.go

## Problem
`DeleteOrphaned` (`repository/content.go:338-358`) has **no `LIMIT`**, and `RunWithBatching` (`jobs/cleanup_job.go:53-102`) loops it expecting batches — so the first call deletes everything in one long lock-holding, WAL-heavy transaction. The batching claim is false. The correct bounded pattern already exists in this repo at `services/read/fetcher/.../feed_item.go:379-401`.

The same unbounded-`DELETE` anti-pattern, at lower volume, is in email `outbox.go:248-268` and `raw_email.go:223-243`.

## What to do
1. Integration test: seed more rows than the batch size; assert the delete happens in multiple bounded passes and each transaction is small.
2. Plumb the batch size through to the SQL (`LIMIT`), copying `feed_item.go:379-401`. Apply to the two email repositories as well.

## Done when
- Integration test proves bounded batches; no `DELETE` in these paths runs without a `LIMIT`.

## Review

Re-verified on `main`: `DeleteOrphaned` had no `LIMIT` and `RunWithBatching` was dead code — the
worker actually wired `cleanupJob.Run` (unbounded, no batching at all) into cron, not
`RunWithBatching`. Fixed by copying the bounded `DELETE ... WHERE id IN (SELECT id ... LIMIT $n)`
pattern from `fetcher/.../feed_item.go:379-401` into all three repositories, and collapsing
`CleanupJob.Run`/`RunWithBatching` into a single `Run()` that always batches (mirrors
`FeedItemsCleanupJob`'s pattern — there's no unbounded variant to accidentally wire up).

- `content.DeleteOrphaned(ctx, olderThan, batchSize)` — bounded, default batch size 1000.
- `email/outbox.DeleteDelivered` and `email/raw_email.DeleteProcessed` — same fix, jobs now loop
  until a pass returns 0 instead of a single unbounded call.
- New integration test `TestContentRepository_DeleteOrphaned_Batching_Integration` (real Postgres):
  seeds 5 orphaned rows, batch size 2, asserts the first call deletes exactly 2 and full drain
  takes 3 passes. Verified it fails on the pre-fix query (deletes all 5 in one call).
- Unit tests updated/added for the batching loop (content `internal/jobs`, email `internal/jobs`)
  proving each job keeps calling until a zero-count pass.

No API/route changes, so no `openapi.yaml` or `CLAUDE.md` route-table updates needed.

`go vet`, `golangci-lint run`, and `go test -race ./...` all green for both `content` and `email`
modules; integration suite green modulo two pre-existing network-dependent tests
(`TestDetectURL_RealFeed`, `TestDetectURL_RealRSSFeed_Integration`, `TestDetectURL_RealHTMLPage_Integration`)
that fail in this sandbox for lack of outbound internet access — unrelated to this change.


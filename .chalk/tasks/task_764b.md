---
id: task_764b
title: [Audit/Tier 2] Delete the two dead cleanup schedulers in services/read/fetcher/internal/jobs
type: task
status: open
priority: 2
labels: [quality,deletion,audit]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-17T10:14:03Z
updated_at: 2026-08-17T10:14:03Z
---
**Source:** Cairn Simplification Audit (read-only pass at HEAD `a6c56a1`, 2026-08-16) — https://claude.ai/code/artifact/286883fb-3f93-49c4-942f-4880251a409f
Read docs/QUALITY_REMEDIATION_STRATEGY.md §0 (rules of engagement) and §2.6 (definition of done) before starting. One finding, one branch, one PR. Re-verify before fixing — all file:line references below were confirmed at `a6c56a1`.

**Audit tier:** 2 (deletions — near-zero risk, compiler-verified).

## Problem
Two scheduler types in `services/read/fetcher/internal/jobs` have **zero call sites** anywhere in the repo (no production caller, no test):
- `feed_items_cleanup_scheduler.go` — `FeedItemsCleanupSchedulerConfig`, `DefaultFeedItemsCleanupSchedulerConfig`, `FeedItemsCleanupScheduler`, `NewFeedItemsCleanupScheduler`, plus `Start`/`Stop`/`run`/`runCleanup`/`RunNow`
- `outbox_cleanup_scheduler.go` — the same shape for `OutboxCleanupScheduler`

`grep -rn 'NewOutboxCleanupScheduler\|NewFeedItemsCleanupScheduler'` matches **only the definitions**. Note neither file has a `_test.go` companion, while both *jobs* do.

**The jobs they wrap are live** — do not touch those:
- services/read/fetcher/cmd/ingest_rss_worker/main.go:87-88 — `jobs.NewOutboxCleanupJob(cfg.OutboxCleanup, outboxRepo)`, `jobs.NewFeedItemsCleanupJob(cfg.FeedItemsCleanup, feedItemRepo)`
- services/read/fetcher/selfhost/rss.go:129,132 — same two, via `rssJobs`

So the scheduling is done elsewhere and these two wrappers were orphaned.

## What to do
1. Re-confirm zero call sites for both constructors.
2. Delete the two `*_scheduler.go` files.
3. Remove any imports your deletion orphans — and nothing else.

## Done when
- Both files are gone, `services/read` builds, and the cleanup jobs still run from the worker and selfhost entrypoints exactly as before.

## Related
- The audit separately flags **duplicated cleanup jobs** (Tier 5) — `feed_items_cleanup_job.go` and `outbox_cleanup_job.go` are near-identical in shape (`Run`/`cleanup*`/`logMetrics`/`GetMetrics`/`RunWithCustomRetention`/`String`), and `services/read/email/internal/jobs/outbox_cleanup.go` is a third variant. That is a **separate, later** dedup task — this one only deletes the dead schedulers.

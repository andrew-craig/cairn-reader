---
id: task_764b
title: [Audit/Tier 2] Delete the two dead cleanup schedulers in services/read/fetcher/internal/jobs
type: task
status: closed
priority: 2
labels: [quality,deletion,audit]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-17T10:14:03Z
updated_at: 2026-08-29T23:29:45Z
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

## Related — corrected 2026-08-17 by the audit author

This task is the audit's **F-S10-1**. An earlier version of this note mapped the Tier 5 "duplicated cleanup jobs" item onto the fetcher's `*_job.go` files. **That was wrong** and would have sent you at live, cron-wired code. The correct mapping:

| Files | State | Finding | Action |
|---|---|---|---|
| fetcher `outbox_cleanup_job.go`, `feed_items_cleanup_job.go` | **live** — cron-wired at `cmd/ingest_rss_worker/main.go:87-103` and `selfhost/rss.go:129-138`, with tests | **in no finding** | **leave alone** |
| fetcher `outbox_cleanup_scheduler.go`, `feed_items_cleanup_scheduler.go` | dead, zero refs, no tests | **F-S10-1** | **this task** — delete |
| email `outbox_cleanup.go`, `raw_email_cleanup.go` | both live, duplicate **each other** | F-S06-2 | Tier 5 dedup — **task_24fa** |

So: delete only the two fetcher `*_scheduler.go` files here. The fetcher `*_job.go` files are not part of any finding, and the Tier 5 dedup is entirely within the **email** service.

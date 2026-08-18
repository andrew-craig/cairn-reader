---
id: task_24fa
title: [Audit F-S06-2/Tier 5] Merge email's duplicated outbox_cleanup and raw_email_cleanup jobs
type: task
status: open
priority: 3
labels: [quality,consolidation,audit]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-17T12:52:21Z
updated_at: 2026-08-17T12:52:21Z
---
**Source:** Cairn Simplification Audit — https://claude.ai/code/artifact/286883fb-3f93-49c4-942f-4880251a409f · file:line detail supplied by the audit author 2026-08-17 and re-verified at HEAD `a6c56a1`.
Read docs/QUALITY_REMEDIATION_STRATEGY.md §0 (rules of engagement) and §2.6 (definition of done) before starting. One finding, one branch, one PR.

**Audit finding:** F-S06-2 | **Audit tier:** 5 (low priority — hazard reduction, no live defect).

## Problem
`services/read/email/internal/jobs/outbox_cleanup.go` and `services/read/email/internal/jobs/raw_email_cleanup.go` **duplicate each other**. Both are **live** — this is a dedup, not a deletion.

## Scope — do not touch the fetcher jobs
The near-identically-named files under `services/read/fetcher/internal/jobs/` are a **different matter entirely**:

| Files | State | Finding | Action |
|---|---|---|---|
| fetcher `*_job.go` (`outbox_cleanup_job.go`, `feed_items_cleanup_job.go`) | **live**, cron-wired at `cmd/ingest_rss_worker/main.go:87-103` and `selfhost/rss.go:129-138`, with tests | **in no finding** | **leave alone** |
| fetcher `*_scheduler.go` | dead, zero refs, no tests | F-S10-1 | Tier 2 delete — **task_764b** |
| email `outbox_cleanup.go`, `raw_email_cleanup.go` | both live, duplicate each other | **F-S06-2** | **this task** |

## What to do
1. Diff the two email jobs; identify what genuinely differs (target table, retention semantics, metrics) versus what is copied.
2. Extract the shared cleanup shape, parameterised on the differences. One job type, two configurations, if that reads cleanly.
3. Keep both cron registrations working; this changes structure, not schedule.

## Done when
- One cleanup implementation in the email service, both jobs still running on their existing schedules, tests green.

## Priority note
The audit rates this Tier 5 with **no live defect**. Do not preempt Tier 0-1 work for it.

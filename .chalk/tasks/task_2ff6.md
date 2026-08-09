---
id: task_2ff6
title: [Dedup index] idx_contents_rss_dedup is not UNIQUE; email/manual content path has no dedup at all
type: task
status: open
priority: 1
labels: [quality,wave2,database]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-09T06:46:24Z
updated_at: 2026-08-09T06:46:24Z
---
Read docs/QUALITY_REMEDIATION_STRATEGY.md §0 (rules of engagement) and §2.6 (definition of done) before starting. Read the full finding text in docs/CODE_QUALITY_REVIEW.md. One finding, one branch, one PR. Re-verify on main first — cited line numbers are from 2026-07-05 and drift.

**Finding:** DB pass (dedup index) + P2-C8-adjacent idempotency | **Wave 2** | **Recipe:** R7 (strategy §2.5) | **Test level:** `//go:build integration` with concurrent writers
**Touches:** services/read/content/migrations, internal/repository/content.go, internal/service/content_service.go

## Problem
- `idx_contents_rss_dedup` (`content/migrations/000001:37-39`) is a **plain index, not UNIQUE**, so the check-then-insert in `CreateFromURL` / `CreateFromHTML` has no DB backstop; concurrent RSS deliveries produce genuine duplicate content rows.
- The dedup check in `BulkCreateFromHTML` runs **only** for `source_type=rss`. Email and manual/web content skip it entirely, there is no unique constraint, and delivery is two separate HTTP calls — so an outbox retry after a partial success creates a duplicate content row every time, and the same email appears multiple times in a user's list. The "at-least-once + dedupe" guarantee in `docs/ARCHITECTURE.md` holds for RSS in the common case and is absent for email/manual.

## What to do
1. Integration test: two concurrent inserts of the same content; assert exactly one row. Second test: deliver the same email content twice; assert one row.
2. Migration making `idx_contents_rss_dedup` UNIQUE. **Check for existing duplicate rows first** and write the dedup backfill in the same migration — it will fail on live data otherwise.
3. `ON CONFLICT DO NOTHING` at the insert site; extend the dedup path to email/manual content.
4. Write a real `down.sql` — do not leave a comment stub.

## Done when
- Concurrent-insert and repeat-delivery integration tests pass; migration applies cleanly against a DB seeded with pre-existing duplicates.


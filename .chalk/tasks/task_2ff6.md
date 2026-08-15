---
id: task_2ff6
title: [Dedup index] idx_contents_rss_dedup is not UNIQUE; email/manual content path has no dedup at all
type: task
status: in_progress
priority: 1
labels: [quality,wave2,database]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-09T06:46:24Z
updated_at: 2026-08-14T23:45:23Z
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

## Review

Re-verified on `main`: `idx_contents_rss_dedup` was still a plain index, and `BulkCreateFromHTML`
still only checked `GetByContentHashAndFeedID` for `source_type=="rss"` — both still exactly as
described.

**Migration** `000004_unique_content_dedup`: backfills pre-existing duplicates (RSS via
`(content_hash, source_feed_id)`, non-RSS via `(content_hash, original_url)`), keeping the
earliest-created row per group as survivor and reassigning `user_contents` from losers to the
survivor (skipping reassignments that would collide with a link the user already has for the
survivor — that duplicate link is simply dropped via `ON DELETE CASCADE`), then makes
`idx_contents_rss_dedup` UNIQUE and adds a new partial-unique `idx_contents_nonrss_dedup
(content_hash, original_url) WHERE source_type != 'rss'` for email/manual content, which has no
feed to key off. Verified by hand against seeded duplicate data (including the "user already
owns both the survivor and a loser" edge case) before wiring up the Go code. Real `down.sql`
included (index reversal only — the backfill itself is intentionally lossy, matching this repo's
existing convention for 000002/000003).

**Insert-site fix — one deliberate deviation from the task's "ON CONFLICT DO NOTHING" wording:**
`DO NOTHING` doesn't return the conflicting row via `RETURNING`, so on a race the caller would be
left holding a locally-generated ID that was never persisted — a phantom ID that then FK-violates
downstream (e.g. `user_contents.content_id`). Used `ON CONFLICT ... DO UPDATE SET updated_at =
contents.updated_at RETURNING <full row>` instead (a no-op update whose only purpose is to make
`RETURNING` fire on the conflict branch too), so `Create`/`CreateWithTx`/`BulkCreate` always end
up holding the true persisted row — new or pre-existing. `BulkCreate` groups by dedup arbiter
(RSS vs. non-RSS, since one `INSERT` can only declare one `ON CONFLICT` target) and collapses
same-key duplicates within a batch before inserting, since Postgres rejects a multi-row `ON
CONFLICT DO UPDATE` that proposes the same key twice in one statement.

Service layer (`CreateFromURL`/`CreateFromHTML`/`BulkCreateFromHTML`) extended: the existing
RSS pre-check now has a non-RSS sibling branch calling the new `GetByContentHashAndURL`. This
closes the gap for "manual" page submissions too (`user_content_handler.go`'s `CreateFromURL`
call with `source_type="manual"`), not just email, per the task title.

**Tests:** `internal/repository/content_dedup_integration_test.go` (10 concurrent goroutines,
RSS and email paths) and `internal/service/content_email_redelivery_integration_test.go`
(sequential repeat-delivery, mirroring an outbox retry) — both verified to fail against the
pre-fix code (stashed the fix, removed migration 000004, reran: all three failed with mismatched
IDs / duplicate rows, for the reason the finding describes) and pass after restoring the fix,
including under `-race` and across repeated runs. Updated every existing unit test/mock that
needed to account for the new `GetByContentHashAndURL` interface method and the new `Create`/
`BulkCreate` query shape. `gofmt`, `go vet`, `golangci-lint`, and `go test -race ./...` all clean.

Docs: updated `services/read/AGENTS.md`'s Deduplication Strategy section and the `contents`
table's index list — the old text explicitly said "not a DB-level UNIQUE constraint", which the
fix makes false.


---
id: task_b174
title: [P2-C7/H8] Recommender upsert conflicts on content-hash id, not just link → batch dropped every poll
type: task
status: closed
priority: 1
labels: [quality,wave2,database]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-09T06:46:24Z
updated_at: 2026-08-14T22:49:01Z
---
Read docs/QUALITY_REMEDIATION_STRATEGY.md §0 (rules of engagement) and §2.6 (definition of done) before starting. Read the full finding text in docs/CODE_QUALITY_REVIEW.md. One finding, one branch, one PR. Re-verify on main first — cited line numbers are from 2026-07-05 and drift.

**Finding:** P2-C7 / H8 | **Wave 2** | **Recipe:** R7 (strategy §2.5) | **Test level:** `//go:build integration` against real Postgres
**Touches:** services/explore/recommender/internal/db/article_repository.go

## Problem
`article_repository.go:77-127` upserts via `ON CONFLICT (link) DO UPDATE`, but `id` is a *separate* primary key set to `SHA256(cleaned_content)` (per `fetcher/migrations/000002`). When two different links carry the same content — syndicated/re-published articles, the exact case the content-hash id was designed to dedupe — the new row's `link` doesn't conflict but its `id` does, giving an **unhandled** PK violation. Because it is one multi-row `INSERT`, the whole batch fails and silently drops every other new article in that fetch cycle, and it **recurs on every subsequent poll** until the feed auto-disables.

An empty or duplicate `<link>` in a batch also trips "cannot affect row a second time". This is the mechanism behind Part 1's H8.

## What to do
1. Integration test: two articles with different links and the same content hash; assert the batch survives and the dedup behavior is deliberate. Also cover an empty/duplicate `link` in the same batch.
2. Fix: arm the `id` constraint (`ON CONFLICT (id) ...`) and decide explicitly what "same content, different link" should do — state the choice in the PR.

## Done when
- Integration test fails before the fix for the PK-violation reason and passes after; no article is silently dropped.

## Review

**Re-verified on main:** confirmed — `CreateBatch` (`article_repository.go:74-144`, lines drifted from the finding's 77-127) still built one multi-row `INSERT ... ON CONFLICT (link) DO UPDATE` with no handling for the separate `id` primary key.

**Failing tests first:** added two integration tests to `article_repository_integration_test.go` (`//go:build integration`, real Postgres via `internal/testutil`):
- `TestIntegration_CreateBatch_ContentHashCollision_BatchSurvives` — seeds one article, then delivers the same content under a new link alongside a genuinely new article in one `CreateBatch` call.
- `TestIntegration_CreateBatch_DuplicateAndEmptyLinksWithinBatch` — one batch with a same-link pair, two empty-link entries, and a fresh article.

Verified against a real local Postgres 16 (the sandbox has no Docker Hub access for testcontainers, so migrations were applied to a local `test_db` for this verification run only — no repo files were left changed) that both tests fail on pre-fix code for the finding's exact reasons: `duplicate key value violates unique constraint "articles_pkey"` and `ON CONFLICT DO UPDATE command cannot affect row a second time`.

**Fix — decision stated explicitly:** article ids are content hashes and links carry a separate UNIQUE constraint, so a single `INSERT ... ON CONFLICT` can only arbitrate one of the two (Postgres allows one conflict target per statement) — and the existing "same link, updated content" behavior (`TestIntegration_Create_DuplicateLink_UpdatesArticle` et al.) is a deliberate, tested feature that has to keep working. `CreateBatch` now pre-filters the batch before building the INSERT (new `dedupeForInsert`):
1. Drop articles with an empty link.
2. Drop batch entries whose id or link repeats earlier in the same batch (keep the first occurrence) — a single statement can't target the same `ON CONFLICT (link)` row twice.
3. Query which of the remaining ids already exist in the table, and drop those too.

The `ON CONFLICT (link) DO UPDATE` INSERT then runs on what's left, unchanged. **Choice:** when incoming content's hash already exists (in the DB or earlier in the batch) under a different link, the first-seen link stays canonical and the syndicated duplicate is dropped outright — no metadata overwrite. The batch's other articles are no longer dropped by the collision.

**Verification:**
- `gofmt -l .` clean, `go vet ./...` clean, `golangci-lint run ./recommender/...` clean
- `go test -race -count=1 ./recommender/...` — all green
- `go test -tags=integration -run TestIntegration ./recommender/internal/db/...` — all 6 tests green against real Postgres (3 pre-existing + 3 new), including the two new tests once the fix is in place
- No API/route boundary changed, so no `openapi.yaml`/`CLAUDE.md` route-table updates needed. `CreateBatch`'s doc comment was extended to explain the dedup decision.


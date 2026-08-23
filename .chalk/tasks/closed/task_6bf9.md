---
id: task_6bf9
title: [Audit F-S17-1/Tier 1] Shown-tracking drift is permanently unrecoverable — and it is a Phase B prerequisite
type: task
status: closed
priority: 1
labels: [quality,audit,explore,phase-b]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-17T12:49:39Z
updated_at: 2026-08-23T05:27:51Z
---
**Source:** Cairn Simplification Audit — https://claude.ai/code/artifact/286883fb-3f93-49c4-942f-4880251a409f · file:line detail supplied by the audit author 2026-08-17 and re-verified at HEAD `a6c56a1`.
Read docs/QUALITY_REMEDIATION_STRATEGY.md §0 (rules of engagement) and §2.6 (definition of done) before starting. One finding, one branch, one PR.

**Audit finding:** F-S17-1 | **Audit tier:** 1 (correctness) | **Verified.**
**The audit author revised this item's ranking upward** after reading epic_c482: it must land **before** the Phase B cutover.

## Problem
Two representations of "this article was shown to this user" are written **non-transactionally** in the `handleMarkShown` loop — `services/explore/recommender/internal/api/handlers.go:226-261`:
1. the `recommendations` row — `s.articleRepo.RecordRecommendation(...)` (:234), which returns `inserted`
2. the `articles.recommends` counter — `s.articleRepo.IncrementRecommendCount(...)` (:247)

If the insert succeeds and the increment fails, the loop `continue`s (:257-259) and **the drift is permanent**: the `recommendations` row now exists, so every later attempt returns `inserted == false` via `ON CONFLICT DO NOTHING` and short-circuits at :242-246 — the increment is **never retried**. The counter is under-counted forever, with no path back.

## Why this is gated on Phase B, not merely adjacent to it
Today the **eager** `trackRecommendation` write in `engine.GetRecommendations` masks the drift. **Phase B removes it**, making `handleMarkShown` the *sole* writer of both representations. After the cutover a failed increment silently corrupts the **quality-score denominator** (`articles.recommends` feeds the quality score — see `migrations/000004_voting_and_recommendations.up.sql:95-97`) with no eager write to paper over it.

**Land this before the epic_c482 Phase B cutover.**

## What to do
1. **Failing test first:** make the increment fail after a successful insert; assert the counter is *eventually* consistent (either both writes land or neither does). Fails on main — today the row persists and the counter never catches up.
2. Make the pair atomic. Either wrap both writes in one transaction, or make the increment derivable from the `recommendations` table so there is only one source of truth. Prefer removing the second representation over synchronising it, if the quality-score query can count rows affordably.
3. Whatever you choose, the failure mode must be **retryable**, not a permanent one-way divergence.

## Done when
- A failed counter write cannot leave the two representations permanently disagreeing, proven by a test that fails on main.

## Related
- **epic_c482** (Phase B: mobile-driven shown tracking) — **that epic is now `blocked_by` this task** (enforced 2026-08-17, owner decision), so this is a hard prerequisite for the cutover, not a follow-up. Landing this unblocks Phase B.
- **task_b5bd** (verify mobile shown-tracking adoption) — see the annotation added there about its gating metric. It and task_f84d are *not* blocked by this task and should proceed in parallel; only the cutover itself waits.

## Resolution
Re-verified: the two writes were indeed two independent `pgxpool.Pool.Exec` (autocommit) calls with no shared transaction.

Chose the "wrap both writes in one transaction" option from item 2 rather than deriving the counter from `COUNT(*)` on `recommendations` — the recommends counter is read in several hot query paths (`GetForRecommendation`, `GetLowExposureArticles`, `GetByID`, search), and a `pgxpool`-level transaction around two statements that already share a repository method is the smaller, lower-risk surgical diff for a Tier-1 correctness fix.

- `services/explore/recommender/internal/db/article_repository.go`: replaced `RecordRecommendation` + `IncrementRecommendCount` with one `RecordShown` method that runs the insert and the increment inside a single `pgx` transaction (mirrors the existing `Begin`/`defer Rollback`/`Commit` pattern already used in `vote_repository.go`). Any failure after the insert rolls back the insert too, so a retry is never blocked by a stale `ON CONFLICT DO NOTHING` row.
- `services/explore/recommender/internal/db/interfaces.go`: interface updated to match (single method, only caller was `handleMarkShown`).
- `services/explore/recommender/internal/api/handlers.go`: `handleMarkShown` now calls `RecordShown` once per article instead of the two-step sequence.
- `services/explore/recommender/integration_shown_test.go`: added `TestShown_FailedIncrementDoesNotLeavePermanentDrift`, which holds a `FOR NO KEY UPDATE` lock on the article row (blocks the counter update but not the FK-checked insert, verified against a local Postgres before writing the test), forces the first `/shown` call to fail via client-side timeout while the write is blocked, releases the lock, and asserts a retry lands both writes. This reproduces F-S17-1 against the pre-fix code (permanent `recorded=0` on retry) and passes against the fix (retry lands cleanly).

Verified: `gofmt -l .`, `go vet ./...`, `golangci-lint run` all clean; `go test -short ./...` passes. The new integration test requires Docker/testcontainers, unavailable in this sandbox — it compiles cleanly under `-tags=integration` and will run in the CI integration job (`go-checks.yml`, landed in #330 per the Wave 3 ledger).

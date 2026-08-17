---
id: task_02c8
title: [Audit/Tier 3] Vote counters: collapse the transition ladder AND fix the asymmetric RowsAffected handling
type: task
status: open
priority: 2
labels: [quality,audit,explore]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-17T12:50:50Z
updated_at: 2026-08-17T12:50:50Z
---
**Source:** Cairn Simplification Audit — https://claude.ai/code/artifact/286883fb-3f93-49c4-942f-4880251a409f · file:line detail supplied by the audit author 2026-08-17 and re-verified at HEAD `a6c56a1`.
Read docs/QUALITY_REMEDIATION_STRATEGY.md §0 (rules of engagement) and §2.6 (definition of done) before starting. One finding, one branch, one PR.

**Audit tier:** 3 (structural) — **but it carries a live correctness bug; treat that half as Tier 1.** Verified.

## Problem — two halves, one file
`services/explore/recommender/internal/db/vote_repository.go`

**(a) The transition ladder.** `RecordVote` (:57-141) hand-rolls the upvote↔downvote transitions with an `existingVoteType`/`oldVoteType` if-else ladder (:65-107), and `RemoveVote` (:184-197) repeats the shape.

**(b) The material half — identical condition, contradictory handling.** The same `result.RowsAffected() == 0` check is treated as fatal in one place and ignorable in the other:
```go
// :95-102  (RecordVote)
if result.RowsAffected() == 0 {
    slog.Warn("vote counter update had no effect (possibly already at 0)", ...)
    return fmt.Errorf("article %s: %w", articleID, apperrors.ErrArticleNotFound)
}

// :203-208  (RemoveVote)
if result.RowsAffected() == 0 {
    slog.Warn("vote counter update had no effect (possibly already at 0)", ...)
}   // ← no error
```
The `RecordVote` branch surfaces as **HTTP 404 for an article that exists** (`internal/api/handlers.go:315`) — the warning text even admits the real cause is "possibly already at 0", which is not a missing article. Same condition, same log line, opposite verdicts; at most one can be right.

## What to do
1. **Decide what `RowsAffected() == 0` means here** and apply it consistently. If it means "counter already at floor", neither site should return `ErrArticleNotFound`; article existence should be checked explicitly instead of inferred from an UPDATE's row count.
2. **Failing test first for (b):** vote on an article whose counter is already at 0 and assert the response is not 404. Fails on main.
3. Then collapse the transition ladder (a) — a small transition table or a single delta computation over (old, new) — once (b) has pinned down the semantics.

## Done when
- The two sites agree on what a zero row count means; no existing article can return 404 from a vote; the transition logic is expressed once.

## Sequencing
Do **(b) before (a)**. Collapsing the ladder first would bake the inconsistent semantics into the shared path.

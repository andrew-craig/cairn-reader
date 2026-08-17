---
id: epic_c482
title: Phase B: switch Explore shown tracking to mobile-driven
type: epic
status: open
priority: 2
labels: [explore,recommender,mobile]
blocked_by: [task_6bf9]
parent: null
remote_task_url: null
created_at: 2026-05-26T11:18:14Z
updated_at: 2026-08-17T21:31:20Z
---
Phase A (b568a25) added a mobile POST /api/v1/explore/shown endpoint and the client-side detection (scroll-gated, top-half-of-viewport) that calls it. To avoid double-counting during rollout, Phase A kept the eager write in engine.GetRecommendations and the new endpoint only records the dedup row.

Phase B completes the cutover: remove the eager trackRecommendation call so that articles.recommends and the recommendations table both reflect 'actually shown above the fold' rather than 'fetched'. This is the semantic shift the team accepted when approving Phase A — articles.recommends keeps its name but its meaning becomes more honest.

Critical sequence: do not remove the eager write until enough mobile clients are sending shown events, or users on old clients will see articles re-recommended on every refresh.

## Blocked by task_6bf9 (enforced 2026-08-17, owner decision)

The Cairn Simplification Audit (finding F-S17-1) found a second reason the cutover cannot go first, independent of client adoption: **the two shown-tracking writes are non-transactional and their drift is permanently unrecoverable.**

In `handleMarkShown` (`services/explore/recommender/internal/api/handlers.go:226-261`), `RecordRecommendation` inserts the dedup row and `IncrementRecommendCount` bumps `articles.recommends`. If the insert succeeds and the increment fails, the loop `continue`s — and because the dedup row now exists, every later attempt returns `inserted == false` via `ON CONFLICT DO NOTHING` and short-circuits. The increment is **never retried**.

Today the eager `trackRecommendation` write in `engine.GetRecommendations` masks this. **This epic removes that write**, making `handleMarkShown` the sole writer of both representations — so after the cutover a failed increment silently corrupts the quality-score denominator (`articles.recommends` feeds the quality score, `migrations/000004_voting_and_recommendations.up.sql:95-97`) with nothing to cover it.

`blocked_by: [task_6bf9]` is set on this epic to enforce that ordering. Note that chalk does not propagate a parent's block to its children, which is intended here: **task_b5bd and task_f84d remain actionable now** — they are the adoption prerequisites and should proceed in parallel with the drift fix.

Also read the metric-bias note added to task_b5bd: its shown:fetched gate is biased downward by the same `inserted`-gated counter, so don't read that gate naively when deciding the cutover is safe.

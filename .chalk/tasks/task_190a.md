---
id: task_190a
title: Remove eager trackRecommendation and move increment to handleMarkShown
type: task
status: open
priority: 2
labels: [explore,recommender]
blocked_by: [task_b5bd,task_b054]
parent: epic_c482
remote_task_url: null
created_at: 2026-05-26T11:18:42Z
updated_at: 2026-05-26T11:18:42Z
---
Code change that completes the cutover. Must land in a single commit so the system is never in a state where neither path increments articles.recommends.

Changes in services/explore/recommender:

1. internal/recommend/engine.go: delete the two trackRecommendation call sites — the loop at lines 55-59 (the <=5 articles branch) and the loop at lines 102-112 (the main path). GetRecommendations becomes a pure read; it no longer writes to recommendations or to articles.recommends.

2. internal/api/handlers.go: inside handleMarkShown, after the existing RecordRecommendation call succeeds for a given article ID, also call s.articleRepo.IncrementRecommendCount. Tolerate apperrors.ErrArticleNotFound (log warn, skip) since unknown IDs in the batch should not fail the request. The Phase A comment block in handleMarkShown explaining the deferred increment can be removed.

3. The trackRecommendation helper at engine.go:186-199 becomes dead code — delete it along with any now-unused imports.

Acceptance:
- The integration tests added in task_b054 pass.
- go build ./recommender/... and go vet ./recommender/... clean.
- Existing unit/integration tests still pass.

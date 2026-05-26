---
id: task_b054
title: Add integration tests for mobile-driven shown semantics
type: task
status: in_progress
priority: 2
labels: [explore,recommender,test]
blocked_by: []
parent: epic_c482
remote_task_url: null
created_at: 2026-05-26T11:18:32Z
updated_at: 2026-05-26T11:54:11Z
---
Add integration tests under services/explore/recommender that capture the post-Phase-B contract. Tests should run against a real Postgres (testcontainers, following the pattern already used in services/explore/recommender/internal/db).

Required cases:

1. Two consecutive GET /api/v1/explore/recommendation calls for the same user, with NO POST /api/v1/explore/shown between them, MUST return overlapping article IDs (proves the eager trackRecommendation has been removed).
2. GET /recommendation, then POST /shown with a subset of the returned IDs, then GET /recommendation again: the IDs that were marked shown MUST NOT appear in the second response; IDs that were returned but not marked shown MAY appear again.
3. POST /shown increments articles.recommends by 1 per article ID (idempotent — calling shown twice for the same (user, article) does not double-increment).
4. POST /shown with a mix of valid and unknown article IDs: response reports the recorded count, unknown IDs are skipped (logged warn), 200 status.

Write these tests first so Phase B's code change has a clear pass/fail signal. They will fail against current main until the next task lands.

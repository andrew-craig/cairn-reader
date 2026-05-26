---
id: epic_c482
title: Phase B: switch Explore shown tracking to mobile-driven
type: epic
status: open
priority: 2
labels: [explore,recommender,mobile]
blocked_by: []
parent: null
remote_task_url: null
created_at: 2026-05-26T11:18:14Z
updated_at: 2026-05-26T11:18:14Z
---
Phase A (b568a25) added a mobile POST /api/v1/explore/shown endpoint and the client-side detection (scroll-gated, top-half-of-viewport) that calls it. To avoid double-counting during rollout, Phase A kept the eager write in engine.GetRecommendations and the new endpoint only records the dedup row.

Phase B completes the cutover: remove the eager trackRecommendation call so that articles.recommends and the recommendations table both reflect 'actually shown above the fold' rather than 'fetched'. This is the semantic shift the team accepted when approving Phase A — articles.recommends keeps its name but its meaning becomes more honest.

Critical sequence: do not remove the eager write until enough mobile clients are sending shown events, or users on old clients will see articles re-recommended on every refresh.

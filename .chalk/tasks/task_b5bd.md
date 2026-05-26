---
id: task_b5bd
title: Verify mobile shown-tracking adoption before Phase B cutover
type: task
status: open
priority: 2
labels: [explore,recommender]
blocked_by: []
parent: epic_c482
remote_task_url: null
created_at: 2026-05-26T11:18:23Z
updated_at: 2026-05-26T11:18:23Z
---
Gating check before any Phase B code change. The new POST /api/v1/explore/shown endpoint logs 'recorded shown articles' on every batch (see handleMarkShown in services/explore/recommender/internal/api/handlers.go). Use those logs to confirm:

1. Shown events are arriving from real users (not just dev devices).
2. The volume is in a plausible range vs. GET /api/v1/explore/recommendation calls — a healthy ratio of shown:fetched would be < 1.0 (since not every fetched article gets scrolled past the mid-point), but not near zero.
3. The mobile client version reporting shown events represents a high enough share of MAU that we can accept losing dedup for the remaining slice when the eager write is removed.

If adoption is too low, hold Phase B until the relevant mobile release rolls out further.

No code changes in this task — this is purely an observability/data check. Close once the threshold is met and link the next task.

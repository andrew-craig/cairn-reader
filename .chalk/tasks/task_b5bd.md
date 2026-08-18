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
updated_at: 2026-07-01T22:29:46Z
---
Gating check before any Phase B code change. The new POST /api/v1/explore/shown endpoint logs 'recorded shown articles' on every batch (see handleMarkShown in services/explore/recommender/internal/api/handlers.go). Use those logs to confirm:

1. Shown events are arriving from real users (not just dev devices).
2. The volume is in a plausible range vs. GET /api/v1/explore/recommendation calls — a healthy ratio of shown:fetched would be < 1.0 (since not every fetched article gets scrolled past the mid-point), but not near zero.
3. The mobile client version reporting shown events represents a high enough share of MAU that we can accept losing dedup for the remaining slice when the eager write is removed.

If adoption is too low, hold Phase B until the relevant mobile release rolls out further.

No code changes in this task — this is purely an observability/data check. Close once the threshold is met and link the next task.

STATUS (2026-07-01): Could not complete this verification from this session — it runs in an ephemeral dev container with no production log access, and there's no log aggregation stack yet (see feature_1a69). Whoever has prod access should run the checks below.

Also found: criterion 3 is not currently measurable even with log access — no app/client version is logged anywhere on the /shown path today. Filed task_f84d to add that; consider criteria 1 & 2 alone as the gate until it lands, or wait on task_f84d first.

Suggested commands against production logs (adjust to your log shipping setup, e.g. `docker compose logs` or your aggregator's query syntax):

1. Real users vs. dev devices — check distinct user_id count and request cadence:
   grep '"recorded shown articles"' <log source> | jq -r .user_id | sort -u | wc -l
   (compare against known dev/test user IDs; look for organic-looking timing, not synchronized bursts from one IP)

2. Shown:fetched ratio — count POST /shown vs GET /recommendation over the same window:
   grep '"path":"/api/v1/explore/shown"' <log source> | wc -l
   grep '"path":"/api/v1/explore/recommendation"' <log source> | wc -l
   ratio = shown_count / recommendation_count, expect somewhere under 1.0 but not near zero.

3. Client version share — blocked on task_f84d (no version data exists yet).

---

## ⚠️ Metric bias found by the Cairn Simplification Audit (2026-08-17)

**Audit report:** https://claude.ai/code/artifact/286883fb-3f93-49c4-942f-4880251a409f

**Metric 2's shown:fetched ratio is biased downward, against a gate whose failure condition is "near zero".** Read this before drawing a conclusion from it.

The `recorded` counter in `handleMarkShown` (`services/explore/recommender/internal/api/handlers.go:226-261`) increments **only when `inserted == true` and both writes succeed** (`recorded++` at :260, gated by the `continue` at :242-246 for `!inserted`). `RecordRecommendation` uses `ON CONFLICT DO NOTHING`, so `inserted` is false for any article the user has already been shown.

So the numerator counts **first-time shows only** — repeat views are silently dropped — while the denominator (`GET /recommendation`) counts **every fetch, including re-fetches of already-shown articles**. The more the client re-requests, the lower the ratio, independent of adoption.

**Consequences for this task:**
- A genuinely healthy client can read as "near zero" if users re-fetch a lot. Do not fail the gate on the raw ratio alone.
- Either count distinct articles on both sides over the window, or read the numerator from the `recommendations` table (rows inserted in the window) rather than from request counts.

**Related:** **task_6bf9** (audit F-S17-1) — the same loop's two writes are non-transactional and drift permanently, and the audit revised that finding **upward to a Phase B prerequisite**: today the eager `trackRecommendation` write in `engine.GetRecommendations` masks the drift, and Phase B removes it. Land task_6bf9 before the epic_c482 cutover.

---
id: task_f84d
title: Log mobile app version on /shown requests
type: task
status: open
priority: 2
labels: []
blocked_by: []
parent: epic_c482
remote_task_url: null
created_at: 2026-07-01T22:29:16Z
updated_at: 2026-07-01T22:29:16Z
---

Surfaced while working task_b5bd: criterion 3 of that gating check ("mobile client version reporting shown events represents a high enough share of MAU") cannot be answered today because no app/client version is captured anywhere on the /shown path.

Verified (2026-07-01):
- `handleMarkShown` (services/explore/recommender/internal/api/handlers.go:263) logs only `user_id`, `requested`, `recorded`.
- The shared chi request-logging middleware (pkg/logging/chi_middleware.go:27-32) logs `request_id`, `method`, `path`, `client_ip` — no `User-Agent` or version header.
- The mobile client's shown-reporting call (apps/mobile/src/services/explore.ts:151-167) does not send an app-version header today.

WHAT TO DO:
- Add an app-version header (e.g. `X-App-Version`) to mobile's POST /api/v1/explore/shown call.
- Log it as a field on the "recorded shown articles" log line in handleMarkShown.
- Once deployed and rolled out, task_b5bd's criterion 3 becomes answerable from logs.

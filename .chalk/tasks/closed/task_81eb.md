---
id: task_81eb
title: Web: Converge apps/web onto @cairn/shared config/api (storage adapter)
type: task
status: closed
priority: 2
labels: []
blocked_by: []
parent: epic_f54e
remote_task_url: null
created_at: 2026-06-24T08:38:09Z
updated_at: 2026-06-28T22:55:21Z
---


## Context
Follow-up from task_5860 (mobile→@cairn/shared migration). The shared package now
exposes the storage-adapter server-URL config (apps/shared/src/config/api.ts) and
apps/mobile consumes it. apps/web still has its own apps/web/src/config/api.ts.

Web's copy carries extra behavior the shared/mobile-baseline version intentionally
omits:
- setServerUrl clears SESSION_STORAGE_KEYS on a real server change.
- getDefaultServerUrl resolves from VITE_API_URL / window.location.origin.

To converge: decide whether the session-clearing-on-server-switch behavior should
become shared (and thus apply to mobile too) or stay web-only via a wrapper, then
delete apps/web/src/config/api.ts in favor of @cairn/shared (web injects its
localStorage adapter + origin-based default at startup, as mobile does).

VERIFICATION: apps/web has no local config/api.ts duplicating shared; web
tsc+build+test pass; auth/session behavior on server switch unchanged for web.

---
id: chore_cc63
title: Remove dead code and stale comments found in docs review
type: chore
status: open
priority: 4
labels: []
blocked_by: []
parent: null
remote_task_url: null
created_at: 2026-10-14T20:28:44Z
updated_at: 2026-10-14T20:28:44Z
---
Found in chore_65f1.
- apps/mobile/src/services/read.ts:440-470 listFeedSubscriptions: deprecated, no callers, and it calls an internal-key-only route.
- Fetcher EnableFeed handler: unused, unless the re-enable feature uses it.
- pkg/middleware StrictCORSConfig: no callers. RateLimitRedis: no callers.
- services/users auth_handler.go ErrorResponse: dead.
- pkg/models: no Go importers (only a replace line in cmd/selfhost/go.mod).
- Stale comments: apps/web/src/services/read.ts header ('later tasks extend with search/add-link'), apps/mobile articleMutations.ts Explore mention, and content handler comments citing old paths (e.g. POST /api/v1/contents).

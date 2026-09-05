---
id: task_d413
title: Fix: selfhost email internal routes 404 (content catch-all swallows them)
type: task
status: in_progress
priority: 2
labels: [quality,selfhost,bugfix]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-09-05T09:21:04Z
updated_at: 2026-09-05T09:21:04Z
---
**Source:** recovered from a stashed WIP change (`git stash` entry) found during a repo cleanup pass and re-verified against current `main` before implementing.

## Problem
In the selfhost single-binary build, `services/read/email/selfhost/email.go`'s `MountEmail` only mounted `/api/v1/source/email` and `/api/v1/source/email/*` on the shared top-level router. It never mounted `/api/v1/internal/source/email*`, even though `services/read/email/internal/api/router.go` defines that internal route (`GET /api/v1/internal/source/email/user/{user_id}/senders`, internal-API-key protected).

Meanwhile `services/read/content/selfhost/content.go` mounts a catch-all at `/api/v1/internal` and `/api/v1/internal/*` for the content service's own internal routes. With email's internal route never registered on the shared router, requests to it fell through to content's catch-all and returned 404 — breaking the internal sender-listing endpoint that the content service's subscription aggregator depends on, in selfhost deployments only (per-service deployments mount each service's full router independently, so they weren't affected).

## What was done
Branch `fix/selfhost-email-internal-routes-404`, commit `d45db9d`: added
```go
r.Handle("/api/v1/internal/source/email", emailRouter)
r.Handle("/api/v1/internal/source/email/*", emailRouter)
```
alongside the existing public mount in `MountEmail`. Verified `cmd/selfhost` still builds (`go build ./...`).

## Done when
- [x] Fix implemented and committed on a branch
- [ ] PR opened and merged
- [ ] Manually verified against a running selfhost stack (docker compose) that `GET /api/v1/internal/source/email/user/{id}/senders` with a valid `X-Internal-API-Key` no longer 404s

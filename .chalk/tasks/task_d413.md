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
alongside the existing public mount in `MountEmail`. Verified `cmd/selfhost` still builds (`go build ./...`) and that `go test ./...` passes in `cmd/selfhost`, `services/read`, and `services/read/email`. PR #375 opened against `main`.

## Follow-up review findings (2026-09-05) — THIS BRANCH IS INCOMPLETE, DO NOT MERGE AS-IS

Raised after review pushback that selfhost email is working in practice. Re-verified by
reproducing the composed `cmd/selfhost` mount structure in a throwaway chi test:

```
WITHOUT FIX (main)  => 404 "404 page not found"
WITH FIX (branch)   => 403 {"error":"HTTPS required"}
```

### 1. The 404 is real, but the blast radius is much smaller than described above
Email *ingestion* never touches this route. The only consumer is
`SubscriptionAggregatorHandler`, which swallows the failure
(`services/read/content/internal/api/handlers/subscription_aggregator_handler.go:81-88`):
`slog.Error("Failed to fetch email subscriptions", ...)` and carries on. The unified
subscriptions endpoint still returns 200 — it just silently omits newsletter senders.
That is why selfhost email appears to work. The wording under "Problem" above
("breaking the internal sender-listing endpoint") overstates the user-visible impact:
the symptom is a silent omission plus a log line, not a failed request.

### 2. The mount fix alone does NOT fix the endpoint — it turns 404 into 403
The internal route is wrapped in `sharedmw.RequireHTTPS`
(`services/read/email/internal/api/router.go:67`), which 403s unless `r.TLS != nil` or
`X-Forwarded-Proto: https` is set (`pkg/middleware/security.go:12-30`). In selfhost the
content service calls itself at `http://localhost:PORT`
(`cmd/selfhost/adapt_content.go:28`, `EmailIngestServiceURL: selfhostBaseURL`), and
`services/read/content/internal/service/email_ingest_client.go:50-57` sets only the
internal API key and request-ID headers. Plain HTTP, no forwarded-proto → 403.

Green CI proved nothing here: no existing test exercises the *composed* selfhost router,
so `go build` + `go test` passing did not validate the fix. The unchecked
"manually verified against a running selfhost stack" box below is what would have caught it.

### 3. Design constraint: the loopback HTTP call is deliberate — do not remove it
An in-process call from content → email in the selfhost binary would sidestep both the
routing collision and the HTTPS check, but it is explicitly **not** the direction to take.
The loopback HTTP call exists to minimise code divergence between the selfhost build and
the hosted stack, so both exercise the same service-to-service path. That is a deliberate
principles-level choice made to incentivise parallel development of the two deployment
modes; diverging them to fix this bug would trade a working shared code path for local
convenience. Any fix must keep the loopback HTTP call intact and address the
`RequireHTTPS` check instead.

## Done when
- [x] Fix implemented and committed on a branch
- [x] PR opened (#375) — **not to be merged as-is; see findings above**
- [ ] Decide how selfhost satisfies `RequireHTTPS` on internal loopback calls, keeping the
      loopback intact (e.g. internal client sets `X-Forwarded-Proto`, or the middleware
      recognises a trusted-loopback condition) — needs a decision, not yet made
- [ ] Add a regression test over the composed `cmd/selfhost` router asserting
      `GET /api/v1/internal/source/email/user/{id}/senders` returns 200, not 404/403
- [ ] Manually verified against a running selfhost stack (docker compose) that
      `GET /api/v1/internal/source/email/user/{id}/senders` with a valid
      `X-Internal-API-Key` returns 200

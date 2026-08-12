---
id: task_204b
title: [P2-IDOR] read/fetcher has zero inbound auth; user_id read from URL path → cross-tenant access
type: task
status: closed
priority: 0
labels: [quality,security,wave1,auth]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-09T06:43:57Z
updated_at: 2026-08-12T09:01:40Z
---
Read docs/QUALITY_REMEDIATION_STRATEGY.md §0 (rules of engagement) and §2.6 (definition of done) before starting. Read the full finding text in docs/CODE_QUALITY_REVIEW.md. One finding, one branch, one PR. Re-verify on main first — cited line numbers are from 2026-07-05 and drift.

**Finding:** P2 tenancy/IDOR matrix | **Wave 1** | **Recipe:** R1 (strategy §2.5) | **Test level:** httptest against the real router constructor
**Touches:** services/read/fetcher/internal/api/router.go, subscription handlers, its openapi.yaml + CLAUDE.md

## Problem
`read/fetcher` (ingest-rss) has **zero inbound authentication** — `internal/api/router.go:73-86`, only `RequireHTTPS`. `user_id` is taken straight from the URL path and never checked against a token. Anything that can reach `ingest-rss:8085` — a sibling container, or an SSRF pivot from C1 — can read, add, or delete **any user's** RSS subscriptions. This is the standout security finding of the Part 2 tenancy matrix. Also unauthenticated: `PATCH /feed/{feed_id}`, which disables a feed for all subscribers.

Context from the tenancy matrix: the rest of the JWT-authenticated surface is uniformly solid — every `RequireAuth` endpoint derives `user_id` from the validated token and backs it with a `WHERE user_id = $jwt` clause. The risk is entire services with no inbound auth, relying only on network topology.

## What to do
1. Failing test first: httptest against the real router, no credentials, on a user-scoped path.
2. Decide the boundary: JWT (`RequireAuth`) if callers have a user context; otherwise `RequireInternalAPIKey` **plus** the resource-owning service re-checking ownership. Never trust `user_id` from the path alone. State the choice in the PR.
3. Apply at the route group; update all callers, `openapi.yaml`, CLAUDE.md.
4. Add the router-inventory allowlist test (R1 step 5).

## Done when
- A cross-tenant request carrying another user's id in the path is rejected, proven by test.


## Review

Re-verified on current `main`: `read/fetcher`'s `/api/v1/source/rss/*` route group still had
only `RequireHTTPS`, no auth — confirmed via a new httptest against the real router asserting
401 with no credentials (failed before the fix, as expected).

**Boundary decision:** `RequireInternalAPIKey`, not JWT. This service has no JWT/Vault
infrastructure of its own and is documented as "internal-facing, accessed through Content Service
gateway" — Content Service already validates the caller's JWT and matches `user_id` before
proxying down. The fetcher's own queries were already correctly scoped by `user_id` from the
path; the gap was that nothing enforced *who* could set that path in the first place.

**Changes:**
- `fetcher/internal/api/router.go`: `RequireInternalAPIKey` applied to the whole
  `/api/v1/source/rss` route group (subscribe/list/unsubscribe/update-feed).
- `fetcher/internal/config/config.go` + `cmd/ingest_rss/main.go`: added required `INTERNAL_API_KEY`
  config, mirroring content service's pattern (fails startup if unset).
- `fetcher/selfhost/rss.go` + `cmd/selfhost/adapt_ingest_rss.go` + `cmd/selfhost/main.go`: threaded
  the shared `internalAuthMiddleware` into the self-host RSS mount.
- `content/internal/service/ingest_rss_client.go` + `content/internal/api/router.go`: the
  content-service client (the one legitimate caller) now sends `X-Internal-API-Key` on every
  call to fetcher.
- `infrastructure/docker/dev/docker-compose.yml`: added `INTERNAL_API_KEY` to the `ingest-rss`
  service env (the worker already had it for its own outbound calls; the API server didn't).
- `fetcher/api/openapi.yaml`, `services/read/AGENTS.md` (CLAUDE.md symlink target),
  `docs/ARCHITECTURE.md`: documented the new auth requirement.
- New test: `fetcher/internal/api/router_auth_test.go` —
  `TestUnauthenticatedRoutesRejected` (including an explicit cross-tenant-path case) and
  `TestRouterInventory_UnlistedRoutesRequireAuth` (the ratchet against future unauthenticated
  routes), mirroring content service's existing `router_auth_test.go`.

**Verified:** `go build`, `go vet`, `golangci-lint run` (0 issues), `go test ./...` all green in
`services/read` and `cmd/selfhost`; `gofmt -l` clean.

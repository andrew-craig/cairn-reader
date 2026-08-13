---
id: task_9df8
title: [Explore auth] explore article-inject + fetch/sync trigger endpoints unauthenticated
type: task
status: closed
priority: 1
labels: [quality,security,wave1,auth]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-09T06:43:57Z
updated_at: 2026-08-13T21:33:10Z
---
Read docs/QUALITY_REMEDIATION_STRATEGY.md §0 (rules of engagement) and §2.6 (definition of done) before starting. Read the full finding text in docs/CODE_QUALITY_REVIEW.md. One finding, one branch, one PR. Re-verify on main first — cited line numbers are from 2026-07-05 and drift.

**Finding:** Theme 1 (explore) | **Wave 1** | **Recipe:** R1 (strategy §2.5) | **Test level:** httptest against the real router constructor
**Touches:** services/explore/recommender router, services/explore/fetcher router, both openapi.yaml files

## Problem
`POST /api/v1/explore/article` and the fetcher's `/fetch` / `/sync` trigger endpoints have no service-to-service auth. Anyone who can reach them can inject articles into the shared explore catalog or hammer the fetch triggers. Same root cause as C1/C2 and the read/fetcher IDOR: "internal" is a comment, not an enforced boundary.

## What to do
1. Failing test first: httptest against the real router constructors, no credentials.
2. Apply `RequireInternalAPIKey` at the route group. Reference: `/api/v1/internal/*` in read/content's router.
3. Update the internal callers to send the key; update both `openapi.yaml` files and the service CLAUDE.md route tables.
4. Add the router-inventory allowlist test (R1 step 5) to both routers.

## Done when
- Both routers reject uncredentialed requests to the trigger/inject endpoints, proven by test.

## Review

Re-verified on `main`: `POST /api/v1/explore/article` (recommender) and the fetcher's
`/fetch`, `/sync`, `/stats` triggers had no auth middleware at all — confirmed still true.

**Fix:**
- Recommender: `internal/api/server.go` now wraps `POST /article` with
  `internalAuthMiddleware.RequireInternalAPIKey` (new constructor param, wired from
  `cfg.InternalAPIKey` in `main.go` and the selfhost `Mount`).
- Fetcher: the router was inline in `main.go` with no separate constructor to test against
  httptest, so it was extracted into `internal/api/router.go` (`NewRouter`, mirroring
  `services/read/fetcher/internal/api/router.go`'s pattern) and the whole
  `/api/v1/explore/feed` group now requires `RequireInternalAPIKey` — `/stats` wasn't named
  in the finding but sits in the same "internal trigger" route group, so it got the same
  boundary rather than leaving one hole in an otherwise-protected group.
- Both services gained a required `INTERNAL_API_KEY` config var (same pattern as
  `services/read/fetcher`), and the fetcher's `RecommenderClient` now sends
  `X-Internal-API-Key` when calling the recommender's `/article` endpoint. Selfhost wiring
  (`cmd/selfhost/adapt_explore_*.go`) threads the shared `internalAuthMiddleware` through.
- docker-compose (dev + prod) now pass `INTERNAL_API_KEY` to both services.
- Docs updated: `services/explore/api/openapi.yaml` (new `internalApiKey` security scheme,
  applied to the 3 endpoints), `services/explore/AGENTS.md` (CLAUDE.md route tables/examples),
  `services/explore/README.md`, `docs/ARCHITECTURE.md` (endpoint tables no longer say "no auth").

**Tests (failing-first, then passing):**
- `services/explore/recommender/internal/api/router_auth_test.go` — new. Proves
  `POST /article` rejects uncredentialed requests (401) and adds the router-inventory
  allowlist ratchet (`chi.Walk` over every route).
- `services/explore/fetcher/internal/api/router_auth_test.go` — new, same pattern for
  `/fetch`, `/stats`, `/sync` plus the router-inventory ratchet.
- Existing integration tests (`-tags=integration`) updated to send the internal key where
  they call these endpoints directly (`recommender/integration_test.go`,
  `integration_shown_test.go`, `fetcher/integration_test.go`).

**Verification run:** `gofmt -l .`, `go vet ./...`, `golangci-lint run ./...`,
`go test ./...` all clean in both `services/explore` and `cmd/selfhost` modules.
`go vet -tags=integration ./...` still reports 3 pre-existing failures unrelated to this
change (`UpdateFetchResult`/`GetRecommendations` signature drift in fetcher/recommender
integration tests, predating this branch — confirmed via `git stash`) — left untouched per
"surgical changes."


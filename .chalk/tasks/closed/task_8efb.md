---
id: task_8efb
title: [H10] Recovery middleware registered before request-ID middleware in all 6 routers → request_id=unknown
type: task
status: closed
priority: 2
labels: [quality,wave3,observability]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-09T06:53:56Z
updated_at: 2026-09-23T07:59:40Z
---
Read docs/QUALITY_REMEDIATION_STRATEGY.md §0 (rules of engagement) and §2.6 (definition of done) before starting. Read the full finding text in docs/CODE_QUALITY_REVIEW.md. One finding, one branch, one PR. Re-verify on main first — cited line numbers are from 2026-07-05 and drift.

**Finding:** H10 (+ Part 2 broadening) | **Wave 3** | **Recipe:** R6 (strategy §2.5) | **Test level:** panicking test handler; assert the panic log carries the real request ID
**Touches:** all 6 service routers, pkg/middleware/recovery.go, error-logging call sites across services

## Problem
`Recovery` is registered **before** `ChiRequestLogger` in all six routers, so `pkg/middleware/recovery.go:24` never sees the request ID and every panic log is stamped `request_id=unknown`, defeating incident correlation. Verified in all 6 routers.

Part 2 broadened this: the per-request logger (`logging.FromContext`) is actually consumed in **exactly one handler in the whole repo** — users' `RefreshToken`. Every other error log across all services uses the global logger with no `request_id`. This is the largest correlation gap in the codebase, bigger than the panic path alone. The email service never populates the logging context at all — it uses chi's own middleware.

## What to do
1. Test first: a panicking test handler through the real router; assert the panic log line carries the real request ID. Fails on main.
2. Reorder so request-ID/logger middleware **wraps** Recovery, in all 6 routers.
3. Populate the logging context in the email service too.
4. Repoint error-logging call sites at the per-request logger. Scope this deliberately — if repointing every call site is too large for one PR, do the routers plus one service here and file a follow-up task for the rest, saying so in the PR.

## Done when
- Panic logs carry a real request ID in all 6 routers, proven by test.


## Plan (2026-09-23)
Re-verified on `main` at `c245c61`: all 6 routers still `Use(sharedmw.Recovery)` before `logging.ChiRequestLogger` (email uses chi's own `RequestID`/`Logger`, so `logging.GetRequestIDFromContext` is always empty there).

- [x] Test first: `pkg/middleware/middlewaretest.AssertPanicLogCarriesRequestID(t, mux)` mounts a panicking route on the real router, sends `X-Request-ID`, and asserts the `panic` log line carries it. One test per router (6). → verify: all 6 fail on main with `request_id=unknown`.
- [x] Reorder so `ChiRequestLogger` wraps `Recovery` in all 6 routers. → verify: the 6 tests pass.
- [x] Email: replace `chimw.RequestID` + `chimw.Logger` with `logging.ChiRequestLogger` (after `RealIP`, so the logged client IP stays correct). → verify: email test passes; `go vet`.
- [x] Step 4, scoped to **users**: repoint request-path `slog.*` calls (handlers, `services/`, `auth/refresh_token.go`) to `logging.FromContext(ctx)`. Background loops (`auth/vault.go`) stay on the global logger — no request. → verify: users unit tests pass.
- [x] File a follow-up task for step 4 in explore, read/content, read/fetcher, read/email.
- [x] `go build ./... && go vet ./... && go test ./...` in every touched module; tick the ledger row in `docs/QUALITY_REMEDIATION_STRATEGY.md`.

## Review (2026-09-23, branch `task_8efb`)
- **Test first.** `pkg/middleware/middlewaretest.AssertPanicLogCarriesRequestID` mounts a panicking route on the real `*chi.Mux`, sends `X-Request-ID`, and asserts the `panic` log line carries it. One `TestRouter_PanicLogCarriesRequestID` per router: `cmd/selfhost` (master router), explore fetcher + recommender, read content + fetcher, read/email, users. All 7 failed on `main` with `request_id = "unknown"`; all 7 pass after the fix.
- **Fix.** `logging.ChiRequestLogger` now wraps `sharedmw.Recovery` in every router. Side benefit: a recovered panic now also produces the `http request completed` line (status 500), which it previously skipped because Recovery sat outside the logger.
- **Email.** `chimw.RequestID` + `chimw.Logger` replaced by `logging.ChiRequestLogger(slog.Default())`, after `chimw.RealIP` so the logged client IP stays the real one. Nothing in email read chi's request-ID key.
- **Step 4, scoped to users.** Request-path `slog.*` calls in `handlers/auth_handler.go`, `services/auth_service.go`, `services/user_service.go` and `auth/refresh_token.go` now go through `logging.FromContext(ctx)`. `auth/vault.go` background loops stay global. `TestRouter_HandlerErrorLogCarriesRequestID` drives a malformed `/auth/refresh` through the real router and asserts the error line carries the request ID; it fails without the repoint. Remaining services filed as **task_603e**.
- **Checks.** `go build`, `go vet`, `go test ./...` clean in pkg/middleware, cmd/selfhost, services/explore, services/read, services/read/email, services/users. `golangci-lint` 0 issues in all linted modules (pkg/middleware's 6 errcheck hits are pre-existing and not CI-linted).
- **Noticed, not changed.** In self-host, each service router is mounted under the master router and both run `ChiRequestLogger`. The master sets `X-Request-ID` on the *response* only, so the inner router mints a second ID: one request, two IDs, two access-log lines. Pre-existing; worth a look when task_603e touches logging.

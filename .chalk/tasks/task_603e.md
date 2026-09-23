---
id: task_603e
title: [H10 follow-up] Repoint request-path error logs at the per-request logger in explore, read/content, read/fetcher, read/email
type: task
status: open
priority: 3
labels: [quality,wave3,ops]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-09-23T07:59:24Z
updated_at: 2026-09-23T07:59:24Z
---
Follow-up to task_8efb, which fixed the middleware order in all routers (panic logs now carry the real request ID) and did step 4 for **services/users** only, as that task allowed.

## Problem
Request-path logs in the remaining services still use the global `slog` logger, so they carry no `request_id` and can't be tied to the access-log line (`http request completed`) or the panic log for the same request. Handler/service-layer `slog.*` call counts at `c245c61`: explore/recommender/internal/api 29, read/content/internal/api 20, explore/fetcher/internal/api 5, read/fetcher/internal/api 5, plus service/repository layers under each that run with a request `ctx`.

## What to do
Follow the users pattern from task_8efb: replace `slog.X(` with `logging.FromContext(ctx).X(` (or `r.Context()` in handlers) wherever the call runs on a request. Leave background loops (fetch/sync goroutines, workers, Vault renewal) on the global logger: they have no request. One service per PR is fine.

## Done when
- A router-level test per service (see `services/users/internal/handlers/router_test.go` `TestRouter_HandlerErrorLogCarriesRequestID`) shows a handler error log carrying the request's `X-Request-ID`.

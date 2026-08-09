---
id: task_8efb
title: [H10] Recovery middleware registered before request-ID middleware in all 6 routers → request_id=unknown
type: task
status: open
priority: 2
labels: [quality,wave3,observability]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-09T06:53:56Z
updated_at: 2026-08-09T06:53:56Z
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


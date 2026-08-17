---
id: task_dd2d
title: [H3+H4] Password-reset tokens never delivered; self-host build panics on reset/verify routes
type: task
status: in_progress
priority: 2
labels: [quality,wave2,ops]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-09T06:46:25Z
updated_at: 2026-08-16T12:43:42Z
---
Read docs/QUALITY_REMEDIATION_STRATEGY.md §0 (rules of engagement) and §2.6 (definition of done) before starting. Read the full finding text in docs/CODE_QUALITY_REVIEW.md. One finding, one branch, one PR. Re-verify on main first — cited line numbers are from 2026-07-05 and drift.

**Finding:** H3 + H4 | **Wave 2** | **Recipe:** R6 (strategy §2.5) | **Test level:** route-level test against the self-host wiring
**Touches:** services/users/internal/services/auth_service.go, cmd/selfhost/users.go, services/users/internal/api/router.go

## Problem
- **H3:** password-reset tokens are generated and stored but **never delivered** to the user (`auth_service.go:527-558`) — no mailer, no log, no return value. The feature is functionally broken despite being marked done.
- **H4:** self-host wiring omits `PasswordResetTokenRepo` and `EmailVerificationService` (`selfhost/users.go:104-134`) while the router always registers those routes (`router.go:97-102`) → nil-interface panic / 500 on `/forgot-password`, `/reset-password` and `/verify-email` in the self-host build.

## What to do
This needs a decision before code: either **wire the feature up properly** (delivery mechanism + self-host dependencies) or **explicitly remove the routes** from the self-host build so they cannot be called. Write both options into this task with a recommendation and confirm before implementing — do not silently pick one.

Then:
1. Test first: hit `/forgot-password` and `/verify-email` against the self-host wiring; assert no panic and a coherent response.
2. Implement the chosen option; update `openapi.yaml` and the users CLAUDE.md route table if routes are removed.

## Done when
- The self-host build cannot panic on these routes, and the reset flow is either functional end-to-end or absent by design (documented).


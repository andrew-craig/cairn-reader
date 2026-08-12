---
id: task_e317
title: [P2-C2/H1/H2/H14] Credential material in logs: verification URL at INFO, token prefixes, mobile token logging
type: task
status: in_progress
priority: 0
labels: [quality,security,wave1,logging]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-09T06:43:57Z
updated_at: 2026-08-12T22:44:28Z
---
Read docs/QUALITY_REMEDIATION_STRATEGY.md §0 (rules of engagement) and §2.6 (definition of done) before starting. Read the full finding text in docs/CODE_QUALITY_REVIEW.md. One finding, one branch, one PR. Re-verify on main first — cited line numbers are from 2026-07-05 and drift.

**Finding:** P2-C2, H1, H2, H14 | **Wave 1** | **Recipe:** R3 (strategy §2.5) | **Test level:** buffer-backed slog handler; assert secrets absent from every line
**Touches:** services/users/internal/services/email_verification_service.go, services/users/internal/services/auth_service.go, services/users/internal/api/handlers/auth_handler.go, apps/mobile/src/services/auth.ts

## Problem
- **P2-C2 (critical):** `email_verification_service.go:84-91` logs the **raw unhashed verification token assembled into a ready-to-click URL, plus the user's plaintext email**, at INFO. Anyone with production log access can take over any unverified account for 24h by replaying the URL. Exploit-ready, not a fragment.
- **H1:** raw refresh-token prefix logged on every `/auth/refresh`, in both layers — `auth_handler.go:295-307` and `auth_service.go:398-405`.
- **H2:** email-verification token logged in full cleartext.
- **H14:** `apps/mobile/src/services/auth.ts:353-417` logs token fragments and full refresh-failure response bodies via unconditional `console.log`.

Violates the project's own "never log passwords or hashes" principle.

## What to do
1. Grep the flagged files **and their whole services** for log calls near token/secret variables.
2. Test: run the refresh and verify-email flows with a buffer-backed slog handler; assert no line contains the token material. Then strip the logging.
3. Mobile: gate diagnostics behind `__DEV__`; never log response bodies of auth endpoints.

## Done when
- Log-capture tests fail before the fix for each flow; mobile logging is `__DEV__`-gated and carries no token material.


---
id: task_cef5
title: [C3] Refresh-token reuse detection never fires for its actual threat; its test asserts the wrong path
type: task
status: closed
priority: 1
labels: [quality,security,wave2,tests]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-09T06:46:25Z
updated_at: 2026-08-15T12:12:22Z
---
Read docs/QUALITY_REMEDIATION_STRATEGY.md §0 (rules of engagement) and §2.6 (definition of done) before starting. Read the full finding text in docs/CODE_QUALITY_REVIEW.md. One finding, one branch, one PR. Re-verify on main first — cited line numbers are from 2026-07-05 and drift.

**Finding:** C3 (root-caused by P2-C3) | **Wave 2** | **Recipe:** R9 (strategy §2.5) | **Test level:** strengthen the existing test until it fails on main
**Touches:** services/users/internal/database/refresh_token_repository.go, internal/services/auth_service.go, auth_service_test.go

## Problem
`refresh_token_repository.go:179-193` — `RevokeToken` **deletes** the row on rotation. `isTokenReused()` can only flag reuse if the same still-present row is queried twice within a 15s grace window. In the real threat scenario — an attacker replays a stolen token *after* the victim has legitimately rotated it — the old row is already deleted, the query returns `ErrNoRows`, and the flow takes the generic "not found" path: **no family revocation, no security alert, no `token_reuse_detected` audit event ever fires.**

The test at `auth_service_test.go:562-587` sleeps 6s and asserts only `assert.Error`, so it passes through the not-found path and never exercises reuse detection. This is the signature security feature of the rotation design and it is inert.

## What to do
1. **First make the existing test honest.** Strengthen it to assert the `token_reuse_detected` audit event and family revocation actually occur in the replay-after-rotation scenario. Watch it fail on current main for that reason. Drop the 6s sleep — it is testing nothing.
2. Then fix the code: stop hard-deleting rotated tokens. Mark them revoked and retain them for the reuse-detection window; sweep them on a schedule.
3. In the PR, state what the old test actually exercised — this is how the pattern gets recognised next time.

## Done when
- Replaying a rotated token revokes the whole family and emits the audit event, proven by a test that failed before the fix.

## Review

PR: https://github.com/andrew-craig/cairn-reader/pull/327 (branch `claude/epic-fefa-next-task-c5rq2k`)

Re-verified on main: confirmed. `RevokeToken` still hard-deleted on rotation; the old "token reuse detection" test slept 6s and asserted only `assert.Error`, which passed via the generic not-found path (no audit event, no family revocation) — reproduced this failing for the finding's exact reason before making any code change.

Fix: added `revoked_at` to `refresh_tokens` (migration 000007). `RevokeToken` now marks revoked instead of deleting; `GetRefreshTokenByHash` returns revoked rows so a replay can be recognized; `isTokenReused` now checks `RevokedAt != nil` (replacing the old `CreatedAt`/`LastUsedAt`-within-15s heuristic, which could only ever fire on a same-request race). Logout's `RevokeToken` treats an already-revoked token as not-found to preserve idempotent double-logout.

Old test exercised: the not-found path only, via the 6s-stale hard-deleted row — never the reuse-detection branch. Strengthened test proves (without the sleep) that replay emits `token_reuse_detected` and revokes the whole family.

Updated the mock/DB-backed unit tests that encoded the old delete/grace-period semantics, and the `revoked_at` schema + rotation-flow description in `services/users/AGENTS.md`. Ledger row ticked in `docs/QUALITY_REMEDIATION_STRATEGY.md` §1.3.

Noted in the PR (not fixed, confirmed pre-existing on main and unrelated): a JWT same-second determinism flake in `TestAuthService_RefreshAccessToken/success`, a body-encoding bug in `TestRegister` (handlers), and a fixture collision in `TestAuthService_ResetPassword/reset_revokes_refresh_tokens` — none touch refresh-token rotation/revocation.


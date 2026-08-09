---
id: task_cef5
title: [C3] Refresh-token reuse detection never fires for its actual threat; its test asserts the wrong path
type: task
status: open
priority: 1
labels: [quality,security,wave2,tests]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-09T06:46:25Z
updated_at: 2026-08-09T06:46:25Z
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


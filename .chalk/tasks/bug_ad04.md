---
id: bug_ad04
title: Web: refresh failure clears tokens even when the server was never reached
type: bug
status: open
priority: 2
labels: [quality,frontend,web,auth]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-09-12T12:12:50Z
updated_at: 2026-09-12T12:12:50Z
---
Split out of task_47c1 on 2026-09-12: this is a live bug on main, independent of the apps/shared consolidation.

## Problem
`apps/web/src/services/auth.ts:283` — `doRefreshAccessToken`'s catch block is commented "Clear tokens on ANY error (network, parse, HTTP) so the user re-authenticates" and calls `clearTokens()` unconditionally. Opening the web app offline (or against an unreachable backend) near access-token expiry forces a full re-login instead of letting the request be retried once connectivity returns.

Mobile had the identical bug. It was fixed on main by task_cab7 (#381) and refined by task_f19d (#391): `apps/mobile/src/services/auth.ts` now converts unreachable-server failures to `NetworkError` (via `apps/mobile/src/utils/http.ts` `fetchOrNetworkError`) and keeps tokens in that case, clearing them only on an actual credential rejection. task_f19d extended that to treat a 5xx refresh response as unreachable rather than as a rejection. The web copy never received either fix.

## What to do
Port the mobile semantics to web. Distinguish "refresh rejected by the server" (4xx from the refresh endpoint -> clear tokens, real logout) from "refresh never reached the server" (offline, DNS/TLS failure, abort, unparseable body, 5xx -> keep tokens, let the caller retry).

Match main's mobile design — do not invent a third model. Read `apps/mobile/src/services/auth.ts` `doRefreshAccessToken` + `ensureValidToken` and `apps/mobile/src/utils/errors.ts` first.

## Done when
- A test asserts that a refresh whose fetch rejects keeps both tokens, and fails against the current `clearTokens()`-on-any-error code.
- A test asserts a 401 from the refresh endpoint still clears tokens.
- A 5xx refresh response keeps tokens (task_f19d parity).

## Notes
The orphaned branch `tier4/mobile-fetchwithauth` (commit 19ae572, ex-PR #365) contains an earlier attempt at this using `RefreshRejectedError`/`RefreshNetworkError`. Main's `NetworkError` model supersedes it — use main's. See task_47c1 for the full history.

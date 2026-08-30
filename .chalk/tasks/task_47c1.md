---
id: task_47c1
title: [FE auth layer] Move the duplicated web/mobile auth.ts into apps/shared; fix H12 + offline-clears-tokens
type: task
status: open
priority: 2
labels: [quality,wave4,consolidation,frontend]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-09T06:53:56Z
updated_at: 2026-08-09T06:53:56Z
---
Read docs/QUALITY_REMEDIATION_STRATEGY.md §0 (rules of engagement) and §2.6 (definition of done) before starting. Read the full finding text in docs/CODE_QUALITY_REVIEW.md. One finding, one branch, one PR. Re-verify on main first — cited line numbers are from 2026-07-05 and drift.

**Finding:** Theme 3 (FE auth) + H12 + network-failure-clears-tokens | **Wave 4** | **Recipe:** R11 step 4 (strategy §2.5)
**Touches:** apps/web/src/services/auth.ts, apps/mobile/src/services/auth.ts, apps/shared, apps/mobile/src/services/read.ts + explore.ts

## Problem
`apps/web/src/services/auth.ts` is a near-verbatim copy of `apps/mobile/src/services/auth.ts` — the whole token-refresh state machine, duplicated. `apps/shared` already demonstrates the right injectable-adapter pattern (the server-URL logic), so the pattern exists; it just was not used here.

Two live bugs sit inside the duplicated code:
- **H12 (mobile):** a second 401 after token refresh is silently swallowed (`services/read.ts:50-72`, `explore.ts:86-108`) — only a *thrown* refresh failure logs the user out, so a re-rejected retry leaves the user stuck in a broken authenticated UI.
- **Both platforms:** any error in `doRefreshAccessToken`, **including plain offline**, calls `clearTokens()`. Opening the app offline near token expiry forces a full re-login instead of a retry.

## What to do
1. Port the web and mobile `auth.ts` tests onto the shared module **first**, using the `apps/shared` injectable-adapter pattern for the platform-specific storage/fetch bits.
2. Fix H12 and the offline-clears-tokens bug **in the shared copy**, so both platforms inherit the fix. Distinguish "refresh rejected by the server" from "refresh failed to reach the server".
3. Repoint web and mobile; delete both copies in the same PR as the last repoint.
4. The refresh-dedup mutex, stale-while-revalidate caching and optimistic-update rollback guards are **correct today** — preserve their behavior, do not redesign them.

## Done when
- One auth/token-refresh implementation lives in `apps/shared`; tests cover the second-401 and offline cases and fail on the old behavior.

---

## Sequencing note from the Cairn Simplification Audit (2026-08-17)

**Audit report:** https://claude.ai/code/artifact/286883fb-3f93-49c4-942f-4880251a409f

**Do task_ca2d first.** The audit found *intra-mobile* duplication that this task's cross-platform move sits on top of: `apps/mobile/src/services/read.ts:25-93` and `apps/mobile/src/services/explore.ts:61-127` each carry a **private** `fetchWithAuth` + `fetchWithAuthAndRetry` pair, byte-identical but for two comment lines. Neither goes through mobile's `AuthService`.

That is adjacent to this task, not the same ground — but landing task_ca2d first collapses three implementations to one, so **this task moves one thing instead of three**.

(For orientation: `apps/web/src/services/auth.ts:312` already carries the shared implementation, and its comment says it "mirrors mobile fetchWithAuth" — i.e. web mirrors a policy mobile itself duplicates twice.)

## ⚠️ Hazard that applies to this task too — error message text is load-bearing

`apps/mobile/src/utils/retry.ts:22-36` decides retryability by **lowercased substring match on the error message**:
```ts
const msg = error.message.toLowerCase();
if (msg.includes('not authenticated') || msg.includes('session expired') || ...) return false;
```
The substrings `Session expired. Please log in again.` and `Not authenticated` must survive the move into `apps/shared` intact. Rephrase either message and those auth errors silently become **retryable** — the client retries a request that can never succeed instead of prompting re-login. Pin the message-to-retryability contract with a test before refactoring.


## Added 2026-08-30 — `isRetryable` must branch on HTTP status, not prose
`apps/mobile/src/utils/retry.ts:19-37` decides retryability by lowercased substring match on `error.message` (`'not authenticated'`, `'session expired'`, `'unauthorized'`, `'forbidden'`, `'not found'`, `'bad request'`) and **defaults to retryable**. It is safe today only by accident: `read.ts`/`explore.ts` throw the server's message *outside* the `withRetry` callback, so server text never reaches `isRetryable` — an invariant nothing tests or enforces. Any future refactor that throws a server-derived message inside the retry callback silently turns a 401 into 3 retries with 1s/2s/4s backoff against the auth endpoint.
Now that the shared auth layer surfaces `response.status`, replace the prose matching with a status-based decision: retry only on network/abort errors and 5xx; never retry 4xx. Keep the existing client-side literals working (`'Session expired. Please log in again.'`, `'Not authenticated'`) for errors thrown before any response exists.

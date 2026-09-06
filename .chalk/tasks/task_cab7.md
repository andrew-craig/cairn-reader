---
id: task_cab7
title: Mobile: keep auth tokens when the server is unreachable
type: task
status: in_progress
priority: 2
labels: [mobile,offline,auth]
blocked_by: []
parent: feature_90a5
remote_task_url: null
created_at: 2026-09-05T23:46:28Z
updated_at: 2026-09-05T23:55:04Z
---
Prerequisite for feature_90a5 (the offline half of task_47c1). doRefreshAccessToken (apps/mobile/src/services/auth.ts:358-433) calls clearTokens() on ANY error, so opening the app offline within the 5-minute expiry buffer wipes tokens and logs the user out. Distinguish 'server rejected the refresh' (4xx: clear tokens, real logout) from 'could not reach the server' (network/timeout/5xx: keep tokens, throw). The thrown network error must NOT contain 'Session expired' or 'Not authenticated': utils/retry.ts:26-31 treats those as non-retryable and ExploreScreen.tsx:201-209 logs out on them. If task_47c1's shared auth consolidation lands first, this collapses into consuming that module. Verify: unit test that an offline refresh keeps tokens and throws a retryable error; 401 refresh still clears tokens and notifies listeners.

## Scope clarification (tech lead, 2026-09-06)
Verified on main: task_47c1 has NOT landed, mobile `auth.ts` is still standalone. This
task stands alone against the current file.

The bug is **three layers deep**, not one. Fixing only `doRefreshAccessToken` keeps the
tokens but the user is still logged out. Full chain on an offline launch near expiry:

1. `fetchWithAuth:468` → `ensureValidToken()`
2. `ensureValidToken:323-332` → `refreshAccessToken()` → network throw
3. `doRefreshAccessToken:425-431` catch → `clearTokens()` on ANY error  ← named in the task
4. `ensureValidToken:329-332` catches, returns **`false`** — collapsing "server rejected"
   and "server unreachable" into one signal  ← **not** named in the task
5. `fetchWithAuth:469-471` throws `'Session expired. Please log in again.'`
6. `retry.ts:26-31` → non-retryable; `ExploreScreen.tsx:202,387` → logout

### Rule to implement
Clear tokens **only** on a 4xx response from `/auth/refresh` (the server rejected the
credential). Every other outcome — network error, timeout, 5xx, malformed body, JSON
parse failure — keeps tokens and throws a retryable error. A server bug must not log a
user out.

### Plan
- [x] Add `NetworkError` (a named `Error` subclass) in `apps/mobile/src/utils/errors.ts`,
      exported from `utils/index.ts`. Message must not contain `session expired`,
      `not authenticated`, `unauthorized`, `forbidden`, `not found` or `bad request`
      (retry.ts substring-matches these). → verify: `withRetry` retries a thrown
      `NetworkError`.
- [x] `doRefreshAccessToken`: clear tokens only when the refresh response status is 4xx;
      otherwise keep tokens and throw `NetworkError`. → verify: offline refresh keeps
      tokens; 401 refresh clears them.
- [x] `ensureValidToken`: stop collapsing both failures into `false`. Rethrow the
      `NetworkError` (keep returning `false` for a genuine rejection). → verify: unit
      test asserts the distinction.
- [x] `fetchWithAuth`: let the `NetworkError` propagate instead of converting it to
      `'Session expired'`. In the 401 catch (`:505-509`), do not `clearTokens()` when
      the refresh failed for network reasons — rethrow instead. → verify: offline
      `fetchWithAuth` rejects with a retryable `NetworkError` and tokens survive.
- [x] Keep the `hasRefresh === false` branch (`:314-321`) as-is — no credential is a
      real logout.
- [x] (added 2026-09-06, reviewer correction) `AuthContext.checkAuthStatus`: on
      `NetworkError` from `ensureValidToken`, keep the stored session — load the
      persisted user via `AuthService.getUser()` and `setUser(...)` — instead of
      `setUser(null)`. Every other error keeps the existing `setUser(null)`
      behaviour. Import `NetworkError` from `../utils`. → verify: test asserts the
      context ends up with the stored user (not null) when `ensureValidToken`
      rejects with a `NetworkError`.

### Constraints
- Do not touch `retry.ts`'s `isRetryable` (unknown messages already retry), the refresh
  mutex, or `ExploreScreen`. `NetworkError` lives in `utils/` so services→utils stays
  the only direction (no import cycle).
  (Superseded 2026-09-06: this constraint originally also listed `AuthContext`. That was
  wrong — see the added plan item above. `AuthContext.checkAuthStatus` is layer 4 of the
  chain and must be changed for the fix to have any user-visible effect.)
- Do not wrap the raw `fetch` inside `fetchWithAuth` (the request itself, after the token
  check) so it throws `NetworkError`. Offline-with-a-valid-token still surfaces a raw
  `TypeError` today. That is task_c87c's job ("fetchWithAuth surfaces a distinguishable
  NetworkError"); handing it over rather than doing it here.
- Preserve the H14 log-redaction behaviour — `auth.test.ts` covers it.
- `NetworkError` is the canonical type task_c87c (phase 2) will consume; do not let that
  task invent a second one.

## Review (tech lead, 2026-09-06)
Implemented by a subagent, independently verified by the reviewer (not taken on report).

**Verification run by the reviewer, not the implementer:**
| Gate | Result |
|---|---|
| `npx jest` (whole mobile suite) | 18 suites, 129 tests, all pass |
| `npx jest src/services src/contexts` | 5 suites, 43 tests, all pass |
| `npm run type-check` | clean |
| `npm run lint` | 0 errors, 12 warnings — all pre-existing, none in a changed file |
| Reverting `auth.ts` + `AuthContext.tsx` only | exactly 5 behavioural failures |

The regression proof matters most: with the two source files reverted but
`errors.ts` and all tests kept, exactly the five behaviour-change tests fail
(offline refresh, 500, `ensureValidToken` NetworkError, `fetchWithAuth` offline,
AuthContext keeps session). The other two new tests — 401-still-clears-tokens and
`withRetry`-retries-`NetworkError` — pass against both old and new code, which is
correct: they are regression guards on behaviour that must not change, not
proofs of the fix.

**Accepted.** The four-layer change is minimal and each layer is load-bearing.
Test quality is good: `authFetch.test.ts` explicitly asserts the thrown message
contains neither `session expired` nor `not authenticated`, which is the exact
hazard that would silently reintroduce the bug, and the `withRetry` test drives
fake timers rather than sleeping through real backoff.

**Known gaps, deliberately left:**
- Offline with a *still-valid* token makes the raw `fetch` inside `fetchWithAuth`
  throw a bare `TypeError`, not a `NetworkError`. Handed to task_c87c.
- `AuthContext.test.tsx`'s second case covers the `isValid === false` branch, not
  the generic (non-`NetworkError`) catch path. Low value to add; noted only.
- Last-write-wins and the rest of the sync model are unaffected by this task.

**Not committed** — left in the working tree on branch
`task_cab7-offline-auth-tokens` pending the go-ahead.

## Review follow-up (PR #381, 2026-09-06)

Two review comments on the pushed commit (686f9fa):

1. **Important — `doRefreshAccessToken` cleared tokens on *any* 4xx.** That's
   broader than the invariant and reintroduces the bug for statuses that
   aren't a credential rejection. Confirmed against the backend: `/auth/*` is
   rate-limited per IP (10 req/min, `pkg/middleware/rate_limit.go:112`), so a
   busy device can get a genuine 429 on refresh — previously that logged the
   user out. The refresh endpoint's OpenAPI spec documents only 200/400/401;
   401 is the actual "invalid, expired, or reused refresh token" response, and
   400 means a malformed request (a client bug), not a rejected credential.
   Fix: clear tokens only on 401 or 403 (403 isn't documented for this
   endpoint but is included as a deliberate, definitive authorization
   rejection, e.g. a disabled account). Every other status — 400, 404, 429,
   5xx — now throws `NetworkError` and keeps the tokens. Added tests for 429
   and 400 (keep tokens, throw `NetworkError`); confirmed both fail against
   the pre-fix code. Trade-off accepted, not solved: routing 400/404 through
   `NetworkError` means `withRetry` retries them 3x with backoff before
   failing — wasted work for a permanently-failing status, but bounded, and
   `NetworkError` stays the single canonical type for task_c87c.

2. **Nit — `AuthContext.checkAuthStatus`'s `NetworkError` branch could throw
   an unhandled rejection.** `AuthService.getUser()` does `JSON.parse` on the
   persisted user and can throw on corrupt storage; that call sat inside the
   catch handler unguarded, widening the catch block's error surface beyond
   what the pre-existing code could do. Fixed with a small try/catch around
   the `getUser()` call that falls back to `setUser(null)` on failure. Added a
   test with a throwing `getUser` inside the `NetworkError` branch, asserting
   a null user and no unhandled rejection.

Verification (apps/mobile): `npx jest` — 18 suites / 132 tests pass (129
existing + 3 new). `npm run type-check` — clean. `npm run lint` — 0 errors, 12
pre-existing warnings, none in a touched file.

Not committed to origin — left as a local commit on
`task_cab7-offline-auth-tokens` pending review.

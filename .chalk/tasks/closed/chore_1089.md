---
id: chore_1089
title: Mobile: route every raw fetch through NetworkError conversion, and lint against new ones
type: chore
status: closed
priority: 2
labels: [mobile,offline,auth]
blocked_by: []
parent: feature_90a5
remote_task_url: null
created_at: 2026-09-06T07:46:07Z
updated_at: 2026-09-06T21:43:03Z
---
Structural guard for a bug class that has now recurred three times across task_cab7 and task_c87c. Converting a failed fetch into a NetworkError is done per-call-site, so every unwrapped fetch is a latent instance of 'a dropped connection is mistaken for a rejected credential'. Two of the three found so far logged the user out; each was found by review, one at a time, never by a test or a rule. There is currently nothing stopping a fourth. Verified on main: 9 raw fetch call sites remain in apps/mobile/src/services/auth.ts (login/mobile:95, register/mobile:123, login:149, register:175, logout:203, refresh:382 which has its own inline handling from task_cab7, upgrade:569, password:600) plus system.ts:13 (unauthenticated health check). task_c87c wrapped only the two in fetchWithAuth (the primary request and the 401 retry) because those were the two that caused a logout. The rest are lower severity but not harmless: AccountScreen surfaces error.message directly in an Alert, so an offline password change or account upgrade currently shows the user a raw 'Network request failed'. Scope: (1) promote AuthService.fetchOrNetworkError to a shared helper so system.ts can use it too. Likely apps/mobile/src/utils/http.ts, keeping the services-to-utils import direction intact and avoiding a cycle; NetworkError already lives in utils/errors.ts. (2) Route all remaining call sites through it. doRefreshAccessToken already distinguishes network from rejection inline and its 401/403-only rule is settled by task_cab7, so refactor it only if the result is provably identical, otherwise leave it and say why. (3) Add an ESLint rule banning bare fetch under apps/mobile/src/services (no-restricted-globals or no-restricted-syntax) with a message naming the helper, so the next call site fails lint rather than review. Verify: existing suite stays green; a test asserts each newly wrapped path rejects with NetworkError rather than TypeError; a deliberately added bare fetch in a service fails lint. Do not change any 4xx/credential-rejection semantics anywhere. This is an error-type and lint change only.

## Fourth instance (2026-09-07, PR #383 review)

A fourth window was found in review before this chore was started: `fetchOrNetworkError` itself matched the abort case with `error instanceof Error && error.name === 'AbortError'`, and a `DOMException` does not extend `Error`, so a timed-out 401 retry still reached `clearTokens()`. Fixed in task_c87c by matching on the name.

Two implications for this chore. First, the helper you promote to `utils/http.ts` must carry the *fixed* guard — copying the pre-fix shape would propagate the bug to all nine remaining call sites at once. Second, the lint rule is now the only thing that has a chance of catching the fifth: four instances, four different reviewers, zero caught by a test written in advance.

## Implementation (2026-09-06)

- Promoted `AuthService.fetchOrNetworkError` to `apps/mobile/src/utils/http.ts` as a standalone `fetchOrNetworkError(url, init?)` function, carrying the *fixed* abort guard (`instanceof TypeError || error.name === 'AbortError'`) from the PR #383 review fix. Exported from `utils/index.ts`. No cycle: `utils/http.ts` only depends on `utils/errors.ts`.
- Routed all 8 remaining `auth.ts` call sites through it: `loginWithDevice`, `registerWithDevice`, `loginWithEmail`, `registerWithEmail`, `logout`, `upgradeAccount`, `changePassword`, plus the two inside `fetchWithAuth` (moved from the old private method, behavior unchanged). `system.ts`'s `getServerVersion` now uses the same helper.
- `doRefreshAccessToken`'s fetch (line ~382) was left as a bare `fetch` on purpose: its surrounding `catch` already converts *any* thrown error to `NetworkError` (task_cab7), which is broader than `fetchOrNetworkError`'s TypeError/AbortError-only guard. Not provably identical, so left in place with an inline comment explaining why, plus a scoped `eslint-disable-next-line no-restricted-syntax` so the new lint rule doesn't flag this one approved exception.
- Added an ESLint override (`apps/mobile/.eslintrc.js`) scoping `no-restricted-syntax` to ban `CallExpression[callee.name='fetch']` under `src/services/**/*.ts`, with a message naming `fetchOrNetworkError`. Verified it fires on a deliberately added bare fetch, then removed the probe file.
- Tests added: `utils/http.test.ts` (unit tests for the helper: success passthrough, TypeError → NetworkError, plain-Error AbortError → NetworkError, non-Error DOMException AbortError → NetworkError, other errors rethrown untouched), `services/authNetworkErrors.test.ts` (each of the 7 behavior-changing auth.ts call sites rejects with NetworkError instead of TypeError when offline, plus a `logout` test), `services/system.test.ts` (success + NetworkError-on-unreachable for `getServerVersion`).
- No changes to any 4xx/credential-rejection semantics.

### Verification gate

| Gate | Result |
|---|---|
| `npx jest` | 30 suites, 177 tests, all pass (was 27/163 before this chore) |
| `npm run type-check` | clean |
| `npm run lint` | 0 errors, 12 warnings — all pre-existing, none in a touched file |
| Lint rule fires on a bare `fetch` added under `src/services/` | confirmed, then reverted (probe file, not committed) |
| Regression proof: `auth.ts`/`system.ts` reverted to pre-chore code, new tests re-run | 7 of 14 new tests fail — the 6 in `authNetworkErrors.test.ts` that assert *changed* behavior (`loginWithDevice`, `registerWithDevice`, `loginWithEmail`, `registerWithEmail`, `upgradeAccount`, `changePassword`) plus the 1 in `system.test.ts` (`getServerVersion` unreachable). `http.test.ts` (tests the standalone helper directly, unaffected by the revert) and the `logout` test (already swallowed the error pre-chore) pass unchanged both before and after — the `logout` test is a regression guard, not a proof, and is labeled as such in the test file. |

### Note on task_c87c

task_c87c (this chore's blocker) was already merged to `main` as PR #383 (commit `14a57f4`) but was still showing `in_progress` in the tracker. Closed it before starting this chore since the code state it depends on was already live.

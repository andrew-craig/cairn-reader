---
id: task_f19d
title: Mobile: treat a 5xx auth response as unreachable, not as a rejected credential
type: task
status: in_progress
priority: 2
labels: []
blocked_by: []
parent: feature_90a5
remote_task_url: null
created_at: 2026-09-12T03:54:54Z
updated_at: 2026-09-12T04:07:24Z
---


## Origin
Second half of the magpie-reviewer finding on PR #389 (task_5bd6). The first half —
`parseJsonResponse` throwing a plain `Error` carrying `NetworkError`'s message — was
folded into that PR. This half was deliberately left out because it changes auth error
semantics across four call sites rather than one line.

## The problem
`AuthService.loginWithDevice`, `registerWithDevice`, `loginWithEmail` and
`registerWithEmail` all do:

    const result = await this.parseJsonResponse(response);
    if (!response.ok) {
      throw new Error(result.message || result.error || '<verb> failed');
    }

A 5xx is therefore indistinguishable, by type, from a 4xx credential rejection. Two
consequences:
1. `LoginScreen.handleGetStarted` treats a 5xx device login as "login was rejected, so
   register instead" and makes a second doomed round trip — the exact symptom task_5bd6
   fixed for `NetworkError` and for unparseable bodies, still reachable via 5xx.
2. `NetworkError`'s own doc comment in `utils/errors.ts` already claims it covers "a 5xx
   response", so the code and the documented contract disagree today.

Same bug class as task_cab7, task_c87c, chore_1089 and the PR #389 fix — see the
2026-09-06 and 2026-09-07 entries in `LEARNINGS.md`.

## Scope to decide before implementing
- Whether 5xx becomes `NetworkError` at `parseJsonResponse`/entry-point level, or whether
  these four call sites should throw the `HttpError` (status-carrying) type that
  task_ebf1 added in `utils/errors.ts`, letting callers branch on the status. The second
  is probably the better long-term shape and would let `NetworkError`'s doc comment be
  corrected rather than satisfied.
- Check every caller that branches on these errors before changing the type —
  `utils/retry.ts` classifies by message substring, and `AccountScreen` surfaces
  `error.message` directly.
- Do not change 4xx/credential semantics.

## Verify
A 5xx device login makes exactly one network attempt and does not fall back to register;
a 4xx still falls back. Existing suite stays green; no new lint errors.

## Scope decision (tech lead, 2026-09-12)
The open question above is now closed: **throw `HttpError(status, message)`** at the four
`!response.ok` sites. Not "5xx becomes `NetworkError`". Reasons, so this is not
relitigated: `HttpError` already exists (task_ebf1) and `outbox.ts` already branches on
`.status`, so this is the type the codebase has converged on; the message text stays
byte-identical, so `retry.ts`'s substring classification and `AccountScreen`'s
`error.message` alert are untouched; and it keeps "the server is broken" distinct from
"I could not reach the server" instead of collapsing them, which is the exact mistake
this whole bug class is made of. `NetworkError`'s doc comment gets **corrected** (drop
the false "5xx response" clause) rather than satisfied.

### In scope
1. `loginWithDevice`, `registerWithDevice`, `loginWithEmail`, `registerWithEmail`
   (`auth.ts:106/134/160/186`): `throw new HttpError(response.status, result.message ||
   result.error || '<verb> failed')`. Message expression unchanged.
2. `LoginScreen.handleGetStarted` (`LoginScreen.tsx:64`): the register fallback runs
   only for a definitive rejection. A `NetworkError` already re-throws; an `HttpError`
   with `status >= 500` must re-throw too. A 4xx still falls through to
   `registerWithDevice()` — that is the only case the fallback was ever meant for.
3. Correct the `NetworkError` doc comment in `utils/errors.ts`.

### Out of scope — do not touch
- 4xx / credential semantics anywhere. A 401 still means what it means.
- `parseJsonResponse`: it keeps throwing `NetworkError` for an unparseable body. That
  was PR #389's fix (a captive portal returning an HTML page is genuinely "could not
  reach the server"), and it fires before the `!response.ok` check.
- `doRefreshAccessToken` (`auth.ts:410-432`): its status handling was decided by
  task_cab7 and is correct as it stands.
- `upgradeAccount` / `changePassword` / `fetchWithAuth` / `fetchWithAuthAndRetry`.
- `retry.ts`. Adding an `HttpError.status` branch to `isRetryable` is a real
  improvement but is not this task; raise it rather than folding it in.

### Verify
- A 5xx device login makes exactly **one** network attempt and surfaces the server's
  message; assert the fetch/`registerWithDevice` call count.
- A 4xx device login still falls back to register — regression guard, must pass before
  and after.
- Confirm each new test fails against pre-change code, and say in the Review section
  which ones are repros and which are regression guards (as PR #389 did).
- `npx jest` 260/260 → green with additions; `npx tsc --noEmit` clean; `npx eslint .`
  stays 0 errors / 12 warnings. Run jest directly, not `npm test` — the npm workspace
  wrapper reports a spurious non-zero exit.

## Review (implementer, 2026-09-12)

Implemented exactly what the scope decision specifies, nothing more.

### Files changed
- `apps/mobile/src/services/auth.ts` — `loginWithDevice`, `registerWithDevice`,
  `loginWithEmail`, `registerWithEmail` now `throw new HttpError(response.status,
  result.message || result.error || '<verb> failed')` instead of `throw new
  Error(...)`. Message expression byte-identical; only the constructor and the added
  `response.status` argument changed. Added `HttpError` to the existing `errors`
  import.
- `apps/mobile/src/screens/LoginScreen.tsx` — `handleGetStarted`'s inner catch now
  re-throws (skipping the register fallback) when `error instanceof NetworkError ||
  (error instanceof HttpError && error.status >= 500)`, instead of just
  `NetworkError`. Added `HttpError` to the existing `errors` import.
- `apps/mobile/src/utils/errors.ts` — `NetworkError`'s doc comment no longer claims
  it covers "a 5xx response"; now says "network failure, timeout, or a
  malformed/unparseable body". `HttpError`'s own doc comment already described the
  status-carrying contract correctly and was left untouched.
- `apps/mobile/src/services/auth.test.ts` — added a `jest.mock('expo-application', ...)`
  (needed so `loginWithDevice`/`registerWithDevice`'s `getDeviceId()` resolves in this
  file, which previously never exercised those two methods) and a new describe block,
  `AuthService auth entry points: non-2xx responses throw HttpError (task_f19d)`, with
  two `it.each` tables (4 entry points × {500, 401}) — 8 new tests.
- `apps/mobile/src/screens/LoginScreen.test.tsx` — added two tests to the existing
  `LoginScreen` describe block, using the real `HttpError` class.

### Disagreements / concerns
None. The scope decision's reasoning holds up under implementation — the change at
each of the four call sites was a one-token diff (`Error` → `HttpError`, plus the
status argument), and `LoginScreen`'s fix was a one-line condition addition. No
open questions found that would require revisiting the decision.

One observation surfaced while writing tests, logged here rather than folded in per
the task's own instruction to raise rather than fold in: `utils/retry.ts`'s
`isRetryable` still classifies purely by `instanceof NetworkError` / message
substring, so an `HttpError` (4xx or 5xx) falls through to the substring check same
as before — confirmed unchanged by the new tests (message text is byte-identical).
Giving `isRetryable` an explicit `HttpError.status`-based branch (matching
`fetchWithAuthAndRetry`'s own `status >= 500` check) was already noted as
out-of-scope in the task description; nothing found in this pass changes that
assessment.

### Verification

**Repro tests (fail against pre-change code, confirmed by stashing
`auth.ts`/`LoginScreen.tsx`/`errors.ts` and re-running with the test files in place):**
- `auth.test.ts` — all 8 new tests in the `non-2xx responses throw HttpError` block
  (`loginWithDevice`/`registerWithDevice`/`loginWithEmail`/`registerWithEmail` × {500,
  401}) fail pre-change: pre-change every one throws a plain `Error`, so
  `toBeInstanceOf(HttpError)` fails for all 8, including the 401 cases — the 401 cases
  are repros for the *type*, not for message/credential handling, which is unchanged
  and covered separately below.
- `LoginScreen.test.tsx` — "makes exactly one network attempt and does not fall back
  to register when device login fails with a 5xx" fails pre-change: with only
  `instanceof NetworkError` checked, an `HttpError(500, ...)` falls through to the
  register-fallback branch exactly like a 4xx would, so `registerWithDevice()` runs
  (mocked with no configured rejection, so it "succeeds") and `onLoginSuccess()` fires
  — the `await waitFor(() => expect(Alert.alert).toHaveBeenCalled())` assertion times
  out because no alert is ever shown. Confirms the exact symptom this task fixes.

**Regression guard (passes both before and after — confirmed in the same stash run):**
- `LoginScreen.test.tsx` — "still falls back to register when device login fails with
  a 4xx" passes unchanged pre- and post-fix: an `HttpError(401, ...)` is not
  `NetworkError` and `401 < 500`, so the fallback still runs in both versions.

**Full suite / tooling (post-fix, from `apps/mobile`, run directly with `npx jest`
per the task's note about the `npm test` wrapper):**
- `npx jest` → 37 suites / 270 tests, all passing (baseline 37/260 + 10 new: 8 in
  `auth.test.ts`, 2 in `LoginScreen.test.tsx`).
- `npx tsc --noEmit` → exit 0, no output (no errors; the two pre-existing
  `expo/tsconfig.base`/`baseUrl` warnings mentioned in the task did not print in this
  run, but the check is clean either way and nothing here touches `tsconfig.json`).
- `npx eslint .` → 0 errors, 12 warnings — identical count and identical warning list
  to the stated baseline (`SubscriptionListScreen.tsx` ×3, `BottomActionMenu.tsx` ×1,
  `AddArticleScreen.tsx` ×1, `VotesScreen.tsx` ×2, `YouScreen.tsx` ×1, `read.ts` ×4);
  none of the touched files appear in the warning list.

### Out-of-scope items respected
Did not touch `parseJsonResponse`, `doRefreshAccessToken`, `upgradeAccount`,
`changePassword`, `fetchWithAuth`, `fetchWithAuthAndRetry`, or `retry.ts`. 4xx/
credential semantics (message text, which statuses clear tokens, which trigger the
register fallback) are unchanged and covered by the existing and new regression
tests.

## Review addendum: scope amendment (implementer, 2026-09-12)

Implemented the amendment exactly as specified: the register fallback in
`LoginScreen.handleGetStarted` now runs only for `error instanceof HttpError &&
error.status === 401`; everything else (`NetworkError`, any other `HttpError`
status, any unrecognized error) propagates to the outer catch.

### Change
`apps/mobile/src/screens/LoginScreen.tsx`:
```ts
if (!(error instanceof HttpError && error.status === 401)) {
  // 401 is the only status meaning "this device isn't registered
  // yet" — every other case (unreachable, another HttpError status,
  // or an unrecognized error) is not evidence this device needs an
  // account, so propagate to the outer catch's alert instead of
  // risking a second doomed (or lockout-deepening) round trip.
  throw error;
}
// Device isn't registered; register it instead.
await AuthService.registerWithDevice();
```
`NetworkError` is no longer referenced in this file (the `>= 500` HttpError branch
and the explicit `NetworkError` check are both gone, subsumed by the inverted
default), so its import was dropped — otherwise it would have been an orphaned
import from this change.

### Tests
`apps/mobile/src/screens/LoginScreen.test.tsx`:
- Added `it.each([403, 429, 400])` — three new tests, each asserting exactly one
  `loginWithDevice` call, no `registerWithDevice` call, and the server's message
  surfaced via `Alert.alert`.
- Rewrote the pre-existing "still falls back to register when device login is
  rejected for a real (non-network) reason" test (which used a bare
  `new Error(...)`, i.e. an unrecognized error type) into "propagates (does not
  fall back to register) when device login fails with an unrecognized error
  type" — the old test's premise (unrecognized ⇒ fallback) is exactly what the
  amendment inverts, so keeping its old assertions would have made the suite
  contradict the new spec.
- Renamed/re-commented the existing 401 test to "still falls back to register
  when device login fails with a 401 (device not registered)" — same assertions,
  since 401 behaves identically under both the old (`>= 500` threshold, where 401
  fell below it) and new (`=== 401`) conditions.
- Left the 5xx-does-not-fall-back test and the offline/NetworkError tests
  unchanged — still valid, since `HttpError(500, ...)` and `NetworkError` both
  still fail the new `=== 401` check.

### Verification
Confirmed all four new/changed tests fail against the branch's committed state
(commit `d5da206`, i.e. the `status >= 500` version) by stashing only the
`LoginScreen.tsx` source edit and running `LoginScreen.test.tsx` with the test
file already updated:
```
✕ propagates (does not fall back to register) when device login fails with an unrecognized error type
✕ makes exactly one network attempt and does not fall back to register when device login fails with a 403
✕ makes exactly one network attempt and does not fall back to register when device login fails with a 429
✕ makes exactly one network attempt and does not fall back to register when device login fails with a 400
Tests: 4 failed, 7 passed, 11 total
```
The 7 passing tests included the 401 fallback guard and the 5xx/offline/
unparseable-body tests, confirming those were unaffected. Restored the source
edit (`git stash pop`) afterward.

Post-fix, from `apps/mobile`:
- `npx jest` → 37 suites / **273** tests, all passing (270 + 3 net new: +3 from
  `it.each([403, 429, 400])`, with the unrecognized-error and 401 tests rewritten
  in place rather than added).
- `npx tsc --noEmit` → exit 0, no output.
- `npx eslint .` → 0 errors, 12 warnings — same baseline list as before (none in
  touched files).

Did not push and did not open a PR, per instruction.

## Scope amendment (tech lead, 2026-09-12)
The original scope decision above said "a 4xx still falls through to
`registerWithDevice()` — that is the only case the fallback was ever meant for."
**That was wrong**, and the implementation faithfully followed it, so the gap is in the
instruction rather than the work. Confirmed against the backend source, not inferred:

`authService.LoginMobile` (`services/users/internal/services/auth_service.go:310`) and
`serviceErrorTable` (`services/users/internal/handlers/errors.go:37`) return:

| Status | Sentinel | Means | Register fallback correct? |
|---|---|---|---|
| 401 | `ErrInvalidCredentials` (from `ErrUserNotFound`) | device is not registered | **yes — the only one** |
| 403 | `ErrHybridAccountDeviceLogin` | an email/password account already exists | no |
| 429 | `ErrAccountLocked`, plus per-IP auth rate limiting (default 10/min) | locked out / throttled | no — a retry deepens it |
| 400 | `ErrInvalidInput` | empty `expo_device_id` | no |

`services/users/api/openapi.yaml` documents 400/401/403/413/500 for
`POST /auth/login/mobile`, matching. So `status >= 500` leaves 400/403/429 still making
a second doomed round trip — the exact bug class this task exists to kill, and on a
locked or rate-limited account the retry makes the situation worse.

`doRefreshAccessToken` in this same file already gets this right by enumerating which
statuses are credential rejections (401/403 there) with a comment naming rate limiting.
`LoginScreen` should be as precise.

**Amended requirement:** the register fallback runs **only** for
`error instanceof HttpError && error.status === 401`. Every other error — `NetworkError`,
any other `HttpError` status, anything unrecognized — propagates to the outer catch so
the user gets the server's message after exactly one attempt.

Note the inversion: unrecognized error types now propagate rather than triggering a
register attempt. That is the safer default — an unknown error is not evidence that this
device needs an account.

Add tests that 403, 429 and 400 each make exactly one network attempt, and keep the 401
fallback guard.

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

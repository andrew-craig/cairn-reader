---
id: task_c87c
title: Mobile: connectivity awareness and offline banner
type: task
status: in_progress
priority: 2
labels: [mobile,offline]
blocked_by: []
parent: feature_90a5
remote_task_url: null
created_at: 2026-09-05T23:36:10Z
updated_at: 2026-09-06T07:03:46Z
---
Phase 2 of feature_90a5. Add expo-network, a useNetworkStatus hook and an isOffline() helper. Global offline banner in RootNavigator. withRetry and fetchWithAuth surface a distinguishable NetworkError so callers can fall back to local data instead of showing generic errors.

## Scope clarification (tech lead, 2026-09-06)

Verified against `main` at d2f92e3. task_cab7 has landed, so `NetworkError`
already exists at `apps/mobile/src/utils/errors.ts` and is exported from
`src/utils/index.ts`. **Extend that type — do not define a second one.**

### The gap task_cab7 deliberately left you

task_cab7 made the *token refresh* path distinguish "server rejected the
credential" from "server unreachable". It explicitly did **not** wrap the raw
`fetch` inside `fetchWithAuth` — the request itself, after the token check. So
today, going offline while holding a still-valid access token makes
`fetchWithAuth` reject with a bare `TypeError: Network request failed`. Closing
that is the core of this task; it was handed over on purpose.

### Design decisions (made here — raise it if you disagree, don't silently deviate)

**1. Check what `expo-network` already gives you before writing a hook.** Recent
SDK versions export a `useNetworkState()` hook as well as
`getNetworkStateAsync()`. Per CLAUDE.md, do not reimplement a capability the
library already has. If `useNetworkState` covers it, `useNetworkStatus` is a thin
wrapper that narrows the result to what this app needs (or is dropped entirely and
the library hook used directly). Read the package's types before deciding, and
state in your report which one you found and why you chose your approach.

**2. The banner is a top overlay, not a layout row.** `RootNavigator` has no
shared chrome and every screen follows the edge-to-edge safe-area strategy in
`apps/mobile/CLAUDE.md` (content scrolls behind OS UI; headers pad by
`insets.top`). A banner that occupies a flex row above `Stack.Navigator` would
reflow every screen and break that. Render it as an absolutely-positioned sibling
*after* the navigator — `top: 0`, full width, `paddingTop: insets.top` — so it
paints above without reflowing anything. It covers the header area while offline;
that is accepted and is the conventional pattern.

**3. `isRetryable` gets an explicit `instanceof NetworkError` check.** Today
`NetworkError` is retryable only *by accident* — `isRetryable` returns true for
anything it doesn't recognise. That is a load-bearing accident sitting on
fragile substring matching. Add `if (error instanceof NetworkError) return true;`
ahead of the substring block. Keep the substring block exactly as it is; other
call sites still depend on it.

**4. Map both the network `TypeError` and the timeout `AbortError` to
`NetworkError` in `fetchWithAuth`.** `withRetry` is the only thing that aborts
these requests (its per-request timeout), so treating an abort as "couldn't reach
the server" is correct here. Note the tradeoff in your report: if a future caller
ever cancels deliberately, it would read as a network failure.

**5. Scope stops at *surfacing* the error.** "Callers can fall back to local data"
means the error is now distinguishable so they *can*. Wiring the actual local-data
fallback into the list and detail screens is task_a8a4 (store reads) and task_c55c
(offline body + "Not available offline" state). Do not build either here.

### Plan
- [ ] Add `expo-network` with `npx expo install expo-network` (SDK 54-compatible
      version, not a hand-picked one). → verify: the Metro Bundle CI check passes.
- [ ] Connectivity source per decision 1: `useNetworkStatus` in `src/hooks/`, and an
      `isOffline()` helper in `src/utils/` for non-React callers. → verify: unit test
      with `expo-network` mocked, covering online, offline, and unknown/undefined
      state (treat unknown as online — never block on an unreliable signal).
- [ ] `src/components/common/OfflineBanner.tsx`, exported from
      `src/components/common/index.ts`. Uses theme constants and `useSafeAreaInsets`
      per decision 2. → verify: renders when offline, renders nothing when online.
- [ ] Wire the banner into `RootNavigator` as an overlay sibling of the
      `Stack.Navigator`. → verify: test asserts the banner is present offline and
      absent online, and that screens still render.
- [ ] `auth.ts` `fetchWithAuth`: wrap the request `fetch` (the one after the token
      check, not the refresh call — that one is already handled) so `TypeError` and
      `AbortError` become `NetworkError`. → verify: test asserts an offline
      `fetchWithAuth` with a *valid, unexpired* token rejects with `NetworkError`,
      not a bare `TypeError`. This test must fail against pre-change code.
- [ ] `utils/retry.ts` `isRetryable`: explicit `instanceof NetworkError` check per
      decision 3. → verify: test asserts a `NetworkError` subclass instance whose
      message happens to contain e.g. 'not found' is still retryable.

### Constraints
- **Do not touch `ReadScreen`, `BookmarksScreen`, `ReadArticleDetailScreen`,
  `AddArticleScreen`, `ExploreScreen`, `AuthContext`, `services/storage.ts` or
  `services/index.ts`.** task_a8a4 is being implemented in parallel and owns every
  one of those files. Touching them will collide. If you believe you need a change
  in one, stop and report it instead of making it.
- Do not define a second network-error type, and do not change `NetworkError`'s
  default message — `retry.ts`'s substring classifier depends on it staying clear
  of 'session expired', 'not authenticated', 'unauthorized', 'forbidden',
  'not found' and 'bad request'.
- Do not change the refresh-token logic in `doRefreshAccessToken` or
  `ensureValidToken`. task_cab7 settled that; the 401/403-only rule stays.
- Do not remove or reorder the existing substring checks in `isRetryable`.
- Preserve the H14 log-redaction behaviour in `auth.ts` — never log token material,
  and gate diagnostic logging behind `__DEV__`. `auth.test.ts` covers this.

### Verification gate (all must pass before you report done)
1. `npx jest` — whole mobile suite green. Report suite/test counts.
2. `npm run type-check` — clean.
3. `npm run lint` — report exact error/warning counts; none in files you touched.
4. **Regression proof:** for each new test asserting *changed* behaviour, confirm it
   fails against the pre-change code. Report which failed and why. Tests that pass
   both before and after are regression guards, not proofs — label them as such.

Nothing here is device-tested. Real React Native offline behaviour (does RN reject
with a `TypeError`? does `expo-network` report correctly in airplane mode?) is
mocked in these tests and unverified — that is task_de93. Say so in your report.

## Review (tech lead, 2026-09-06)

Implemented by a subagent in an isolated worktree
(`task_c87c-connectivity`, branched from `main` d2f92e3), **independently
re-verified by the reviewer** — every gate below was run by the reviewer.

| Gate | Result |
|---|---|
| `npx jest` (whole mobile suite) | 22 suites, 146 tests, all pass |
| `npm run type-check` | clean |
| `npm run lint` | 0 errors, 12 warnings — all pre-existing, none in a touched file |
| Regression proof (3 source files reverted, new tests kept) | exactly 4 failures, matching the 4 behaviour tests |
| Regression proof (retry call site only reverted) | exactly 1 failure, the intended test |

The second proof matters: reverting *only* the retry-fetch call site to a bare
`fetch` — leaving the helper and the primary call site intact — fails exactly one
test. That isolates the fix to the specific call site rather than to `auth.ts`
generally.

**Accepted.**

### What was good
- Decision 1 was answered by reading `expo-network`'s `.d.ts` (v8.0.8) rather than
  assuming. `useNetworkState()` already handles subscribe/unsubscribe, so
  `useNetworkStatus` is a thin narrowing wrapper and `isOffline()` uses
  `getNetworkStateAsync()`, both sharing one classification rule. No
  reimplementation.
- `zIndex: 1001` on the banner, one above `TopBlurGradient`'s 1000 — an
  interaction that would otherwise have been found on a device.
- The new retry-path test asserts token survival through the real storage state
  (`getAccessToken()` returns the refreshed token) rather than by spying on
  `clearTokens`. That asserts the observable outcome, not the implementation.

### Defect found in review and fixed — caused by a reviewer spec error
The plan said to wrap "the request `fetch` ... not the refresh call — that one is
already handled". That was meant to exclude `doRefreshAccessToken`'s fetch, which
task_cab7 wrapped. It was read, reasonably, as also excluding the **401-retry**
fetch. The implementer flagged the gap rather than silently fixing or silently
omitting it, which is what surfaced the error.

The consequence was not cosmetic. Window: request returns 401 → `refreshAccessToken()`
succeeds → connection drops → the retry `fetch` throws a raw `TypeError` → misses
the `instanceof NetworkError` branch → falls into `clearTokens()` +
`'Session expired. Please log in again.'`. A network failure wiping tokens and
logging the user out: the exact bug task_cab7 exists to prevent, surviving in the
one window nobody had covered. Not exotic either — a flaky connection is precisely
what produces a mid-flight 401 followed by a dropped retry.

Fixed by extracting `AuthService.fetchOrNetworkError` and routing **both** call
sites through it. The existing `instanceof NetworkError` branch needed no change;
it now simply receives the right error type.

**Pattern worth noting across this feature:** `NetworkError` conversion is
per-call-site, so every unwrapped `fetch` is a latent instance of this same bug.
Three tasks in, it has appeared three times (refresh fetch in cab7, primary fetch
here, retry fetch here). Any new `fetch` added to `auth.ts` must go through
`fetchOrNetworkError`.

### Known gaps, deliberately left
- **Login/loading while offline shows no banner.** The banner covers only the
  authenticated `Stack.Navigator` branch. This is correct scoping — that case needs
  its own error copy, not a bare banner — but it is real and currently untracked.
  Should become its own task.
- **Nothing is device-tested.** `expo-network` is mocked throughout, so whether it
  reports correctly in airplane mode, and whether React Native actually rejects
  with `TypeError`, are both unverified. That is task_de93 and it remains the
  single largest unverified assumption in this feature.
- Metro Bundle unverified locally for the same aarch64/`hermesc` reason recorded in
  task_a8a4. Must be confirmed green on CI before merge.

**Not committed** — left in the working tree on branch `task_c87c-connectivity`
in worktree `.claude/worktrees/agent-a7098bec3eb06ba8a`, pending the go-ahead.

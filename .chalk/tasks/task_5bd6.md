---
id: task_5bd6
title: Mobile: offline-aware login and loading states
type: task
status: in_progress
priority: 3
labels: [mobile,offline,auth]
blocked_by: []
parent: feature_90a5
remote_task_url: null
created_at: 2026-09-06T07:46:17Z
updated_at: 2026-09-12T01:45:31Z
---
Gap identified during task_c87c review. The OfflineBanner added by task_c87c is rendered as a sibling of the authenticated Stack.Navigator in RootNavigator, so it only appears once the user is logged in. RootNavigator returns early for the two unauthenticated branches (the isLoading spinner and the LoginScreen), and neither shows any connectivity state. This was correctly scoped out of task_c87c rather than folded in, because a bare banner is the wrong answer here: unlike the authenticated screens, which can fall back to locally stored articles, login genuinely cannot proceed offline. The user needs to be told why the attempt will fail, not just that they are offline. Current behaviour: a login attempt while offline surfaces whatever the raw fetch rejection produces. Note that the auth entry points (loginWithDevice, registerWithDevice, loginWithEmail, registerWithEmail at auth.ts:95/123/149/175) are among the raw fetch call sites covered by chore_1089, so land that first and this task consumes the NetworkError it produces rather than string-matching. Scope: decide and implement the offline state for the isLoading and LoginScreen branches. Options worth weighing rather than assuming: extend the banner to those branches, add offline-specific copy plus a disabled or retrying submit button on LoginScreen, or both. Cold start offline while already holding valid tokens must keep working exactly as task_cab7 made it work. Do not regress that. Verify: test asserts LoginScreen shows an offline-specific message rather than a raw network error when a login is attempted offline, and that the already-authenticated cold-start-offline path is unchanged.

## Scope clarification (tech lead, 2026-09-12)
Reviewed against main after task_ebf1 merged. Decisions below are made, not open —
push back with a reason if one is wrong, don't silently pick differently.

### 1. Part of the premise is already fixed — do not re-fix it
The description (written 2026-09-06) says an offline login "surfaces whatever the raw
fetch rejection produces". chore_1089 has since landed: `loginWithDevice`,
`registerWithDevice`, `loginWithEmail` and `registerWithEmail` all go through
`fetchOrNetworkError`, so an offline attempt now rejects with `NetworkError`, and
`LoginScreen`'s `Alert.alert(..., error.message)` already shows "Unable to reach the
server. Please try again later." That is not a raw `TypeError` and is not the problem
any more. What remains is that nothing tells the user *before* they try, and the device
path below.

### 2. Real bug: a network failure is treated as "login failed, so register instead"
`LoginScreen.handleGetStarted` wraps `AuthService.loginWithDevice()` in a bare
`catch {}` that falls through to `AuthService.registerWithDevice()`. Offline, the first
call throws `NetworkError`, the catch swallows it, and the app makes a *second* doomed
round trip before alerting — the user waits through two timeouts.

This is the same error-class confusion as task_cab7, task_c87c and chore_1089 (see the
2026-09-06 and 2026-09-07 entries in `LEARNINGS.md`): "the server said no" and "I could
not reach the server" collapsed into one branch. Fix it here — the fallback to register
is only correct for a *rejection*; a `NetworkError` must propagate to the outer catch
untouched. Test that an offline device login makes exactly one network attempt.

### 3. Connectivity indication on the unauthenticated branches — render the banner
`RootNavigator` returns early for both the `isLoading` spinner and `LoginScreen`, so
`OfflineBanner` (a sibling of the authenticated `Stack.Navigator`) never renders for
either. Render it on all three branches.

**Do not move `SyncTriggerEffect` with it.** Its doc comment is explicit that living
inside the authenticated tree is what stops the sync trigger running logged out. It
stays exactly where it is. Add a test asserting it is not mounted on the logged-out
branches, so a future refactor cannot quietly move it.

### 4. The banner is necessary but not sufficient — add inline copy
The description is right that a bare banner is the wrong answer on its own: unlike the
authenticated screens, login genuinely cannot proceed offline, and "You're offline" does
not say that. Add offline-specific copy on `LoginScreen` explaining that signing in
needs a connection.

### 5. Keep the submit buttons enabled while offline
Rejecting the description's "disabled submit button" option. `expo-network` can report
offline when the device is in fact usable (captive portals, emulator quirks, a
transitional state), and a disabled button gives the user no way to find out. Offline
copy plus a failed attempt that explains itself is recoverable; a dead button is not.
The existing `isLoading` disabling stays as is.

### 6. Do not regress cold start offline with valid tokens
task_cab7 made an offline cold start keep the user logged in; `AuthContext` was the
fourth layer of that fix (see `LEARNINGS.md`, 2026-09-06). That path must be unchanged:
offline + valid tokens still lands in the authenticated tree, never on `LoginScreen`.
Cover it with a test.

### Out of scope
No change to any auth service, token handling, or 4xx/credential semantics. No retry or
auto-retry-on-reconnect for login — if the user is offline they can tap again once the
banner clears.

## Review (implementer, 2026-09-12)
Implemented all six scope items as specified; no disagreements.

- `LoginScreen.handleGetStarted`: the inner `catch` now re-throws `NetworkError`
  instead of falling through to `registerWithDevice`, so an offline device login
  makes exactly one network attempt.
- `LoginScreen`: added offline-specific copy under the header (`useNetworkStatus`),
  shown only while offline. Submit buttons unchanged (still gated on `isLoading`
  only, per item 5).
- `RootNavigator`: both the `isLoading` and `!isAuthenticated` early returns now
  render `<OfflineBanner />` alongside their content. `SyncTriggerEffect` untouched
  — still only rendered in the authenticated tree.
- New `LoginScreen.test.tsx` (5 tests) and 3 new tests in `RootNavigator.test.tsx`
  (7 total in that file). Verified against pre-fix code: the "exactly one network
  attempt" and "offline copy" LoginScreen tests fail without the fix (2/5 fail);
  the "banner on loading branch" and "banner on login branch" RootNavigator tests
  fail without the fix (2/7 fail). The cold-start-offline-with-valid-tokens
  RootNavigator test passes on both pre- and post-fix code, as expected — it is a
  regression guard for existing behavior (task_cab7), not a new-bug repro.

Verified: `npx tsc --noEmit` clean; `npm run lint` still 12 warnings / 0 errors
(unchanged from main); `npm test` 37/37 suites, 258/258 tests (was 36/250 before).

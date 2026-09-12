---
id: task_47c1
title: [FE auth layer] Move the duplicated web/mobile auth.ts into apps/shared; fix H12 + offline-clears-tokens
type: task
status: in_progress
priority: 2
labels: [quality,wave4,consolidation,frontend]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-09T06:53:56Z
updated_at: 2026-09-12T12:14:05Z
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


---

## Status re-assessment (2026-09-12) — the first attempt is orphaned, and the scope has shrunk

### What happened to the first attempt

This task was implemented once and lost to a stacked-PR mishap, 70 seconds wide:

| PR | Head → Base | Merged | Result |
|---|---|---|---|
| #364 | `tier4/mobile-fetchwithauth` → **`main`** | 2026-08-30 08:17:39Z | squash-merged as `a482cee` — task_ca2d landed |
| #365 | `fe-auth/shared-auth` → **`tier4/mobile-fetchwithauth`** | 2026-08-30 08:18:49Z | merged into **tier4**, not main |

PR #365's base was never repointed at `main` after #364 merged, so its merge commit `19ae572` landed on a branch that had already been squash-merged away. GitHub reports #365 as MERGED; its content reached no other ref. `origin/tier4/mobile-fetchwithauth` is now 2 ahead / 39 behind main and exists only to hold that commit.

**Do not rebase or merge that branch.** `apps/mobile/src/services/auth.ts` has drifted 218 lines on main since the merge base, concentrated in exactly the region `19ae572` deletes, and main's offline model is the better one (see below). Redo the work fresh; harvest `19ae572` as a design reference only.

### Superseded — drop from this task's scope

- **task_ca2d** (the sequencing prerequisite noted above) is **closed and on main**. `read.ts` and `explore.ts` now call `AuthService.fetchWithAuth*`; the two private copies are gone. Mobile is already down to one implementation.
- **The offline-clears-tokens bug, on mobile only**, was fixed independently on main by **task_cab7 (#381)** and refined by **task_f19d (#391)**. Main converts unreachable-server failures to `NetworkError` via `apps/mobile/src/utils/http.ts` `fetchOrNetworkError`, keeps tokens in that case, and (task_f19d) treats a 5xx refresh response as unreachable rather than as a rejection. This supersedes `19ae572`'s `RefreshRejectedError`/`RefreshNetworkError` split — **carry main's `NetworkError` model into the shared module, not the branch's.**

### Split out — no longer this task's job

- **bug_ad04** — web still clears tokens on *any* refresh error (`apps/web/src/services/auth.ts:283`). The mobile fix never crossed over.
- **bug_8123** — H12, the swallowed second 401, still live on **both** platforms (`apps/mobile/src/services/auth.ts:529`, `apps/web/src/services/auth.ts:342`).

Both are cheap standalone fixes and are deliberately independent of the consolidation. If they land first, this task inherits them; if this task lands first, they apply to the shared copy. Neither blocks the other — just don't fix the same bug twice in two places.

### What remains in scope here

The consolidation itself, and only that. Still true on main as of 2026-09-12:

- `apps/shared/src/services/` **does not exist**; `apps/shared/src/index.ts` exports only `./types`, `./config/api`, `./utils/throttle`.
- `apps/mobile/src/services/auth.ts` is 621 lines, `apps/web/src/services/auth.ts` is 372 — still two parallel token-refresh state machines.

Reusable from `19ae572` (read it, don't cherry-pick it):
- `apps/shared/src/services/auth.test.ts` — 250 lines of vitest against an in-memory `StorageAdapter`.
- `apps/shared/vitest.config.ts`, the `apps/shared` `test`/`type-check` scripts, and the `shared` job added to `.github/workflows/web-checks.yml`.
- The subclass shape: `class AuthService extends SharedAuthService` on mobile, adding only `getDeviceId`, `loginWithDevice`/`registerWithDevice`, `upgradeAccount`, `fetchWithAuthAndRetry`; web reduced to a re-export.

Unchanged constraints from the original task: preserve the refresh-dedup mutex, the 5-minute proactive-refresh buffer and the single-401 retry as-is, and keep the `Session expired. Please log in again.` / `Not authenticated` strings verbatim (see the hazard section above — `apps/mobile/src/utils/retry.ts:28` still matches on them).

---

## Implementation review (2026-09-13) — branch `task_47c1-shared-auth`

### What moved

`apps/shared/src/services/auth.ts` now holds the single token-refresh state
machine: the session accessors, the proactive-refresh buffer, the refresh-dedup
mutex, `ensureValidToken`, `fetchWithAuth`, email login/register, `logout` and
`changePassword`. Persistence goes through the `StorageAdapter` that
`configureStorage` already injects, so the module knows nothing about
AsyncStorage or localStorage.

Session state is module-level `let`, not static class fields. A static field
assigned as `this.x = …` from a static method reached through the subclass
creates an own property on the subclass and shadows the base — which would have
split the session in two on mobile the moment anything touched the base class
directly.

- `apps/mobile/src/services/auth.ts`: 621 lines → 100. Now
  `class AuthService extends SharedAuthService`, adding only `getDeviceId`,
  `loginWithDevice`/`registerWithDevice` (through the shared `authenticate`
  helper), `upgradeAccount`, and `fetchWithAuthAndRetry` (mobile's `withRetry`).
- `apps/web/src/services/auth.ts`: **deleted**, not reduced to a re-export
  (§4 — remove obsolete paths). Its five importers now take `AuthService` from
  `@cairn/shared` directly.

### Where NetworkError / fetchOrNetworkError ended up, and why

Moved to `apps/shared/src/utils/errors.ts` and `apps/shared/src/utils/http.ts`,
and deleted from `apps/mobile/src/utils/`. This was forced, not optional: the
shared state machine has to throw the same `NetworkError` that mobile's ~18
`instanceof NetworkError` call sites test against, and two copies of the class
would make every one of those checks silently false. The 18 import lines were
repointed at `@cairn/shared`; no other line in those files changed.

`utils/retry.ts` stayed in mobile — `withRetry` is only used by mobile's
`fetchWithAuthAndRetry`. `utils/http.test.ts` also stayed in mobile: its
`DOMException`-is-not-an-`Error` assertion is true in React Native's runtime and
false in Node, so it belongs in the suite that runs under jest-expo.

### Behavior changes I could not avoid

1. **Web inherits the mobile offline fix — bug_ad04 is fixed by this change.**
   Consolidating means picking one refresh-failure policy, and the only way to
   keep web's buggy one would have been a configuration seam whose sole purpose
   is preserving a bug. Web now distinguishes "server rejected the credential"
   (401/403 → clear tokens) from "never reached the server" (offline, unparseable
   body, 400/404/429/5xx → keep tokens, surface `NetworkError`). **bug_ad04
   should be closed as fixed here**, and 8 of the new shared tests fail against
   the old web behavior (verified by reinstating it and re-running).
2. **`fetchWithAuth` keeps web's `Headers`-based header builder**, not mobile's
   object spread — it merges a `Headers` instance or entry array correctly, and
   it always owns `Authorization` rather than letting caller headers override it.
   Two header assertions in `authFetch.test.ts` moved from `.Authorization` to
   `.get('Authorization')`.
3. **Mobile's `__DEV__`-gated diagnostic `console.log`s in the auth flow are
   gone.** `__DEV__` is a React Native global and does not exist in shared or on
   web. `console.error` on real failures is unchanged, as is the H14 guarantee
   that no token material is logged.
4. **Mobile `changePassword` now goes through `fetchWithAuth`** (web's version),
   so it gains the single 401 retry. `upgradeAccount` stayed on its own
   `ensureValidToken` + `fetchOrNetworkError` path — it is mobile-only, so
   nothing was gained by rewriting it.
5. Web's login/register now throw `HttpError` (carrying status) instead of plain
   `Error`, and the generic fallback message when the server sends none is
   `'Email login failed'` rather than `'Authentication failed'`. `HttpError`
   extends `Error`, so web's `err.message` handling is unaffected.

### Left alone deliberately

- **bug_8123 (H12)** — a second 401 after a successful refresh is still returned
  to the caller. Both platforms had this identical bug, so moving it preserved
  it exactly, in one place, with no seam required. There is a clearly labelled
  characterization test for it in the shared suite; bug_8123 will invert that
  assertion.
- `utils/retry.ts`'s prose-matching `isRetryable`. The load-bearing strings
  (`Session expired. Please log in again.`, `Not authenticated`) survive
  verbatim; replacing the classifier with an HTTP-status one is a separate,
  still-unfiled change.
- The 5-minute refresh buffer, the dedup mutex and the single-401 retry are
  byte-for-byte the same policy.

### Verification (all green)

| Package | Checks |
| --- | --- |
| `apps/shared` | `tsc --noEmit` clean · `vitest run` **32 passed** (new suite) |
| `apps/web` | `tsc --noEmit` clean · `eslint .` 0 errors · `vitest run` **31 passed** |
| `apps/mobile` | `tsc --noEmit` clean · `eslint .` 0 errors · `jest --ci` **273 passed / 37 suites** |

The 12 remaining mobile and 1 web eslint warnings are all pre-existing on main
(confirmed against `origin/main`) — none were introduced here.

`apps/shared` gained a `vitest` dev-dep, `test`/`type-check` scripts, and a
`shared` job in `web-checks.yml` (which already triggers on `apps/shared/**`, as
does `mobile-checks.yml`). `apps/mobile/jest.setup.ts` is new: the suite never
imports `src/config/init.ts`, so it has to wire `configureStorage` itself now
that the session persists through the injected adapter.

### Note for the reviewer

This branch was developed in a working tree shared with another concurrent
session that had unrelated uncommitted work in it (the bookmarks/votes count
fix, bug_0099). Only task_47c1 paths were staged; for
`apps/mobile/src/services/read.ts` and `apps/web/src/services/read.ts` — the two
files both changes touched — the committed version was the import change alone,
reconstructed from `HEAD`, leaving the other session's edits unstaged.

That work has since landed on main as #396, and this branch was rebased onto it
(`origin/main` @ 2fb8609). Both `read.ts` files merged without conflict: the
import edits and the new `countUserContents` methods sit in different regions.
The verification table above is from the post-rebase run, on a clean tree with
no foreign changes in it.

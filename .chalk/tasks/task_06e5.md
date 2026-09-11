---
id: task_06e5
title: Mobile: app-foreground and reconnect sync trigger
type: task
status: in_progress
priority: 2
labels: [mobile,offline]
blocked_by: []
parent: feature_90a5
remote_task_url: null
created_at: 2026-09-11T09:36:33Z
updated_at: 2026-09-11T09:37:18Z
---
Split out of task_ebf1 (tech lead, 2026-09-11). Phase 4a of feature_90a5.

Build the single AppState/connectivity listener that task_c55c deferred and task_ebf1
inherited, and wire the prefetch consumer to it now. task_ebf1 adds the outbox drain as
the second consumer on top of this.

Landing it separately because it is small, it fixes two bugs live on main today, and it
gives task_ebf1 a listener to plug into instead of building one inside a much larger
change.

## Scope
- A module owning the trigger (`src/services/syncTrigger.ts` + a hook, or equivalent)
  that fires when AppState transitions to `active` and when connectivity transitions
  offline -> online. Mounted once inside the authenticated tree (RootNavigator renders
  OfflineBanner in the same place) so it never runs logged out.
- Consumers run in a fixed order, and the order is the module's contract, not the
  caller's: outbox drain first (not present yet), then `ArticlePrefetchService.run()`.
  Leave the seam explicit so task_ebf1 adds the drain without restructuring.
- Re-entrancy: a trigger firing while a run is in progress must not stack.
  `ArticlePrefetchService` already has its own inFlight guard; do not duplicate it.
- Only fire on a *transition* to online, not on every render where isOffline is false.

## Bugs this fixes (both live on main)
1. Regaining connectivity does not prefetch until the user opens or pull-to-refreshes
   the Read tab (30s TTL).
2. `ReadArticleDetailScreen` shows "Not available offline"; if connectivity returns
   while that screen is open, the render guard (`!article.content && isOffline`) stops
   matching and the screen falls through to a blank `ArticleContent`. The
   content-loading effect keys off `initialArticle.id` and reads connectivity through
   `isOfflineRef`, so nothing re-triggers the fetch. Backing out and re-opening recovers.

## Design decision to make (state it in the task before coding)
Bug 2 is a screen-state bug, not a store bug: a background prefetch writing the body into
SQLite does not re-render an already-mounted detail screen. task_ebf1's note says to fix
it "via the reconnect trigger, not a screen-local listener". Recommended reading of that:
the screen keeps using the shared `useNetworkStatus()` hook (no new listener), but its
content-loading effect stops hiding connectivity behind `isOfflineRef` and re-runs when
connectivity returns and the body is still missing. The ref exists to stop flapping from
restarting a *successful* load, so gate the re-run on "still no content" rather than
removing the ref wholesale. If a different approach is cleaner, say so before building it.

## Verify
- Unit tests: fires on offline->online transition; fires on AppState background->active;
  does not fire on active->active or on a re-render with unchanged state; does not stack
  concurrent runs; does not run when unauthenticated.
- Unit test for the detail screen: mounted offline with no stored body renders
  "Not available offline", then connectivity returns -> content is fetched and rendered
  (never a blank body).
- Existing mobile suite stays green; no new lint errors.

## Plan (2026-09-11)
1. `src/services/syncTrigger.ts` — non-React module: a fixed-order `consumers` array
   (comment marks where task_ebf1 inserts the outbox drain, ahead of prefetch) and a
   `SyncTrigger.run()` with its own module-level `inFlight` guard for the *composite*
   sequence (separate from, and not a replacement for, `ArticlePrefetchService`'s own
   guard). → verify: unit test, run() invokes the consumer, second concurrent call is a
   no-op, a call after completion runs again.
2. `src/hooks/useSyncTrigger.ts` — hook: tracks the previous `AppState` status and the
   previous `isOffline` value in refs, calls `SyncTrigger.run()` only on
   background/inactive→active and true→false transitions. → verify: unit tests for each
   transition and each non-transition per the Verify list above.
3. `src/components/common/SyncTriggerEffect.tsx` — trivial component (`useSyncTrigger();
   return null;`) so it can be mounted the same way as `OfflineBanner` (a hook can't be
   called conditionally inside `RootNavigator` itself — it already returns early for
   `isLoading`/`!isAuthenticated` before any such call would sit). Exported from
   `components/common/index.ts`.
4. `RootNavigator.tsx` — render `<SyncTriggerEffect />` next to `<OfflineBanner />`,
   after the auth gate. → verify: extend `RootNavigator.test.tsx` to assert the hook
   fires only when authenticated.
5. `ReadArticleDetailScreen.tsx` — content-loading effect fix, see design decision below.

## Design decision: ReadArticleDetailScreen fix
Going with the recommended approach, with one simplification. Rather than adding a new
ref to track "do we still have no content", the effect can read `article.content`
straight from render-scope state: the effect's dependency array will include `isOffline`,
so every time it re-runs (on an offline→online *or* online→offline flip) the component
has already re-rendered first, and the closure the effect runs with therefore already
carries the latest `article` state — no separate ref needed to get a fresh read.
Concretely:
- Replace `isOfflineRef` (and the read through it) with the hook's `isOffline` value
  directly, added to the effect's dependency list.
- Replace the top-of-effect `if (initialArticle.content) return;` guard with
  `if (article.content) return;` — on the initial run these are equivalent (`article`
  state is seeded from `initialArticle`), but on a later, reconnect-triggered run it
  correctly reflects "still missing" instead of the frozen initial value, which is what
  stops a completed load from being restarted by later connectivity flapping (the
  property the ref used to provide).
- The rest of the effect body (store lookup, hash check, network fetch) is unchanged;
  it already re-does the store lookup first, so a body written by the reconnect
  prefetch's background `ArticleStore.saveBody` is picked up without a network call
  when the hash still matches.
No new listener is added; `useNetworkStatus()` is still the only connectivity source in
this screen.

## Review
Built as planned, no deviations.

- `src/services/syncTrigger.ts` (new) — `SyncTrigger.run()`, a fixed `consumers` array
  (currently just `ArticlePrefetchService.run()`, with a comment marking where
  task_ebf1's drain goes, ahead of it) and its own module-level `inFlight` guard so a
  trigger firing mid-run doesn't stack a second pass through the consumers. Does not
  touch `ArticlePrefetchService`'s own guard.
- `src/hooks/useSyncTrigger.ts` (new) — tracks previous `isOffline` and previous
  `AppState` status in refs/closure state; calls `SyncTrigger.run()` only on a
  true→false connectivity flip and a non-active→active AppState change, never on an
  unchanged re-render or an active→active event. Assumes 'active' as the AppState
  baseline at mount (the tree it lives in only renders in the foreground), so it only
  needs to catch later transitions.
- `src/components/common/SyncTriggerEffect.tsx` (new) + export from
  `components/common/index.ts` — trivial `useSyncTrigger(); return null;` wrapper,
  needed because RootNavigator itself can't call the hook conditionally (it returns
  early for the loading/unauthenticated states before any such call would sit).
- `src/navigation/RootNavigator.tsx` — renders `<SyncTriggerEffect />` next to
  `<OfflineBanner />`, after the auth gate, so it never mounts logged out.
- `src/screens/ReadArticleDetailScreen.tsx` — content-loading effect now depends on
  `isOffline` directly (dropped `isOfflineRef`) and gates re-entry on `article.content`
  instead of the frozen `initialArticle.content`, per the design decision above. Fixes
  the blank-body bug: reconnecting re-runs the effect, which re-checks the store first
  (picking up a body the reconnect-triggered prefetch may have just written) and falls
  back to a network fetch, so the screen goes straight from "Not available offline" to
  rendered content.

Tests added: `src/services/syncTrigger.test.ts`, `src/hooks/useSyncTrigger.test.ts`, two
cases in `src/navigation/RootNavigator.test.tsx` (sync trigger mounts only when
authenticated), one case in `src/screens/ReadArticleDetailScreen.test.tsx` (reconnect
fetches and renders content, never a blank body).

Tradeoff: `useSyncTrigger`'s AppState baseline assumes the tree is already in the
foreground at mount (see above) rather than reading `AppState.currentState`. That value
is a plain function in this project's RN jest mock (not a string), which would have
made the very first genuine `active` event look like a transition in tests; hardcoding
the assumption sidesteps that without weakening the real-world behavior, since the
mount location already guarantees the app is foregrounded when this hook attaches.

`npm test`, `npm run type-check`, and `npm run lint` all pass from `apps/mobile` (lint:
0 errors, only pre-existing warnings on files this task didn't touch).

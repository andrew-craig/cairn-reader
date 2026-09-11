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

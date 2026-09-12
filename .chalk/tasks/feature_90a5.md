---
id: feature_90a5
title: Offline reading mode
type: feature
status: open
priority: 3
labels: []
blocked_by: []
parent: null
remote_task_url: null
created_at: 2026-04-08T09:02:33Z
updated_at: 2026-04-08T09:02:33Z
---


## Summary

Saved articles are readable on the mobile app with no network, and actions taken while
offline (status, favorite, reading position, archive) sync when the connection returns.
This delivers the product requirement in `docs/product_requirements.md` line 31
("saved content is stored on device for access offline. Changes are synced when the
connection is restored").

## Current state (investigated 2026-09-05)

- **Nothing is readable offline today.** `ReadScreen` shows a stale list from
  `READ_LIST_CACHE_KEY` (task_5229), but list items carry no `cleaned_html` since the
  list/detail split (task_bfad). `ReadArticleDetailScreen` always calls
  `ReadService.getContentById()` over the network.
- **Two disconnected AsyncStorage caches** (`ARTICLES_KEY`, `READ_LIST_CACHE_KEY`) in
  `apps/mobile/src/services/storage.ts` (task_179f). Neither holds article bodies.
- **No connectivity detection.** No netinfo / expo-network; every failure is inferred
  from a thrown fetch error. `withRetry` retries list calls only; mutations are
  fire-and-forget with `console.error`. `BookmarksScreen` has no cache at all.
- **Detail screen is route-params only** (`ReadArticleDetailScreen.tsx:35`) and shows a
  blank body when the lazy detail fetch fails.
- **Auth clears tokens on any refresh error** (`auth.ts:426-432`), so opening the app
  offline near token expiry logs the user out (task_47c1). Hard blocker.
- **Content is inline HTML** rendered by `react-native-render-html` from a string, so a
  stored body renders offline with no further work. Images are remote `<img src>` URLs;
  the backend never proxies or stores images.
- **Backend has no delta-sync, ETag or version endpoint**, but every list item carries
  `content.content_hash` (`ContentSummaryResponse`, `apps/shared/src/types/read.ts:5-7`),
  so a client can detect changed bodies from a list page without fetching detail.
  PATCH is field-level and replay-safe but last-write-wins with no client timestamp.
  DELETE returns 404 on replay. POST add-URL has no idempotency key.
- **Web** has no service worker, manifest or IndexedDB, and its requirements doc lists
  offline/PWA as an explicit non-goal (`web_app_requirements.md` lines 18, 447).

## Decisions (assumptions to confirm before implementation)

1. **Mobile only.** Web/PWA stays a non-goal per its requirements doc.
2. **Automatic, not manual.** Bodies of every `unread`/`reading` article in the Read list
   are downloaded in the background after each list sync, newest first, capped
   (the 100 most recent Read-list articles). No per-article "download" button in v1.
3. **Images stay remote.** Consistent with the product doc's out-of-scope note
   ("images loaded from source"). Offline articles show text; images fail silently.
   Image caching is a follow-up.
4. **expo-sqlite for the local store**, not AsyncStorage. Bodies are up to 5MB each and
   Android AsyncStorage has a 6MB total default cap. SQLite gives per-row storage,
   queries by status, and an outbox table. Expo SDK 54 ships it.
5. **expo-network for connectivity** (SDK 54 `useNetworkState`). No extra native config.
6. **Last-write-wins on sync.** No backend change for conflict resolution in v1; replayed
   PATCHes may clobber a newer value from another device. Acceptable for a
   single-device-dominant read-it-later app; noted as a known limitation.
7. **Explore article content is never synced.** Explore and add-URL stay online-only.

## Plan

Each phase ships end to end on its own and is tracked as a sub-task.

- [x] **Prerequisite — task_cab7** Keep auth tokens when the server is unreachable.
      `doRefreshAccessToken` must not `clearTokens()` on a network error, only on a
      server rejection, and the thrown network error must not carry the auth strings
      that `retry.ts` and `ExploreScreen` key on. Collapses into task_47c1 if that
      lands first. Blocks phase 3.
      → verify: offline refresh keeps tokens and throws a retryable error; 401 still
      clears tokens.
- [x] **Phase 1 — task_a8a4** Single SQLite local article store. Replaces `ARTICLES_KEY`
      and `READ_LIST_CACHE_KEY`; `ReadScreen`, `BookmarksScreen` (query `is_favorite`)
      and the detail screen read store-first then refresh. Cleared on logout. Supersedes the dual-cache part of task_179f
      (task_179f now blocked by this and should be re-scoped afterwards).
      → verify: existing `storage.test.ts`, `ReadScreen` behaviour unchanged online;
      new store tests for upsert/list/clear.
- [x] **Phase 2 — task_c87c** Connectivity awareness. `useNetworkStatus`, global offline
      banner, `NetworkError` type surfaced by `fetchWithAuth`/`withRetry`.
      → verify: unit tests for error classification; banner toggles with mocked
      network state.
- [x] **Phase 3 — task_c55c** Body prefetch and offline reading. Background download of
      `cleaned_html` into the store (bounded concurrency, newest first, cap of 100,
      Read list only, never Explore content). Diff by `content_hash` from the list
      page so unchanged bodies are never re-downloaded. Detail screen renders the
      stored body immediately, resolves the article by id from the store (not only
      route params), and shows a "Not available offline" state when offline with no
      stored body. Evicts on archive/delete and beyond cap.
      → verify: airplane-mode manual test reads a previously synced article; unit
      tests for prefetch selection, cap eviction and staleness check.
- [x] **Phase 4 — task_ebf1** Offline mutation outbox. Store-first writes, outbox
      table drained on reconnect, app foreground and pull-to-refresh, in `created_at`
      order. 2xx deletes the row; 404 on a replayed DELETE counts as success;
      definitive 4xx (except 401) drops the row and logs; network/5xx/401 keeps the
      row, bumps `attempts` and halts the batch to preserve order. scroll_position
      coalesced per article. Archive error no longer swallowed.
      → verify: unit tests for enqueue/drain/coalesce/404 handling; manual test:
      archive and favorite offline, reconnect, server state matches.
- [ ] **Phase 5 — task_de93** End-to-end airplane-mode QA pass, coverage check, fill in the
      Review section, capture lessons in `LEARNINGS.md`.
- [ ] **Phase 6 — task_43fc** Docs: `apps/mobile/CLAUDE.md`, `docs/ARCHITECTURE.md`,
      `docs/product_requirements.md` (move out of Future Enhancements), non-goals.

## Relationship to PR #379

PR #379 was an earlier planning attempt with seven sub-tasks. Its content_hash diffing,
reader empty state, by-id resolution, Bookmarks caching, outbox replay rules and QA pass
are folded in here. Its Explore body sync and Settings section are dropped per the
confirmed decisions. PR #379 should be closed without merging so its sub-tasks are not
created alongside these.

## Out of scope for this feature

- Web app offline / PWA.
- Image download or proxying (client or backend). Tracked by feature_9d64.
- Backend delta-sync (`updated_since`) or ETag support. Would cheapen phase 3 syncs at
  scale; raise as a separate backend task if sync cost becomes a problem.
- Server-side conflict resolution (client timestamps / versions on PATCH).
- Manual per-article download and a Settings "Offline reading" section (cached count,
  clear data). Logout already clears the store.

## Rough sizing

| Phase | Size |
|---|---|
| task_cab7 auth prerequisite | S |
| task_a8a4 store | M |
| task_c87c connectivity | S |
| task_c55c prefetch + offline read | M |
| task_ebf1 outbox + sync | M |
| task_de93 QA pass | S |
| task_43fc docs | S |

## Review (task_de93, 2026-09-12)

Written from the merged code on `main` and the closed sub-tasks
(`task_cab7`, `task_a8a4`, `task_c87c`, `task_c55c`, `task_ebf1`, `task_06e5`, `task_5bd6`,
`task_f19d`, all in `.chalk/tasks/closed/`), not from this plan text. Checkboxes
above are ticked for the phases confirmed shipped; Phase 5 (this task) and Phase 6
(`task_43fc`, docs) are not done and stay open.

### What shipped

**Auth prerequisite (task_cab7).** The plan described this as fixing
`doRefreshAccessToken`; the shipped fix is four layers deep — `doRefreshAccessToken`
(clear tokens only on 401/403, everything else throws `NetworkError`),
`ensureValidToken` (stops collapsing "rejected" and "unreachable" into one `false`),
`fetchWithAuth` (lets `NetworkError` propagate instead of converting it to "Session
expired"), and `AuthContext.checkAuthStatus` (keeps the stored session on
`NetworkError` instead of `setUser(null)`). `NetworkError`/`HttpError` in
`utils/errors.ts` are the canonical error types every later phase builds on.

**Local store (task_a8a4).** `ArticleStore` (`src/services/articleStore.ts`) is a single
SQLite table (`expo-sqlite`), replacing the two disconnected AsyncStorage caches
(`ARTICLES_KEY`, `READ_LIST_CACHE_KEY`). `ReadScreen`, `BookmarksScreen` and
`ReadArticleDetailScreen` read store-first, then refresh from the network. The schema
migrates via a `PRAGMA user_version` ladder now shared with the outbox table (extracted
into `src/services/db.ts` in task_ebf1). Cleared on logout (`AuthContext.logout`).

**Connectivity layer (task_c87c).** `useNetworkStatus`/`isOffline()`
(`src/hooks/useNetworkStatus.ts`, `src/utils/network.ts`) wrap `expo-network`'s
`useNetworkState`. A global `OfflineBanner` renders as an absolutely-positioned overlay
so it never reflows a screen. `fetchWithAuth` wraps both its primary fetch and its
401-retry fetch to surface `NetworkError` instead of a bare `TypeError`/`AbortError` —
review found and closed two additional unwrapped-fetch windows beyond the one named in
the task description, plus a `DOMException`-vs-`TypeError` abort-detection gap in a
follow-up review round (PR #383).

**Prefetch (task_c55c).** `ArticlePrefetchService` (`src/services/articlePrefetch.ts`)
downloads `cleaned_html` for unread/reading Read-list articles with no cached body:
3-worker bounded concurrency, newest-first, capped at the 100 most recent, diffed by
`content.content_hash` (plumbed onto the shared `Article` type as `contentHash` — a
blocker the original plan didn't name) so an unchanged body is never re-downloaded.
Never runs offline; aborts the remaining batch on the first `NetworkError`; single-flight.
`ReadArticleDetailScreen` resolves the article by id from the store (not just route
params), renders a stored body immediately when the hash matches, and shows
"Not available offline" when offline with no stored body.

**Outbox and reconnect sync (task_ebf1, task_06e5).** The app-foreground/reconnect
listener was split out of task_ebf1 into its own task (`task_06e5`, "Phase 4a") because
both prefetch and the outbox needed the same trigger and building it twice was waste —
`SyncTrigger`/`useSyncTrigger` (`src/services/syncTrigger.ts`,
`src/hooks/useSyncTrigger.ts`) fire on an offline→online transition and an
AppState background→active transition, running the outbox drain before
`ArticlePrefetchService.run()` in a fixed order, isolated so one consumer's rejection
doesn't stop the other. `ArticleMutations` (`src/services/articleMutations.ts`) is the
single facade for all six mutation call sites (mark-read, "reading", scroll position,
favorite, archive); `Outbox` (`src/services/outbox.ts`) coalesces by
`(article_id, field)`, classifies replay outcomes (2xx/404-on-delete → success,
definitive 4xx-except-401 → drop, network/5xx/401 → halt the whole batch to preserve
order), and a store-level guard in `articleStore.ts`'s `UPSERT_SQL` freezes the four
user-state columns (and skips the row entirely for a pending delete) for any article
with a pending outbox row, so a list sync arriving before the drain can't clobber a
queued offline edit or resurrect an archived article.

**Auth/login hardening exposed by the offline work (task_5bd6, task_f19d).** Neither
was in the original 6-phase plan; both surfaced from testing the offline auth path.
`task_5bd6`: the offline banner and offline-specific copy now render on the
pre-authenticated branches too (loading spinner, `LoginScreen`), a device login that
fails with `NetworkError` no longer falls through to a second doomed
`registerWithDevice()` call, and submit buttons stay enabled offline (deliberate —
`expo-network` can misreport a usable connection as offline, and a dead button gives
no recovery path). `task_f19d`: the four device/email login and register entry points
now throw `HttpError(status, message)` instead of a plain `Error`, so `LoginScreen`'s
register-fallback logic runs only for a real 401 ("device not registered") and not for
a 5xx, 403, 429 or 400 — closing the same "server said no" vs. "couldn't reach the
server" confusion this whole feature keeps surfacing (see `LEARNINGS.md`).

### Divergences from the plan text above

1. The plan folds the reconnect/foreground trigger into Phase 4 (task_ebf1). It shipped
   as a separate task (`task_06e5`) landed just ahead of task_ebf1, because task_c55c
   (Phase 3) had already deferred it and task_ebf1 needed the same listener rather than
   building a second one.
2. The plan's Phase 4 line says the outbox is "drained on reconnect, app foreground and
   pull-to-refresh." Pull-to-refresh only drains through the same path after task_ebf1's
   own scope clarification (item G) routed `ReadScreen`'s pull-to-refresh through
   `SyncTrigger.run()` instead of calling `ArticlePrefetchService.run()` directly, which
   is what Phase 3 had shipped.
3. `content_hash` diffing (Phase 3) required an unplanned prerequisite: the shared
   `Article` type had no field for it and neither `read.ts` transform populated it. That
   plumbing had to ship before anything else in Phase 3 could work.
4. Two auth/login sub-tasks not in the original 6-phase list (`task_5bd6`, `task_f19d`)
   were added afterward, both consequences of testing the offline login/cold-start path
   more carefully than the plan anticipated.
5. Phase 1's own scope note says bodies are "cached opportunistically, not prefetched" —
   correct for what Phase 1 shipped, but superseded by Phase 3's bulk background
   prefetch with hash diffing, which is what the plan's Phase 3 description already
   called for.

### Known limitations, accepted

- **Last-write-wins (decision 6).** No client timestamp or version on PATCH. task_ebf1's
  store-level guard prevents the worst case — a list sync clobbering a *queued* offline
  edit before it replays — but does not solve genuine concurrent-device conflicts once
  both writes reach the server.
- **Images stay remote (decision 3).** Offline articles render text only; broken images
  are expected. Tracked separately as `feature_9d64`.
- **Explore stays online-only (decision 7).** Never cached, never synced, never written
  to `ArticleStore` — verified untouched (`explore.ts`, `EXPLORE_CACHE_KEY`) across every
  phase.
- **No backend delta-sync or ETag support.** `content_hash` diffing avoids re-downloading
  an unchanged *body*, but the list page itself is still fetched in full on every sync.
- **No server-side conflict resolution** beyond the client-side guard above.
- **No manual per-article download and no Settings "Offline reading" section.** Logout
  clearing the store is the only user-facing control.
- **An article archived on another device lingers locally** until the local archive path
  runs — the accepted consequence of task_a8a4's decision 2 (sync is upsert-only, never
  bulk-delete).
- **Nothing in Phases 1-4 was device-tested before this task.** Every network,
  connectivity and SQLite behaviour was exercised through mocks and a `node:sqlite`-backed
  real-SQL test double. The on-device airplane-mode walkthrough is Phase 5's remaining,
  user-run half (see the checklist in `task_de93`); this feature does not ship as
  verified-on-hardware until that runs.

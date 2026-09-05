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
  fire-and-forget with `console.error`.
- **Auth clears tokens on any refresh error** (`auth.ts:426-432`), so opening the app
  offline near token expiry logs the user out (task_47c1). Hard blocker.
- **Content is inline HTML** rendered by `react-native-render-html` from a string, so a
  stored body renders offline with no further work. Images are remote `<img src>` URLs;
  the backend never proxies or stores images.
- **Backend has no delta-sync, ETag or version.** `updated_at` on `contents` and
  `user_contents` is the only staleness signal. PATCH is field-level and replay-safe but
  last-write-wins with no client timestamp. DELETE returns 404 on replay. POST add-URL
  has no idempotency key.
- **Web** has no service worker, manifest or IndexedDB, and its requirements doc lists
  offline/PWA as an explicit non-goal (`web_app_requirements.md` lines 18, 447).

## Decisions (assumptions to confirm before implementation)

1. **Mobile only.** Web/PWA stays a non-goal per its requirements doc.
2. **Automatic, not manual.** Bodies of every `unread`/`reading` article in the Read list
   are downloaded in the background after each list sync, newest first, capped
   (proposed: 200 articles). No per-article "download" button in v1.
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
7. **Explore content and add-URL stay online-only.**

## Plan

Each phase ships end to end on its own and is tracked as a sub-task.

- [ ] **Prerequisite: task_47c1** (offline part only): `doRefreshAccessToken` must not
      `clearTokens()` on a network error, only on a server rejection. Blocks phase 3.
- [ ] **Phase 1 — task_a8a4** Single SQLite local article store. Replaces `ARTICLES_KEY`
      and `READ_LIST_CACHE_KEY`; `ReadScreen` and detail screen read store-first then
      refresh. Cleared on logout. Supersedes the dual-cache part of task_179f
      (task_179f now blocked by this and should be re-scoped afterwards).
      → verify: existing `storage.test.ts`, `ReadScreen` behaviour unchanged online;
      new store tests for upsert/list/clear.
- [ ] **Phase 2 — task_c87c** Connectivity awareness. `useNetworkStatus`, global offline
      banner, `NetworkError` type surfaced by `fetchWithAuth`/`withRetry`.
      → verify: unit tests for error classification; banner toggles with mocked
      network state.
- [ ] **Phase 3 — task_c55c** Body prefetch and offline reading. Background download of
      `cleaned_html` into the store (bounded concurrency, newest first, cap), detail
      screen renders stored body immediately, refreshes when `content.updated_at`
      changes, evicts on archive/delete and beyond cap.
      → verify: airplane-mode manual test reads a previously synced article; unit
      tests for prefetch selection, cap eviction and staleness check.
- [ ] **Phase 4 — task_ebf1** Offline mutation outbox. Store-first writes, outbox
      table drained on reconnect and app foreground, scroll_position coalesced per
      article, 404-on-replayed-DELETE treated as success, archive error no longer
      swallowed.
      → verify: unit tests for enqueue/drain/coalesce/404 handling; manual test:
      archive and favorite offline, reconnect, server state matches.
- [ ] **Phase 5 — task_43fc** Docs: `apps/mobile/CLAUDE.md`, `docs/ARCHITECTURE.md`,
      `docs/product_requirements.md` (move out of Future Enhancements), non-goals.

## Out of scope for this feature

- Web app offline / PWA.
- Image download or proxying (client or backend).
- Backend delta-sync (`updated_since`) or ETag support. Would cheapen phase 3 syncs at
  scale; raise as a separate backend task if sync cost becomes a problem.
- Server-side conflict resolution (client timestamps / versions on PATCH).
- Manual per-article download and storage-usage UI.

## Rough sizing

| Phase | Size |
|---|---|
| task_47c1 offline part | S |
| task_a8a4 store | M |
| task_c87c connectivity | S |
| task_c55c prefetch + offline read | M |
| task_ebf1 outbox + sync | M |
| task_43fc docs | S |

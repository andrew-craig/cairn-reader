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

## Goal

Let mobile users read their saved articles without a network connection. Saved
content is stored on device; edits made offline sync when the connection returns
(product_requirements.md line 31).

## Scope decisions (confirmed 2026-09-06)

- **Mobile only.** Web offline (PWA/service worker) stays a documented non-goal.
- **Auto-cache, bounded.** On each sync, cache the Read list plus proactively
  fetch and store the full `cleaned_html` for the **50 most-recent saved
  articles**, and cache the **20 most-recent Explore recommendations** with their
  bodies (Explore recommendation payloads already include `content` inline, so no
  extra detail fetch). No per-article "download" button.
- **Text-only offline.** Article HTML/text is cached; remote `<img>` simply fail
  to load with no connection. Image caching is out of scope (relates to
  feature_9d64).
- **Offline edits queue.** A small write outbox replays mark-read / favorite /
  archive / scroll-position PATCH+DELETE on reconnect.
- **Storage backend:** `expo-sqlite` (confirmed).

## Prerequisite / coordination

- **task_47c1 (offline-clears-tokens)** is a hard blocker. Today, opening the app
  offline within 5 min of access-token expiry runs `doRefreshAccessToken`, the
  `fetch` throws, the `catch` calls `clearTokens()` → full logout + tokens wiped
  (`apps/mobile/src/services/auth.ts:358-433`). `AuthContext.checkAuthStatus`
  hits this on every cold start. `ExploreScreen.loadExploreArticles` similarly
  calls `logout()` on any error message matching auth substrings
  (`ExploreScreen.tsx:201-210`). Offline mode is pointless until refresh
  distinguishes "server rejected" from "couldn't reach server". Piece 1 below is
  the minimal fix; if task_47c1's full `apps/shared` auth consolidation lands
  first, piece 1 collapses into using that.
- **task_179f (dual caches)** is folded in: `storage.ts` today has two
  disconnected AsyncStorage stores for the same articles (`@cairnreader:articles`
  write-only + `@cairnreader:read_list_cache`). The new offline store replaces
  both as the single source of truth. The archive hard-delete-vs-status part of
  task_179f is coordinated separately unless it falls out naturally in piece 5.
- **No backend changes.** There is no batch-content or delta/"since" endpoint;
  the sync walks the cursor-paginated list and per-item compares `content_hash`
  (present in list summaries) to decide which bodies to (re)download. ~50
  sequential `getContentById` calls, throttled, is acceptable. A future backend
  batch/delta endpoint can be tracked as its own task.

## Design

New offline layer under `apps/mobile/src/` (shared types stay in `apps/shared`):

- **Connectivity** — add `@react-native-community/netinfo` (Expo-compatible).
  `useNetworkStatus()` hook + a small module exposing `isOnline()` and a
  reconnect event.
- **OfflineStore** — backed by `expo-sqlite` (new dep; chosen over AsyncStorage
  because the feature stores many ~100 KB article bodies and needs per-row writes
  + a transactional outbox — AsyncStorage does whole-value JSON rewrites and hits
  Android's ~6 MB limit). Tables: `articles` (summary fields + `cleaned_html` +
  `content_hash` + `source` (`read`|`explore`) + `cached_at`), `outbox`
  (`content_id`, `op`, `payload`, `created_at`, `attempts`). One-time import from
  the existing AsyncStorage caches, then the old `storage.ts` cache code is
  deleted (no back-compat).
- **OfflineSyncService** — runs on app foreground, on reconnect, and on
  pull-to-refresh: (1) flush outbox in order; (2) walk Read list via cursor
  pagination, keep the 50 most-recent; (3) diff by `content_id` + `content_hash`;
  (4) fetch `getContentById` for new/changed Read items; (5) store the 20
  most-recent Explore recommendations with their inline `content`; (6) evict
  local articles outside the 50 / 20 windows (LRU by `added_at`).
- **Read path** — `ReadScreen` + `BookmarksScreen` + `ExploreScreen` render from
  OfflineStore then revalidate (extends the existing SWR pattern; Bookmarks gains
  caching it lacks today). `ReadArticleDetailScreen` reads `cleaned_html` from
  OfflineStore before calling `getContentById`; offline + uncached shows a clear
  "Not available offline" state instead of a blank body. Detail screens resolve
  the article by id from the store, not only from navigation params.
- **Write path** — `ReadService.updateUserContent` / `deleteUserContent`: when
  offline or on network failure, enqueue to outbox + apply optimistically to
  OfflineStore; when online, write through and update the store. Replaces today's
  fire-and-forget `.catch(log)` that silently loses offline edits.
- **UI** — offline banner when `!isOnline`; Settings "Offline reading" section
  (cached article count, "Clear offline data"); update the stale storage section
  of `apps/mobile/CLAUDE.md`.

## Plan (multi-PR, "piece N/7", each independently mergeable)

1. **Auth: keep tokens when the server is unreachable.** → verify: unit test —
   offline refresh keeps tokens + throws a retryable error; 401 refresh still
   clears. Fix `ExploreScreen` logout-on-substring path. (Absorbed by task_47c1
   if that lands first.)
2. **Connectivity + storage foundation.** Add `@react-native-community/netinfo`
   + `expo-sqlite`. `useNetworkStatus`. `OfflineStore` schema + CRUD + one-time
   import from AsyncStorage caches; delete old `storage.ts` caches (folds in
   task_179f cache consolidation). → verify: OfflineStore unit tests; app boots;
   Read list still renders from the new store.
3. **Sync engine.** `OfflineSyncService`: Read-list walk keeping 50 most-recent +
   body prefetch + `content_hash` diff; Explore top-20 with inline content;
   eviction. Wired into ReadScreen / BookmarksScreen / ExploreScreen focus +
   pull-to-refresh + app foreground + reconnect. → verify: sync unit tests with
   mocked services; airplane-mode launch shows cached Read + Explore lists.
4. **Offline reader.** `ReadArticleDetailScreen` / `ExploreArticleDetailScreen`
   read body from OfflineStore; "Not available offline" state; by-id resolution.
   → verify: cached article opens offline; uncached shows the message; existing
   detail tests pass.
5. **Write outbox.** Outbox enqueue on offline/failed mutation + optimistic local
   apply + ordered replay on reconnect/foreground; route `updateUserContent` /
   `deleteUserContent` through it. → verify: offline favorite/archive/scroll
   persist locally and sync on reconnect; ordering + 4xx-drop tests.
6. **UI polish + settings.** Offline banner, Settings "Offline reading" section
   (cached count + clear data), `apps/mobile/CLAUDE.md` storage section refresh.
   → verify: snapshot/interaction tests; manual airplane-mode walkthrough.
7. **QA pass.** End-to-end airplane-mode walkthrough across cold start, list,
   reader, offline edits, reconnect sync; coverage check on new modules.

## Subtasks

| Piece | Task | Blocked by |
|---|---|---|
| 1/7 | task_3d59 — keep auth tokens when server is unreachable | — |
| 2/7 | task_1f23 — connectivity + expo-sqlite OfflineStore foundation | task_3d59 |
| 3/7 | task_c73b — OfflineSyncService (bounded Read + Explore sync) | task_1f23, task_3d59 |
| 4/7 | task_ccf3 — read cached article bodies in the reader screens | task_c73b |
| 5/7 | task_ec80 — write outbox (queue offline edits, replay on reconnect) | task_c73b |
| 6/7 | task_44ca — offline banner, Settings section, docs | task_ccf3, task_ec80 |
| 7/7 | task_d091 — end-to-end airplane-mode QA pass | task_44ca |

Related: task_47c1 (FE auth-layer consolidation), task_179f (mobile archive
semantics + dual caches) — both partly folded into pieces 1/2/5.

## Review

_(to be filled in as pieces land)_

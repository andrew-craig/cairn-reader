---
id: task_1f23
title: Offline mobile: connectivity detection + expo-sqlite OfflineStore foundation (piece 2/7)
type: task
status: open
priority: 3
labels: [mobile,offline]
blocked_by: [task_3d59]
parent: feature_90a5
remote_task_url: null
created_at: 2026-09-05T23:13:31Z
updated_at: 2026-09-05T23:14:18Z
---
Foundation for offline reading (feature_90a5).

Scope:
- Add dependencies: @react-native-community/netinfo and expo-sqlite (both Expo-managed compatible; expo ~54 / RN 0.81).
- useNetworkStatus() hook + a small module exposing isOnline() and a 'reconnected' event. First network-detection in the app — keep it minimal, no new global state library (Context/hook only, per ENGINEERING_PRINCIPLES).
- OfflineStore service (static-class pattern like other apps/mobile/src/services/*), backed by expo-sqlite. Tables:
  - articles: content_id (pk), source ('read'|'explore'), summary fields (title, author, url, image_url, description, status, is_favorite, scroll_position, added_at), cleaned_html (nullable), content_hash, cached_at.
  - outbox: id, content_id, op ('patch'|'delete'), payload (json), created_at, attempts. (Table created here; consumed in piece 5.)
- One-time import from the existing AsyncStorage caches (@cairnreader:articles, @cairnreader:read_list_cache, @cairnreader:explore_cache), then delete the old StorageService cache code in apps/mobile/src/services/storage.ts. This folds in task_179f's dual-cache consolidation — one source of truth. No backward-compat shim (CLAUDE.md section 4).
- Repoint ReadScreen / BookmarksScreen / ExploreScreen list priming at OfflineStore (behaviour-preserving; sync logic comes in piece 3). Route list changes through useCursorArticleList's onResetLoaded hook.

Verify: OfflineStore unit tests (CRUD, eviction helper, import path) with expo-sqlite; app boots; Read/Bookmarks/Explore lists still render from the new store; npm test + type-check + lint green in apps/mobile.

Blocked by: task_3d59 (piece 1).

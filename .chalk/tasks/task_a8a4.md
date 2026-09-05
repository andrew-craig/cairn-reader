---
id: task_a8a4
title: Mobile: single SQLite-backed local article store (replaces AsyncStorage article caches)
type: task
status: open
priority: 2
labels: [mobile,offline]
blocked_by: []
parent: feature_90a5
remote_task_url: null
created_at: 2026-09-05T23:36:10Z
updated_at: 2026-09-05T23:36:10Z
---
Phase 1 of feature_90a5. Introduce one local store (expo-sqlite) for read-list articles: metadata, user state (status/is_favorite/scroll_position/updated_at) and a nullable body column. ReadScreen, BookmarksScreen (query is_favorite from the store; it has no cache today) and ReadArticleDetailScreen read from the store first and refresh from the network. No import from the old AsyncStorage caches: the store repopulates on first sync. Removes ARTICLES_KEY and READ_LIST_CACHE_KEY from storage.ts (EXPLORE_CACHE_KEY stays). Clears the store on logout. Supersedes the dual-cache part of task_179f.

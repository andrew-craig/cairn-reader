---
id: task_c73b
title: Offline mobile: OfflineSyncService — bounded Read + Explore sync (piece 3/7)
type: task
status: open
priority: 3
labels: [mobile,offline]
blocked_by: [task_1f23,task_3d59]
parent: feature_90a5
remote_task_url: null
created_at: 2026-09-05T23:13:41Z
updated_at: 2026-09-05T23:14:18Z
---
Sync engine for offline reading (feature_90a5).

Scope — OfflineSyncService, triggered on app foreground, on reconnect (useNetworkStatus), and on pull-to-refresh:
1. Flush the write outbox first (implemented in piece 5; call site stubbed here).
2. Read list: walk ReadService.listUserContents by cursor, keep the 50 most-recent by added_at. Diff local vs remote by content_id + content.content_hash (present in list summaries). For new/changed items in the window, call ReadService.getContentById to fetch and store cleaned_html. Throttle the sequential detail calls (no batch endpoint exists).
3. Explore: store the 20 most-recent ExploreService.getRecommendations entries — recommendation payloads already include full 'content' inline, so no extra fetch.
4. Eviction: drop articles outside the 50 (read) / 20 (explore) windows, LRU by added_at.
5. Surface sync state to the list screens (extends the existing stale-while-revalidate 'Showing cached data' banner pattern in ArticleListScreen).

Constraints: no backend changes; ~50 sequential detail requests throttled is acceptable. Keep sync logic out of ExploreScreen.tsx (already 485 lines / fragile) — put it in the service.

Verify: OfflineSyncService unit tests with mocked ReadService/ExploreService (window bounding, hash-diff skip, eviction, throttle); airplane-mode cold launch shows the last-synced Read and Explore lists; npm test + type-check + lint green.

Blocked by: task_1f23 (piece 2), task_3d59 (piece 1).

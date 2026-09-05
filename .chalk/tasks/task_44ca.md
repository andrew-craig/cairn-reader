---
id: task_44ca
title: Offline mobile: offline banner, Settings section, docs (piece 6/7)
type: task
status: open
priority: 3
labels: [mobile,offline]
blocked_by: [task_ccf3,task_ec80]
parent: feature_90a5
remote_task_url: null
created_at: 2026-09-05T23:14:06Z
updated_at: 2026-09-05T23:14:19Z
---
UI polish + docs for feature_90a5.

Scope:
- Offline indicator: a lightweight banner (reuse the existing stale-banner style in ArticleListScreen) shown when useNetworkStatus reports offline, across the Read/Bookmarks/Explore/reader screens.
- Settings: add an 'Offline reading' section showing the cached article count (Read + Explore) and a 'Clear offline data' action that wipes the OfflineStore articles table (keeps the outbox). Fixed cache bounds (50/20) — no user-facing tuning.
- Docs: refresh the stale storage section of apps/mobile/CLAUDE.md (currently documents only the removed @cairnreader:articles cache) to describe OfflineStore, the sync triggers, the outbox, and the 50/20 bounds. Note offline scope in the app's feature list.

Verify: snapshot / interaction tests for the banner and Settings section; manual airplane-mode walkthrough; type-check + lint green.

Blocked by: task_ccf3 (piece 4), task_ec80 (piece 5).

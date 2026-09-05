---
id: task_de93
title: Mobile: end-to-end airplane-mode QA pass for offline reading
type: task
status: open
priority: 3
labels: [mobile,offline,qa]
blocked_by: [task_ebf1]
parent: feature_90a5
remote_task_url: null
created_at: 2026-09-05T23:46:28Z
updated_at: 2026-09-05T23:46:28Z
---
Final QA for feature_90a5 once phases 1-4 have landed. Walkthrough on device or simulator in airplane mode: cold start offline near token expiry stays logged in; Read and Bookmarks lists show last-synced content; a synced article opens with its full body (images broken, expected); an unsynced article shows the 'Not available offline' state; offline mark-read, favorite, archive and scroll replay on reconnect with backend state matching and no duplicates; offline banner appears and clears with connectivity; online behaviour of Read/Bookmarks/reader unchanged. Also: coverage check on the store, prefetch and outbox modules; fill in the Review section of feature_90a5.md; capture lessons in LEARNINGS.md.

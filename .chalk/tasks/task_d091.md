---
id: task_d091
title: Offline mobile: end-to-end airplane-mode QA pass (piece 7/7)
type: task
status: open
priority: 3
labels: [mobile,offline]
blocked_by: [task_44ca]
parent: feature_90a5
remote_task_url: null
created_at: 2026-09-05T23:14:14Z
updated_at: 2026-09-05T23:14:19Z
---
Final QA for feature_90a5, once pieces 1-6 have landed.

Walkthrough (device or simulator, airplane mode):
- Cold start offline near/after token expiry -> stays logged in, no token wipe (piece 1).
- Read list + Bookmarks + Explore show last-synced content offline; counts match the 50/20 bounds.
- Open a cached article offline -> full body renders (images broken, expected). Open an uncached article -> 'Not available offline' state.
- Offline: mark read, favorite, archive, scroll a few articles. Reconnect -> outbox replays, backend reflects the changes, no duplicates, order preserved.
- Offline banner appears/clears with connectivity.
- Settings 'Clear offline data' empties the cache; next sync repopulates.
- Regression: online behaviour of Read/Bookmarks/Explore/reader unchanged.

Also: coverage check on OfflineStore / OfflineSyncService / outbox (services target 90%+ per ENGINEERING_PRINCIPLES); update LEARNINGS.md (create it — does not exist yet) with anything found during the feature; fill in the Review section of feature_90a5.md.

Blocked by: task_44ca (piece 6).

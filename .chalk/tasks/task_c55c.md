---
id: task_c55c
title: Mobile: prefetch article bodies and read from the local store when offline
type: task
status: open
priority: 2
labels: [mobile,offline]
blocked_by: [task_a8a4,task_c87c]
parent: feature_90a5
remote_task_url: null
created_at: 2026-09-05T23:36:10Z
updated_at: 2026-09-06T03:13:44Z
---
Phase 3 of feature_90a5 (the user-facing offline reading). After each successful list sync, download cleaned_html for unread/reading articles that lack a body (bounded concurrency, newest first, capped at the 100 most recent Read-list articles) into the store. Explore article content is never synced. Diff by content.content_hash from the list page (present on every list item) so unchanged bodies are never re-downloaded. Detail screen resolves the article by id from the store (not only route params), renders the stored body immediately, and refreshes online when the hash changed. When offline with no stored body, show a clear 'Not available offline' state instead of a blank body. Prefetch runs after list sync on focus, pull-to-refresh, app foreground and reconnect; keep it in a service, not in the screens. Evict bodies on archive/delete and beyond the cap. Images remain remote (out of scope).

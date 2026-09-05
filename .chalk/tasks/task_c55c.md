---
id: task_c55c
title: Mobile: prefetch article bodies and read from the local store when offline
type: task
status: open
priority: 2
labels: [mobile,offline]
blocked_by: [task_a8a4,task_c87c,task_47c1]
parent: feature_90a5
remote_task_url: null
created_at: 2026-09-05T23:36:10Z
updated_at: 2026-09-05T23:36:10Z
---
Phase 3 of feature_90a5 (the user-facing offline reading). After each successful list sync, download cleaned_html for unread/reading articles that lack a body (bounded concurrency, newest first, capped at the 100 most recent Read-list articles) into the store. Explore article content is never synced. Detail screen renders the stored body immediately and refreshes online when content.updated_at changed. Evict bodies on archive/delete and beyond the cap. Images remain remote (out of scope).

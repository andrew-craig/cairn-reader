---
id: task_ebf1
title: Mobile: offline mutation outbox and reconnect sync
type: task
status: open
priority: 2
labels: [mobile,offline]
blocked_by: [task_c55c]
parent: feature_90a5
remote_task_url: null
created_at: 2026-09-05T23:36:10Z
updated_at: 2026-09-05T23:36:10Z
---
Phase 4 of feature_90a5. Status, favorite, scroll_position and archive (DELETE) writes go store-first and enqueue an outbox row when the request fails with a NetworkError. A sync worker drains the outbox on reconnect, app foreground and pull-to-refresh, in created_at order, before the body prefetch runs. 2xx deletes the row; 404 on a replayed DELETE counts as success; definitive 4xx (except 401) drops the row and logs; network/5xx/401 keeps the row, bumps attempts and halts the batch to preserve order. Coalesce scroll_position (and repeated PATCHes) per article so the queue stays bounded. Verify with unit tests for enqueue, ordered replay, 4xx drop, 5xx halt and coalescing. Fixes the swallowed archive error from task_179f. Add-URL stays online-only.

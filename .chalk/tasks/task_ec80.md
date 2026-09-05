---
id: task_ec80
title: Offline mobile: write outbox — queue offline edits, replay on reconnect (piece 5/7)
type: task
status: open
priority: 3
labels: [mobile,offline]
blocked_by: [task_c73b]
parent: feature_90a5
remote_task_url: null
created_at: 2026-09-05T23:13:59Z
updated_at: 2026-09-05T23:14:18Z
---
Offline write queue for feature_90a5. Implements product_requirements.md line 31 ('changes are synced when the connection is restored') and fixes today's fire-and-forget mutations that silently lose offline edits.

Scope:
- Route ReadService.updateUserContent (status / is_favorite / scroll_position PATCH) and ReadService.deleteUserContent through OfflineStore:
  - Always apply the change optimistically to the local articles row.
  - If online: write through to the backend; on success clear any matching queued op; on network failure fall through to enqueue.
  - If offline: enqueue an outbox row (content_id, op, payload, created_at, attempts=0).
- Outbox replay (called at the start of OfflineSyncService, piece 3): process rows in created_at order. On 2xx delete the row. On definitive 4xx (except 401) drop the row and log. On network/5xx/401 keep the row, bump attempts, stop the batch (preserve ordering). Coalesce multiple PATCHes to the same content_id.
- Scroll-position writes are frequent — coalesce / last-write-wins per content_id, don't grow the queue unbounded.
- Optional (coordinate with task_179f): make offline archive use status='archived' via the outbox rather than hard delete. Keep to the delete path if that expands scope too far.

Verify: unit tests — offline favorite/archive/scroll persist locally and appear in the outbox; reconnect replays them in order; 4xx drops the row; 5xx retains and halts; PATCH coalescing. npm test + type-check + lint green.

Blocked by: task_c73b (piece 3).

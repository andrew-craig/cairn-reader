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

## Note from task_a8a4 review (tech lead, 2026-09-06)

Constraint discovered while reviewing the store implementation, recorded here
because it lands on this task rather than that one.

`ArticleStore.upsertMany` overwrites `is_read`, `is_favorite`, `scroll_fraction`
and `read_at` from the server's list page — see design decision 2 in task_a8a4
(sync is upsert-only, server rows win). Today that is harmless, because an offline
user-state edit is already lost anyway.

Once this outbox exists, it stops being harmless. The sequence: user favourites an
article offline → store row updated, `updateUserContent` fails with a
`NetworkError`, outbox row enqueued → connectivity returns → a list sync happens
to run before the outbox drains → `upsertMany` overwrites `is_favorite` back to
the server's stale value → the UI silently reverts the user's action while the
write is still pending in the queue.

So this task must reconcile the two. The options, none of them free:
- Drain the outbox before any list sync writes to the store (the description
  already orders the drain before the *body prefetch*; this extends that ordering
  to the list upsert as well, and needs the sync path to actually enforce it).
- Have `upsertMany` skip user-state columns for rows with a pending outbox entry,
  which means the store needs to know about the outbox — a coupling worth thinking
  about before adopting.
- Have the outbox re-apply its pending writes to the store after any sync.

Whichever is chosen, add a test for the interleaving above specifically: pending
write + list sync arriving first + assert the user's value survives. It will not
be caught by the enqueue/replay/drop/halt/coalesce tests already listed in the
description, because those never run a list sync concurrently.

---
id: task_ebf1
title: Mobile: offline mutation outbox and reconnect sync
type: task
status: in_progress
priority: 2
labels: [mobile,offline]
blocked_by: []
parent: feature_90a5
remote_task_url: null
created_at: 2026-09-05T23:36:10Z
updated_at: 2026-09-11T22:52:51Z
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

## Inherited from task_c55c (tech lead, 2026-09-08)
Phase 3 deferred its app-foreground and reconnect prefetch triggers to this task, on the
grounds that the outbox needs the same `AppState`/network-change listener and building it
twice is waste. When that listener lands here, wire **both** consumers to it:
`ArticlePrefetchService.run()` as well as the outbox drain.

Two concrete symptoms that trigger should fix, both live on main today:
1. A user who regains connectivity does not prefetch until they open or pull-to-refresh
   the Read tab (30s TTL).
2. `ReadArticleDetailScreen` shows "Not available offline"; if connectivity returns while
   that screen is open, the render guard (`!article.content && isOffline`) stops matching
   and the screen falls through to a **blank** `ArticleContent` — nothing re-triggers the
   fetch, since the content-loading effect keys off `initialArticle.id` and reads
   connectivity through a ref. Backing out and re-opening recovers. Fix it via the
   reconnect trigger, not a screen-local listener.

## Scope clarification (tech lead, 2026-09-11)
Reviewed the phase 1-3 code on main before assigning. Decisions below are made, not
open — push back with a reason if one is wrong, don't silently pick differently.

### Split
The inherited app-foreground/reconnect trigger is now **task_06e5**, which blocks this
task. Do not build a listener here: consume the one task_06e5 lands, adding the outbox
drain as the first consumer ahead of `ArticlePrefetchService.run()`.

### 1. The `upsertMany` clobber — decision: guard in SQL, not ordering
Of the three options in the note above, take the second: `upsertMany` skips the
user-state columns (`is_read`, `is_favorite`, `scroll_fraction`, `read_at`) for any row
with a pending outbox entry. Ordering the drain before the list sync is *not* sufficient
on its own — a list request already in flight can resolve mid-drain, and a drain that
halts on a 5xx leaves rows queued while syncs keep running. The guard is correct under
every interleaving; the ordering is a freshness nicety on top.

The coupling concern in the note is answered by putting the outbox table in the **same**
`cairnreader.db`, so the guard is an `EXISTS (SELECT 1 FROM outbox WHERE ...)` inside the
existing upsert statement — one table, one module, no cross-layer dependency. Do not
introduce a second database or have the store call into a service.

Note `body`/`content_hash` handling in `UPSERT_SQL` must keep working exactly as it does
today; the guard applies only to the four user-state columns.

### 2. Schema
New `outbox` table in `cairnreader.db` as migration step 3 via `PRAGMA user_version`
(step 2 is task_c55c's `content_hash` — follow that pattern exactly). Cleared alongside
articles in `ArticleStore.clear()`, which `AuthContext` already calls on logout — a
queued write must never replay against a different account.

### 3. Coalescing — one row per (article_id, field)
Key the table on `(article_id, field)` where field is one of `status`, `is_favorite`,
`scroll_position`, `delete`. Enqueue is `ON CONFLICT(article_id, field) DO UPDATE SET
payload = excluded.payload`, **leaving `created_at` untouched** so a coalesced write keeps
its place in the queue instead of jumping to the back. Enqueuing a `delete` for an article
removes that article's other rows — the DELETE supersedes them and replaying a PATCH
against a deleted row is a guaranteed 404.

### 4. Call sites — add a facade, don't repeat the pattern six times
Six call sites need the same three steps (write the store, try the network, enqueue on
`NetworkError`): `markCompleted`, the `status: 'reading'` effect, the throttled scroll
save, the unmount scroll flush, `handleToggleFavorite` and `handleArchive`, all in
`ReadArticleDetailScreen.tsx`. Route them through one module rather than inlining the
logic six times. This also removes the `.catch(console.error)` swallowing that
task_179f left behind, including the archive error.

Only `NetworkError` enqueues. A definitive 4xx at write time is a real rejection and must
surface as it does today — do not queue it.

### 5. Drain semantics (restating the description's rules as the acceptance list)
Ordered by `created_at` ascending. 2xx deletes the row. 404 on a replayed `delete`
counts as success and deletes the row. Definitive 4xx other than 401 drops the row and
logs. `NetworkError`/5xx/401 keeps the row, bumps `attempts` and **halts the whole batch**
(global halt, not per-article) so ordering is preserved. `fetchWithAuth` already refreshes
on 401, so a 401 reaching the drain means refresh genuinely failed — halting is correct.

`attempts` is bookkeeping for diagnosis in v1: bump it, do not add backoff or a
drop-after-N rule unless you can point at a concrete need.

### 6. Out of scope
Add-URL stays online-only (`AddArticleScreen`). No backend change. No conflict resolution
beyond last-write-wins — decision 6 on feature_90a5 stands.

### Tests
Beyond enqueue / ordered replay / 4xx drop / 5xx halt / coalesce, the interleaving test
the note above calls for is mandatory: pending outbox write + a list sync arriving first
+ assert the user's value survives in the store. Also cover 404-on-replayed-delete and
delete-supersedes-pending-patches.

## Pre-assignment review (tech lead, 2026-09-11, second pass)
Re-read the phase 1-3 code on main after task_06e5 landed. Four gaps in the scope above
that would have blocked or silently mis-implemented the drain. These are decisions, not
options — push back with a reason if one is wrong.

### A. HTTP status is not available to the drain today — plumb it
`ReadService.updateUserContent` (read.ts:176-178) and `deleteUserContent` (read.ts:207-210)
collapse every non-2xx into `new Error(message)`. The status is discarded, so
"404 on a replayed delete is success", "definitive 4xx drops the row" and "5xx halts"
are all unimplementable as written.
Decision: add `HttpError extends Error { readonly status: number }` to
`apps/mobile/src/utils/errors.ts`, export it from `utils/index.ts`, and throw it from
those two methods on `!response.ok`. **Keep the existing message text byte-for-byte** —
`utils/retry.ts:32-37` classifies retryability by message substring, and
`ExploreArticleDetailScreen.tsx:58` surfaces `error.message`. This is an error-type
change only; no status/credential semantics change anywhere.

### B. A pending `delete` must stop the list sync resurrecting the article
Not covered by the user-state-column guard in decision 1 above, and a visible bug without
it: archive offline -> `ArticleStore.remove()` drops the row and a `delete` outbox row is
queued -> the next list sync still sees the article server-side -> `upsertMany` re-INSERTs
it and the archived article reappears in the Read list until the drain succeeds.
Decision: `upsertMany` must skip entirely (neither insert nor update) any row with a
pending `delete` outbox entry. Test it.

### C. Ordering needs a tiebreaker
`created_at` in milliseconds collides — `markCompleted` and the unmount scroll flush can
enqueue in the same tick. Order the drain by `created_at ASC, rowid ASC` so replay order
is total and deterministic.

### D. Guard granularity — per-article, not per-field
The `EXISTS` guard in decision 1 keys on `article_id` alone, so any pending row for an
article freezes all four user-state columns against the server, not just the one field
queued. Accepted: simpler SQL, and the effect is transient (it lasts until the drain).
Chosen deliberately — do not "fix" it to a per-field CASE ladder without raising it first.

### E. Naming trap
The outbox `field` values (`status`, `is_favorite`, `scroll_position`, `delete`) are the
*server's* PATCH field names. The store's scroll column is `scroll_fraction`, and
`articles.scroll_position` is a separate legacy column. `status` maps to the store's
`is_read` + `read_at`. Map explicitly at the boundary; do not assume the names line up.

### F. Module layout
Preferred: extract `getDb()` and the migration ladder into `apps/mobile/src/services/db.ts`
unchanged, so `articleStore.ts` and a new `outbox.ts` share one database and one migration
ladder without either exporting its internals. Alternative layouts are fine if simpler —
state the reason. Non-negotiable: one `cairnreader.db`, one migration ladder in one place,
and `ArticleStore.clear()` clears the outbox too (AuthContext calls it on logout; a queued
write must never replay against a different account).

### G. Third trigger — pull-to-refresh
`ReadScreen.tsx:44` calls `ArticlePrefetchService.run()` directly after `upsertMany`.
Route it through `SyncTrigger.run()` instead, so the drain gets the pull-to-refresh trigger
the description requires and the fixed consumer order (drain, then prefetch) is honoured in
every path rather than only on reconnect/foreground.

---
id: task_387d
title: [iOS native 07] Outbox as the only write path, ArticleMutations
type: task
status: closed
priority: 1
labels: [ios,offline]
blocked_by: []
parent: epic_bc48
remote_task_url: null
created_at: 2026-11-14T02:35:53Z
updated_at: 2026-11-15T03:40:23Z
---
## Goal
Reader and list edits are written locally and queued in one step, and the outbox is the only thing that sends them (decision 9). Same coalescing and failure classification as today.

## Source
`apps/mobile/src/services/outbox.ts`, `articleMutations.ts`; tests `outbox.test.ts`, `articleMutations.test.ts`. The classification rules port unchanged; the write path does not.

## Plan
- [x] `ReadService` takes the user id at init and `UserSession` owns the instance (decision 10). Remove the per-call `auth.user` lookup.
- [x] `ArticleMutations` (owned by `UserSession`): `markCompleted`, `markReading`, `saveScrollPosition`, `setFavorite`, `archive`, `saveToReads`. Each is one database transaction, the store write plus the outbox upsert, followed by a drain request that the caller does not wait for. They throw only for a database failure, never for the backend's answer.
- [x] Outbox rows keyed on `(article_id, field)`, `field` in `status` | `is_favorite` | `scroll_position` | `list`; a repeat replaces the payload and keeps `created_at`.
- [x] `saveToReads` upserts the Feed item as a Reads item (`addedAt` now, body kept only if present) and queues the `list` row.
- [x] `Outbox.drain`: replay in `created_at` order (rowid breaks ties), stop at the first row that cannot succeed. 2xx removes the row. A definitive 4xx other than 401 drops it, and a dropped `list` row also deletes the article from the store. `NetworkError`, 401, 5xx, `AuthError.sessionExpired` and unrecognised errors keep the row. No backoff, no drop-after-N.
- [x] Remove a sent row only if its payload is still the one that was sent, so an edit coalesced into the row during the request is sent by the next pass.
- [x] Drain is single-flight and runs again if one was requested while it was running. Requests are ignored while `Connectivity` reports offline (wired in task 08; until then the drain simply halts on `NetworkError`).
- [x] `Outbox.rejections`: an `AsyncStream` of dropped rows `(articleID, field)`, one consumer (the signed-in root, task 09).
- [x] `Outbox.isRetryable` shared by the drain and anything else that classifies errors.

## Verify
- `outbox.test.ts` ported case for case, minus `attempts` and `pendingSaveIds` (replaced by the store's observations).
- `articleMutations.test.ts` re-expressed: each mutation leaves the store change and the outbox row together or neither (inject a failing write); nothing is sent outside the drain.
- New tests: favourite offline then unfavourite online ends with `false` on the server stub; a payload replaced mid-request is not lost; a dropped `list` row deletes the article and yields a rejection; a dropped `status: archived` row yields a rejection.

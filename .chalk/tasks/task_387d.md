---
id: task_387d
title: [iOS native 07] Outbox and ArticleMutations: offline write queue
type: task
status: open
priority: 1
labels: [ios,offline]
blocked_by: [task_ec50,task_c194]
parent: epic_bc48
remote_task_url: null
created_at: 2026-11-14T02:35:53Z
updated_at: 2026-11-14T02:35:53Z
---
## Goal
Reader and list edits succeed offline and replay on reconnect, with the same coalescing and failure rules as today.

## Source
`apps/mobile/src/services/outbox.ts`, `articleMutations.ts`; tests `outbox.test.ts`, `articleMutations.test.ts`.

## Plan
- [ ] `Outbox.enqueue` keyed on `(article_id, field)`, `field` in `status` | `is_favorite` | `scroll_position` | `list`; repeats coalesce into one row.
- [ ] `Outbox.drain`: replay in `created_at` order (rowid breaks ties), stop at the first row that cannot succeed. 2xx removes the row. A definitive 4xx other than 401 drops it, and a dropped `list` row also deletes the article from `ArticleStore`. `NetworkError`, 401, 5xx and unrecognised errors keep the row and bump `attempts`. No backoff, no drop-after-N.
- [ ] `Outbox.pendingSaveIds` (never throws; empty on read failure) and `isRetryable`.
- [ ] `ArticleMutations`: `markCompleted`, `markReading`, `saveScrollPosition`, `setFavorite`, `archive`, `saveToReads`. Store first, then backend, enqueue on a retryable error, rethrow a definitive 4xx. `saveToReads` upserts the Feed item as a Reads item (`addedAt` now, body kept only if present) and removes it again on a definitive 4xx.

## Verify
- Both test files ported case for case.

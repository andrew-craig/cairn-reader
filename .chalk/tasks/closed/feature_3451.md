---
id: feature_3451
title: Mobile: queue Save to Reads offline with a pending placeholder in Reads
type: feature
status: closed
priority: 2
labels: [mobile,offline]
blocked_by: []
parent: null
remote_task_url: null
created_at: 2026-11-01T22:03:26Z
updated_at: 2026-11-01T22:08:36Z
---
Users can Save to Reads (Feed swipe, reader button) while offline. The save is queued in the Outbox as a `list` row, the article is written to ArticleStore as a Reads item, and Reads shows it with a "Waiting to save" indicator until the outbox drains.

## Plan
- [x] Outbox: add `list` field; `pendingSaveIds()`; on a dropped (definitive 4xx) `list` row also delete the optimistic ArticleStore row
- [x] ArticleStore: `listPendingSaves()` (stored articles with a pending `list` outbox row, newest save first)
- [x] ArticleMutations.saveToReads(article): store-first (addedAt = now, matching the server's bump), then PATCH via withOutboxOnRetryableError; definitive 4xx → remove from store + rethrow
- [x] FeedScreen: swipe uses saveToReads; filter pending-save ids out of every fetched page (no resurrection by refetch / FeedCache)
- [x] Reader: Save to Reads uses saveToReads
- [x] ReadsScreen: merge queued saves (snapshot taken before each reset fetch) above the server page; pending flag → ArticleRow indicator; refresh indicator after SyncTrigger
- [x] ArticleRow: `pending` prop → "Waiting to save — will sync when online" label (theme colours)
- [x] Tests: mutations, outbox, store, Feed, Reads, reader, ArticleRow
- [x] AGENTS.md updated
- [x] type-check, eslint, jest pass; commit on feat/feed-swipe-to-reads

## Review
- One code path online/offline: `ArticleMutations.saveToReads(article)` upserts the Feed item into ArticleStore as a Reads item (`addedAt` = now, matching the server's re-stamp on move), then `moveToReads` via `withOutboxOnRetryableError(id, 'list', { list: 'reads' })`. A definitive 4xx removes the store row and rethrows. No schema migration needed: `outbox.field` is unconstrained TEXT.
- Outbox: `list` field; `pendingSaveIds()`; a dropped `list` row (definitive 4xx during drain) also deletes the article row so a rejected save doesn't linger as a ghost in Reads.
- Feed: swipe and reader both use `saveToReads`; row leaves Feed + FeedCache on success or queue, alert + spring back only on a definitive rejection. `fetchPage` filters ids with a queued save, so neither a refetch nor the cache it writes can bring the item back.
- Reads: queued saves (`ArticleStore.listPendingSaves`) are merged above the server page (not during search) with an `ArticleRow` `pending` note in the theme `info` colour. The merge set is snapshotted before each first-page fetch, so an item that drains after the fetch stays visible until a page that includes it; the indicator is re-read after `SyncTrigger.run()`.
- Logout: ArticleStore.clear() already empties the outbox; ReadsScreen state unmounts with the authenticated tree. Nothing else needed.
- Verified: type-check, eslint on changed files, jest 43 suites / 339 tests pass.
- Open: an item whose queued save is rejected during a drain stays listed (without indicator) until the next Reads refresh; the reader keeps any body it already loaded when saving (Feed rows have none).

---
id: task_c194
title: [iOS native 06] Local database and ArticleStore (GRDB)
type: task
status: open
priority: 1
labels: [ios,offline]
blocked_by: []
parent: epic_bc48
remote_task_url: null
created_at: 2026-11-14T02:35:53Z
updated_at: 2026-11-14T10:08:43Z
---
## Goal
The on-device store of Reads articles that makes the Reads tab render instantly and work offline.

## Source
`apps/mobile/src/services/db.ts`, `articleStore.ts`; test `articleStore.test.ts` (569 lines: this is the specification).

## Plan
- [ ] Add GRDB via Swift Package Manager (decision 3), pinned to a major version.
- [ ] One `DatabaseQueue` in Application Support, new file name (the RN `cairnreader.db` is not opened; decision 4). Exclude nothing from backup beyond defaults.
- [ ] One `DatabaseMigrator` migration creating the final schema directly: `articles` (with `content_hash`) and `outbox` (`article_id`, `field`, `payload`, `created_at`, `attempts`, primary key `(article_id, field)`, implicit rowid). Do not replay the RN `user_version` ladder.
- [ ] `ArticleStore`: `upsertMany`, `listRecent(limit)`, `getById`, `saveBody`, `updateUserState`, `remove`, `clear` (also empties the outbox), `listPendingSaves`, and the prefetch helpers the RN store exposes (bodies missing, eviction outside the cap).
- [ ] Port `UPSERT_SQL` verbatim, including its three guards: skip the insert while an archive is queued; keep local `is_read` / `is_favorite` / `read_at` / `scroll_fraction` while any outbox row exists for the article; drop the cached body when `content_hash` changes.
- [ ] Read failures in `listRecent` resolve to an empty list rather than throwing (matches today; `ReadsScreen.storeFailure.test.tsx`).

## Verify
- `articleStore.test.ts` ported case for case against an in-memory database.

---
id: task_c194
title: [iOS native 06] Local database and ArticleStore (GRDB)
type: task
status: in_progress
priority: 1
labels: [ios,offline]
blocked_by: []
parent: epic_bc48
remote_task_url: null
created_at: 2026-11-14T02:35:53Z
updated_at: 2026-11-14T22:55:05Z
---
## Goal
The on-device store of Reads articles that makes the Reads tab render instantly and work offline.

## Source
`apps/mobile/src/services/db.ts`, `articleStore.ts`; test `articleStore.test.ts` (569 lines: this is the specification).

## Plan
- [x] Add GRDB via Swift Package Manager (decision 3), pinned to a major version.
- [x] One `DatabaseQueue` in Application Support, new file name (the RN `cairnreader.db` is not opened; decision 4). Exclude nothing from backup beyond defaults.
- [x] One `DatabaseMigrator` migration creating the final schema directly: `articles` (with `content_hash`) and `outbox` (`article_id`, `field`, `payload`, `created_at`, `attempts`, primary key `(article_id, field)`, implicit rowid). Do not replay the RN `user_version` ladder.
- [x] `ArticleStore`: `upsertMany`, `listRecent(limit)`, `getById`, `saveBody`, `updateUserState`, `remove`, `clear` (also empties the outbox), `listPendingSaves`, and the prefetch helpers the RN store exposes (bodies missing, eviction outside the cap).
- [x] Port `UPSERT_SQL` verbatim, including its three guards: skip the insert while an archive is queued; keep local `is_read` / `is_favorite` / `read_at` / `scroll_fraction` while any outbox row exists for the article; drop the cached body when `content_hash` changes.
- [x] Read failures in `listRecent` resolve to an empty list rather than throwing (matches today; `ReadsScreen.storeFailure.test.tsx`).

## Verify
- `articleStore.test.ts` ported case for case against an in-memory database.

## Review
- `Store/AppDatabase.swift`: GRDB 7.12 (up to next major), `store.sqlite` in Application Support, one `v1` migration creating `articles` and `outbox`. Opened lazily; a failed open is not cached (ports the RN `getDb` retry fix).
- `Store/ArticleStore.swift`: every RN store method, including `listFavorites` (Bookmarks) and the prefetch helpers. `UPSERT_SQL` ported with its three guards.
- Schema deviations from RN, following the Swift `Article` model: one `scroll_position` column (the 0–1 fraction, which the server also calls `scroll_position`) replaces RN's legacy `scroll_position` + `scroll_fraction` pair and is the guarded column; no `tags` column (the Swift model has none); dates stored as seconds since 1970 (REAL).
- Tests: `CairnReaderTests/ArticleStoreTests.swift` ports every case except the v1→v2 migration ones (no ladder, decision 4); adds a full-field round trip and a test that all four user-state columns freeze. Outbox rows are inserted directly until task_387d ports the outbox.

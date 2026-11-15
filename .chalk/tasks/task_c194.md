---
id: task_c194
title: [iOS native 06] Local database, ArticleStore as source of truth for Reads, UserSession (GRDB)
type: task
status: open
priority: 1
labels: [ios,offline]
blocked_by: [task_1162]
parent: epic_bc48
remote_task_url: null
created_at: 2026-11-14T02:35:53Z
updated_at: 2026-11-15T00:01:49Z
---
## Goal
The on-device store of Reads articles. It is the source of truth for the Reads tab (decision 8): the list is an observation of it, and network pages are only written into it. Also the start of `UserSession`, the object that owns everything belonging to the signed-in user (decision 10).

## Source
`apps/mobile/src/services/db.ts`, `articleStore.ts`; test `articleStore.test.ts` (569 lines). The upsert rules port unchanged; observation, reconciliation and the session scope are new.

## Plan
- [ ] Add GRDB via Swift Package Manager (decision 3), pinned to a major version.
- [ ] Database file in the App Group container from task_1162 (decision 11), new file name (the RN `cairnreader.db` is not opened; decision 4). Open it as GRDB's database-sharing guide prescribes for a database shared with an extension (`DatabasePool`, WAL, suspension handling), so the extension can be added without changing the file.
- [ ] If the database cannot be opened or migrated, delete it and recreate it once: it is a cache plus the outbox.
- [ ] One `DatabaseMigrator` migration creating the final schema directly: `articles` (with `content_hash`, one scroll column holding the fraction) and `outbox` (`article_id`, `field`, `payload`, `created_at`, primary key `(article_id, field)`, implicit rowid). No legacy `scroll_position` or `tags` columns and no `attempts` counter. Do not replay the RN `user_version` ladder.
- [ ] `ArticleStore` writes: `upsertMany`, `saveBody`, `updateUserState`, `remove`, and the prefetch helpers (bodies missing, eviction outside the cap).
- [ ] Port `UPSERT_SQL` with its three guards: skip the insert while an archive is queued; keep local `is_read` / `is_favorite` / `read_at` / scroll fraction while any outbox row exists for the article; drop the cached body when `content_hash` changes.
- [ ] `ArticleStore.syncPage(_:newerThan:hasMore:)`: in one transaction, upsert a Reads page and delete the stored rows strictly inside the page's `added_at` window that the page did not return and that have no outbox row. The window runs from the previous page's last `added_at` (none for the first page) down to this page's last `added_at` (none when `hasMore` is false).
- [ ] Observations (`ValueObservation`): `observeReads(window)` yielding articles newest first, each with a `pendingSave` flag (an outbox `list` row exists); `observeArticle(id)` for the reader; `observeStoredIDs()` for the Feed's exclusion set. Plain reads: `getById`.
- [ ] `UserSession`: created with the signed-in `User`, owns the database and `ArticleStore`. `close()` cancels work started through it, waits for it, then deletes the database file. `SessionModel` holds `session: UserSession?` in place of `user` and the `UserDataStore` list: it creates one at sign-in and at a restored launch, and closes it at logout and when `AuthService.signOuts` fires. Remove the `UserDataStore` protocol.

## Verify
- `articleStore.test.ts` ported against an in-memory database, minus the cases for `listRecent` / `listFavorites` / `listPendingSaves` and the read-failure fallbacks, which are replaced by the observations (listed in the PR).
- New tests: `syncPage` deletes a row the server dropped, keeps one with an outbox row, and keeps rows outside the window and on its boundaries; `observeReads` emits after upsert, remove and an outbox change; recreate-on-corruption; `SessionModel` closes the session on logout and on a service-initiated sign-out, and the file is gone afterwards.

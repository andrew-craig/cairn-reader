---
id: task_288e
title: [iOS native 06b] Rework the store for decisions 8, 10 and 11: App Group database, observations, page sync, UserSession
type: task
status: closed
priority: 1
labels: [ios,offline]
blocked_by: []
parent: epic_bc48
remote_task_url: null
created_at: 2026-11-15T00:05:27Z
updated_at: 2026-11-15T03:13:43Z
---

## Goal
Bring the store that task_c194 (06, merged in #461 against the earlier plan) built into line with decision_adb1: it becomes the source of truth for Reads (decision 8), is owned by a `UserSession` (decision 10) and lives in the App Group (decision 11). The schema and the guarded upsert from 06 stay.

## Starting point
`Store/AppDatabase.swift` (a lazily opened `DatabaseQueue` singleton in Application Support, `v1` migration) and `Store/ArticleStore.swift` (every RN store method, with read failures answering empty), tested by `ArticleStoreTests.swift`.

## Plan
- [x] Database file in the App Group container from task_1162, opened as GRDB's database-sharing guide prescribes for a database shared with an extension (`DatabasePool`, WAL, suspension handling). Remove `AppDatabase.shared`; the Application Support file from earlier builds of this branch is not migrated.
- [x] If the database cannot be opened or migrated, delete it and recreate it once: it is a cache plus the outbox. This replaces the retry-on-next-call behaviour.
- [x] Edit the `v1` migration in place (nothing has shipped): drop the `attempts` column from `outbox`.
- [x] `ArticleStore.syncPage(_:newerThan:hasMore:)`: in one transaction, upsert a Reads page and delete the stored rows strictly inside the page's `added_at` window that the page did not return and that have no outbox row. The window runs from the previous page's last `added_at` (none for the first page) down to this page's last `added_at` (none when `hasMore` is false).
- [x] Observations (`ValueObservation`): `observeReads(window)` yielding articles newest first, each with a `pendingSave` flag (an outbox `list` row exists); `observeArticle(id)` for the reader; `observeStoredIDs()` for the Feed's exclusion set.
- [x] Remove what the observations replace: `listRecent`, `listFavorites`, `listPendingSaves`, `clear`, and the answer-empty-on-failure wrapper. Keep `getById` and `listPrefetchCandidates`, now throwing.
- [x] `UserSession`: created with the signed-in `User`, owns the database and `ArticleStore`. `close()` cancels work started through it, waits for it, then deletes the database file. `SessionModel` holds `session: UserSession?` in place of `user` and the `UserDataStore` list: it creates one at sign-in and at a restored launch, and closes it at logout and when `AuthService.signOuts` fires. Remove the `UserDataStore` protocol.
- [x] Update `apps/ios/CLAUDE.md`.

## Verify
- `ArticleStoreTests` still pass for the upsert guards, body staleness, eviction and user-state updates; the cases for removed methods are deleted and listed in the PR.
- New tests: `syncPage` deletes a row the server dropped, keeps one with an outbox row, and keeps rows outside the window and on its boundaries; `observeReads` emits after upsert, remove and an outbox change; recreate-on-corruption; `SessionModel` closes the session on logout and on a service-initiated sign-out, and the file is gone afterwards.

## Review
- Built and tested locally (Xcode 26.6, iPhone 17 simulator, iOS 26): 189 tests pass, warnings as errors.
- `syncPage` takes `olderThan:` rather than `newerThan:`: the argument is the previous page's last `added_at`, and the page holds items older than it.
- The database file is per user (`store-<user id>.sqlite`), so a file left behind by a sign-out that did not finish (app killed mid-close) is never read for another account.
- Database opening is lazy and done once per session (one recreate on failure; a second failure is returned for the rest of the session), so `UserSession` creation never throws and a broken database surfaces through the store calls and observations.
- `UserSession.start(_:)` is the way later tasks (outbox drain, sync) start work that `close()` cancels and awaits.
- Removed test cases (methods removed): listRecent ordering, listFavorites, listPendingSaves (now the `pendingSave` flag of `observeReads`), clear and clear-clears-outbox (now `UserSession.close()` erasing the file), reads-never-throw, retry-after-failed-open (now recreate-on-failure), and the two `UserDataStore` cases in SessionModelTests.


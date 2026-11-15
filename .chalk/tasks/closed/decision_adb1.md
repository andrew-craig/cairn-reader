---
id: decision_adb1
title: [iOS native 00b] Architecture changes before the store: store as source of truth, outbox-only writes, session scope, App Group
type: decision
status: closed
priority: 1
labels: [ios,mobile]
blocked_by: []
parent: epic_bc48
remote_task_url: null
created_at: 2026-11-15T00:00:01Z
updated_at: 2026-11-15T00:01:49Z
---
Four changes to the plan for tasks 06 onward, made after reviewing the code from tasks 01–05 and the RN service layer the remaining tasks port. They continue the numbering of decision_1686.

| # | Decision | What it replaces | Affects |
|---|----------|------------------|---------|
| 8 | **DECIDED (owner, 2026-11-14): the on-device store is the source of truth for Reads.** The Reads list is a GRDB `ValueObservation` over `articles` (joined to `outbox` for the pending flag). Network pages are only written into the store. Feed, Bookmarks and search stay network lists. | The RN screen's in-memory list reconciled by hand with the store: the Reads invalidation flag, the queued-saves snapshot rule, re-reading the pending indicator after each sync, the reader's "remove from the originating list" callbacks, and passing the list and index into the reader route. | 06b, 08, 10, 11, 12 |
| 9 | **DECIDED (owner, 2026-11-14): the outbox is the only write path.** Every mutation is one transaction (store write + outbox row) followed by a drain. Nothing is sent to the backend except by the drain. | Store write, then a live request, then enqueue only on a retryable failure. That shape lets an older queued edit replay over a newer live one (favourite offline, unfavourite online before the drain, drain replays `true`), and loses the edit if the app dies between the store write and the enqueue. | 07, 10, 11, 12 |
| 10 | **DECIDED (owner, 2026-11-14): user data is scoped to the session.** One `UserSession` owns the database, store, outbox, feed cache, sync and a `ReadService` bound to the user id. `SessionModel` creates it at sign-in and closes it (cancel, then erase) at sign-out; the signed-in view tree is keyed on the user id. | The `UserDataStore` protocol and the `clear()` list in `SessionModel`, `ReadService` reading the user id from `AuthService` on every call, and RN-style global `SyncTrigger` / `Outbox` singletons. | 06b, 07, 08, 09 |
| 11 | **DECIDED (owner, 2026-11-15): a share extension is in scope for the future, so its storage constraints are met now.** The database, the Keychain session and the saved server URL live in the App Group `group.com.cairnapp.cairnreader`. The extension itself is not part of this epic. | Database in the app's own Application Support, Keychain item with no access group, server URL in `UserDefaults.standard`. Moving them later would force a sign-out and a re-download (decision 4: no migrations). | 05a (task_1162), 06b, 17 |

## Consequences accepted
- **Screen and mutation tests are re-expressed, not ported case for case.** `articleStore.test.ts` and `outbox.test.ts` still port (minus the cases for removed behaviour, each listed in its PR). `articleMutations.test.ts`, `ReadsScreen*.test.tsx`, `FeedScreen.test.tsx` and the two detail-screen suites are rewritten against the new shape. The epic's "ported case for case" principle is amended accordingly.
- **A definitive rejection arrives from the drain, not from the call.** Mutations do not throw for backend reasons. The drain drops the row, the next Reads sync restores the server's state in the store, and the UI follows through the observation. Rejected archives and saves are announced through one `Outbox.rejections` stream that the signed-in root turns into an alert.
- **The store now deletes.** A Reads page sync removes stored rows inside the fetched window that the server no longer returns (archived elsewhere), except rows with an outbox entry.
- **A broken database breaks Reads.** There is no network-only fallback list any more (`ReadsScreen.storeFailure.test.tsx` does not port). The database is a cache plus the outbox, so if it cannot be opened or migrated it is deleted and recreated once.
- **The iOS and Expo service layers diverge further** than the epic's risk section assumed. The ordering defect behind decision 9 also exists in `apps/mobile` and is tracked separately as bug_2140.
- **Decision 11 needs a provisioning change:** the App Group must be registered and the match profiles regenerated (task_1162, task_4713).

## Not decided here
- The share extension's own design. Two things it must solve are known: `AuthService` caches the session in memory and refresh tokens rotate, so two processes need a coordinated refresh; and an extension writing to the shared database must follow GRDB's database-sharing guide, as task 06b does on the app side.

## Done when
- [x] Tasks 07, 08, 09, 10, 11, 12, 17 and 18 and the epic are rewritten against these decisions. Task 06 (task_c194) merged against the earlier plan while this was in review, so its share of the change is task_288e (06b). task_1162 (05a) blocks 06b, which blocks 07.

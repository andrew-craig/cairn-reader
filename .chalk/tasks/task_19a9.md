---
id: task_19a9
title: [explore/fetcher] Integration suite: test isolation + EndToEndFlow hardcoded IDs + EmptyResponse behavior
type: task
status: open
priority: 2
labels: []
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-29T23:42:51Z
updated_at: 2026-08-29T23:43:03Z
---


## Detail (found while working bug_96d7, 2026-08-30)

bug_96d7 fixed the 3 named root causes (PRs #357 NULL-scan, #358 SSRF/httptest, #359 UpdateFetchResult arity). Running the now-compiling `//go:build integration` suite for services/explore/fetcher against local Postgres surfaced further, independent problems:

1. **No per-test DB isolation.** `testutil.SetupTestDB` connects to one shared `fetcher_db`; `CleanupTestDB` only `DELETE`s `feeds` + `fetch_history` (not the SERIAL sequence). Run individually every test passes; run together, `internal/sync` tests fail on accumulated rows (`TestSyncFeeds_SkipsComments` "got 5", `TestSyncFeeds_DuplicatesInList` "got 7", `TestSyncFeeds_PreservesExistingMetadata`, `TestSyncFeeds_HTTPError`). TESTING.md claims "each test creates a unique test database" — explore/fetcher's testutil does not. Fix: unique schema/DB per test (mirror explore/recommender testutil) or robust truncate+sequence-reset in CleanupTestDB.

2. **`TestEndToEndFlow` hardcodes `UPDATE feeds SET url=... WHERE id = 1/2/3`.** IDs come from a never-reset SERIAL, so these match zero rows and the fetcher hits the real `https://example.com/feed1.xml` (404). Also its RSS servers bind loopback → needs `fetchtest.AllowLoopback`. Fix: capture returned IDs from `ListFeeds`, and thread an AllowLoopback context.

3. **`TestSyncFeeds_EmptyResponse` vs code.** Test asserts an empty feed list is not an error (success, 0 feeds imported); `syncFeeds` returns `no feed URLs found in <source>`. Behavior decision needed: should an empty upstream list be a hard error? If not, change `syncFeeds`; if yes, change the test.

---
id: bug_96d7
title: [explore/fetcher] Integration-tagged test suite is broken: compile errors, SSRF-guard/httptest conflict, NULL scan bug
type: bug
status: open
priority: 2
labels: [quality,tests,explore]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-15T23:28:41Z
updated_at: 2026-08-15T23:28:41Z
---
Discovered while wiring the new integration-test CI job (task_ac75). services/explore/fetcher
tags essentially its entire test suite with //go:build integration, so none of it has ever
compiled or run in CI. That let real drift accumulate silently:

1. Compile errors (2 files): fetcher/integration_test.go and
   fetcher/internal/db/feed_repository_test.go call
   repo.UpdateFetchResult(ctx, id, success) but the real signature is now
   UpdateFetchResult(ctx, id, success, string, string) — tests were never updated
   when the repository method's signature grew two params.

2. SSRF guard vs. httptest: fetcher/internal/fetcher/fetcher_test.go (TestFetchSingleFeed_*)
   uses httptest.NewServer (binds 127.0.0.1) to mock feed responses. The pkg/rss/fetch
   SSRF guard (R2) now rejects all loopback/private/link-local addresses unconditionally,
   so these tests fail with "fetch: blocked address ... 127.0.0.1". Needs a documented
   test-only bypass or dependency-injected transport in the guarded fetcher — don't weaken
   the guard itself.

3. NULL scan bug: fetcher/internal/sync/feed_sync_test.go
   (TestSyncFeeds_Success) fails with "cannot scan NULL into *string" on the feeds.title
   column via ListFeeds — looks like a real repository bug (title is nullable in the
   schema but the scan destination isn't), not just a test issue. Re-verify against
   current main and confirm before fixing.

task_ac75's new `test-integration-explore` CI job intentionally scopes to
`./recommender/...` only and skips fetcher, so this doesn't block that PR — but it means
fetcher's ~20+ integration tests (feed repository, fetcher, sync) still don't run anywhere.
Follow the strategy's normal loop (re-verify → failing test → fix → prove → PR) once picked
up; likely 2-3 separate PRs given the three distinct root causes.

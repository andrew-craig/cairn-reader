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

---

## Re-confirmed by the Cairn Simplification Audit (2026-08-17)

Re-verified at HEAD `a6c56a1`. The audit reaches this bug from the other direction: it groups the
`UpdateFetchResult` arity drift (item 1 above) under its **X9 pattern — "tests that do not
execute"** — and makes the causal point explicit: in every X9 case the code was wrong *and* the
test that would have caught it did not run. This bug is the worked example.
**Audit report:** https://claude.ai/code/artifact/286883fb-3f93-49c4-942f-4880251a409f

**One correction to the scoping note above.** This task states that the `test-integration-explore`
job "intentionally scopes to `./recommender/...` only and skips fetcher". It actually scopes to:

```
go test -race -tags=integration -count=1 -timeout 10m ./recommender/internal/db/...
```

One directory *inside* recommender — not all of it. So `services/explore/recommender/integration_test.go`
and `integration_shown_test.go` (**1,183 lines**) run nowhere either, falling through the gap between
this bug's scope and that job's. That exclusion is now tracked as **task_527a**; it is not covered here.

**Also related:** the audit cites a second X9 instance in the same family — an `"id"`/`"user_id"`
column mismatch across **26 test call sites**. If that surfaces while you are working item 1's
signature drift, note it rather than absorbing it silently; it may warrant its own task.

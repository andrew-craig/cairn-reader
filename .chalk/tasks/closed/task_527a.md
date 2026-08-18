---
id: task_527a
title: [Audit X9/Tier 0] explore integration job runs only recommender/internal/db — 1,183 lines of recommender integration tests never run
type: task
status: closed
priority: 1
labels: [quality,ci,audit-x9]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-17T10:12:17Z
updated_at: 2026-08-18T21:00:07Z
---
**Source:** Cairn Simplification Audit (read-only pass at HEAD `a6c56a1`, 2026-08-16) — https://claude.ai/code/artifact/286883fb-3f93-49c4-942f-4880251a409f
Read docs/QUALITY_REMEDIATION_STRATEGY.md §0 (rules of engagement) and §2.6 (definition of done) before starting. One finding, one branch, one PR. Re-verify before fixing — all file:line references below were confirmed at `a6c56a1`.

**Audit pattern:** X9 (tests that do not execute) | **Tier 0** — prerequisite.

## Problem
`.github/workflows/go-checks.yml`, job `test-integration-explore`, runs:
```
go test -race -tags=integration -count=1 -timeout 10m ./recommender/internal/db/...
```
Scoped to **one subdirectory**. The recommender's root-package integration tests — `services/explore/recommender/integration_test.go` and `integration_shown_test.go`, **1,183 lines** — are excluded and run nowhere. The job's own comment acknowledges this: they use an older hardcoded-connection setup (TEST_DB_* against a pre-existing `cairn`/`cairn_test_db` database, no self-migration) rather than `internal/testutil.SetupTestDB`'s testcontainers provisioning.

These are the tests cited elsewhere as the safety net for recommender behaviour. They are not one.

## What to do
1. Migrate the two root-package files onto `internal/testutil.SetupTestDB` (testcontainers, self-migrating) so they need no pre-provisioned database — this is the in-repo reference pattern, don't invent a new one.
2. Widen the job's scope to cover them.
3. Triage what fails; expect drift, since nothing has compiled or run these in CI.

## Done when
- The recommender's root-package integration tests execute in CI, provisioning their own database.

## Related / scope boundary
- bug_96d7 covers `services/explore/fetcher`'s integration-tagged suite (compile errors, SSRF-guard/httptest conflict, NULL scan bug). **That is a different exclusion** — bug_96d7's text describes this job as scoping to `./recommender/...`, but it actually scopes to `./recommender/internal/db/...`, which is why the recommender root tests fall through the gap between the two tasks. Do not assume bug_96d7 covers this.

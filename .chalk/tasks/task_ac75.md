---
id: task_ac75
title: [Integration tier] Add a CI job running //go:build integration tests against a real Postgres
type: task
status: open
priority: 1
labels: [quality,wave3,ci]
blocked_by: [task_2ff6]
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-09T06:53:55Z
updated_at: 2026-08-14T22:49:01Z
---
Read docs/QUALITY_REMEDIATION_STRATEGY.md §0 (rules of engagement) and §2.6 (definition of done) before starting. Read the full finding text in docs/CODE_QUALITY_REVIEW.md. One finding, one branch, one PR. Re-verify on main first — cited line numbers are from 2026-07-05 and drift.

**Finding:** Integration tier | **Wave 3** | **Recipe:** R10 (strategy §2.5)
**Touches:** .github/workflows/go-checks.yml
**Blocked by:** the Wave 2 database tasks — their `//go:build integration` tests are this job's payload.

## Problem
The repository has `//go:build integration`-tagged tests but **nothing runs them in CI**. Every Wave 2 DB fix (PK collision, ON CONFLICT, unique dedup index, bounded deletes) is only provable at this tier; without a CI job those tests rot immediately and the bugs regress silently.

## What to do
1. Add an integration job to `go-checks.yml`: a Postgres service container, real migrations applied, then `go test -race -tags=integration ./...` per service that has tagged tests.
2. Match the local invocation documented in `docs/TESTING.md` so the two do not drift.
3. Router-inventory tests from the Wave 1 auth tasks ride the normal test job — no extra config needed for those.

## Done when
- The Wave 2 integration tests run on every PR and fail the build when they fail.


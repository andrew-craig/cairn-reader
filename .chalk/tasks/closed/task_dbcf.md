---
id: task_dbcf
title: [Web CI] apps/web has no CI at all — add web-checks.yml (tsc, eslint, vitest)
type: task
status: closed
priority: 1
labels: [quality,wave3,ci]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-09T06:53:55Z
updated_at: 2026-08-15T22:54:01Z
---
Read docs/QUALITY_REMEDIATION_STRATEGY.md §0 (rules of engagement) and §2.6 (definition of done) before starting. Read the full finding text in docs/CODE_QUALITY_REVIEW.md. One finding, one branch, one PR. Re-verify on main first — cited line numbers are from 2026-07-05 and drift.

**Finding:** Web CI gap (verified 2026-08-09) | **Wave 3** | **Recipe:** R10 (strategy §2.5)
**Touches:** .github/workflows/web-checks.yml (new)

## Problem
`apps/web` has **no CI at all**. `go-checks.yml` covers the Go services and `mobile-checks.yml` covers mobile, but no workflow runs any check on `apps/web/**`. Web typecheck, lint and tests are currently the individual developer's responsibility, which means they are effectively optional.

## What to do
1. Copy the shape of `.github/workflows/mobile-checks.yml`, path-filtered on `apps/web/**`.
2. Steps: `npx tsc --noEmit`, `npx eslint .`, `npx vitest run`.
3. Confirm it is green on current main before merging — if the existing code fails a check, fix that in the same PR or the ratchet cannot land.

## Done when
- A PR touching `apps/web/**` runs and must pass typecheck, lint and vitest.


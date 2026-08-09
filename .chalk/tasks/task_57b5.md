---
id: task_57b5
title: [H11] Delete the ~250-line dead JWT-validation duplicate in services/users/internal/auth/jwt.go
type: task
status: open
priority: 2
labels: [quality,wave4,consolidation]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-09T06:53:56Z
updated_at: 2026-08-09T06:53:56Z
---
Read docs/QUALITY_REMEDIATION_STRATEGY.md §0 (rules of engagement) and §2.6 (definition of done) before starting. Read the full finding text in docs/CODE_QUALITY_REVIEW.md. One finding, one branch, one PR. Re-verify on main first — cited line numbers are from 2026-07-05 and drift.

**Finding:** H11 | **Wave 4** | **Recipe:** R11 (strategy §2.5)
**Touches:** services/users/internal/auth/jwt.go (delete), its tests

## Problem
`services/users/internal/auth/jwt.go` duplicates roughly 250 lines of `pkg/auth/validator.go`, and the duplicate is **dead in the request path** — only tests call it. Its passing tests give false confidence, and security fixes to the canonical validator will never propagate to it.

## What to do
1. Confirm on current main that nothing in the request path imports it (grep, and check the build still passes with it removed).
2. Port any assertion the duplicate's tests make that `pkg/auth`'s tests do not already cover, onto `pkg/auth`.
3. Delete the duplicate and its tests **in the same PR**.

## Done when
- Only `pkg/auth/validator.go` implements JWT validation; the users service builds and tests pass.


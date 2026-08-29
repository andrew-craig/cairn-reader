---
id: task_8b77
title: [users build tag] services/users/test/integration lacks //go:build integration → bare go test hangs
type: task
status: closed
priority: 2
labels: [quality,wave3,ci]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-09T06:53:55Z
updated_at: 2026-08-28T11:31:28Z
---
Read docs/QUALITY_REMEDIATION_STRATEGY.md §0 (rules of engagement) and §2.6 (definition of done) before starting. Read the full finding text in docs/CODE_QUALITY_REVIEW.md. One finding, one branch, one PR. Re-verify on main first — cited line numbers are from 2026-07-05 and drift.

**Finding:** users build tag (medium, CI/DX) | **Wave 3** | **Recipe:** R10 (strategy §2.5)
**Touches:** services/users/test/integration/*.go

## Problem
`services/users/test/integration/*.go` is missing the `//go:build integration` tag that every sibling service uses (`services/read`, `services/read/email`, `services/explore` all gate theirs). Consequence: a naive `go test ./...` in `services/users` — or at repo root — **hangs indefinitely** in `WaitForDatabase` (`testutil/setup.go:221`) trying to reach Postgres at `localhost:5433`, instead of skipping. A real CI and onboarding footgun.

## What to do
1. Add `//go:build integration` to the files, matching the sibling convention exactly.
2. Verify `go test ./...` in `services/users` completes without a database.

## Done when
- Bare `go test ./...` in services/users and at repo root no longer hangs.


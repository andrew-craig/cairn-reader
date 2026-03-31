---
id: task_e669
title: Add TypeScript type-check CI step
type: task
status: closed
priority: 1
labels: [ci,mobile]
blocked_by: []
parent: epic_b701
created_at: 2026-03-31T07:57:02Z
updated_at: 2026-03-31T08:16:55Z
---
Add a GitHub Actions job that runs 'tsc --noEmit' on PRs touching apps/mobile/**. The type-check script already exists in package.json. This is the highest-value check — it catches type errors, null safety violations, and API contract mismatches that cause runtime crashes. Zero setup needed beyond the workflow file.

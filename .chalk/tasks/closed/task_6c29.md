---
id: task_6c29
title: Add ESLint CI step
type: task
status: closed
priority: 1
labels: [ci,mobile]
blocked_by: []
parent: epic_b701
created_at: 2026-03-31T07:57:04Z
updated_at: 2026-03-31T08:16:55Z
---
Add a GitHub Actions job that runs 'eslint .' on PRs touching apps/mobile/**. ESLint is already configured (.eslintrc.js) with expo + @typescript-eslint/recommended rules. Catches unused vars, explicit any usage, and Expo-specific issues. No setup needed beyond the workflow file.

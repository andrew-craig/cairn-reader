---
id: epic_b701
title: Mobile App PR Checks
type: epic
status: closed
priority: 1
labels: []
blocked_by: []
parent: null
created_at: 2026-03-31T07:56:57Z
updated_at: 2026-03-31T08:16:55Z
---
Implement GitHub Actions CI checks for the mobile app (apps/mobile/) to protect against bad deploys. Currently the mobile app has zero PR checks — the only workflows are TestFlight deployment triggers. The backend has docker-test.yml for PR validation, but nothing equivalent exists for mobile. This epic covers adding TypeScript type checking, ESLint, Metro bundle verification, unit tests, and dead code detection as required PR checks.

---
id: task_e3fe
title: Set up Jest and add unit tests
type: task
status: closed
priority: 2
labels: [ci,mobile,testing]
blocked_by: []
parent: epic_b701
created_at: 2026-03-31T07:57:09Z
updated_at: 2026-03-31T08:16:55Z
---
Install Jest + @testing-library/react-native and create initial unit tests. Currently no test framework is installed at all. The mobile CLAUDE.md has aspirational test patterns documented. Start with service layer tests (pure functions in src/services/ — easy wins), then expand to component tests. Add a CI job that runs 'npm test' on PRs touching apps/mobile/**.

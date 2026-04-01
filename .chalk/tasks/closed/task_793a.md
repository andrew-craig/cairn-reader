---
id: task_793a
title: Create go-checks.yml workflow with go vet matrix job
type: task
status: closed
priority: 1
labels: [ci,backend]
blocked_by: []
parent: epic_d014
created_at: 2026-04-01T03:36:17Z
updated_at: 2026-04-01T17:44:19Z
---
Create .github/workflows/go-checks.yml following the mobile-checks.yml pattern. Start with a go vet job using a matrix strategy for all 4 Go modules:
- services/explore/ (go.mod)
- services/read/ (go.mod, covers content/ and fetcher/)
- services/read/email/ (go.mod)
- services/users/ (go.mod)

Path filters should trigger on changes to services/**, pkg/**, and the workflow itself. Use Go 1.24 setup. Must handle the replace directives for pkg/ modules (full repo checkout needed).

Tooling status: go vet is built into Go, already in all Makefiles. No setup needed.
Runs standalone: YES (no Docker/Postgres required).

Note: The read service has sub-services (content, fetcher) under one go.mod, so `go vet ./...` from services/read/ covers both. The email service has its own go.mod at services/read/email/.

---
id: task_a0c7
title: Add go test with race detection job to go-checks.yml
type: task
status: closed
priority: 1
labels: [ci,backend,testing]
blocked_by: []
parent: epic_d014
created_at: 2026-04-01T03:36:25Z
updated_at: 2026-04-01T19:04:45Z
---
Add a unit test job to go-checks.yml that runs go test with race detection. Use matrix strategy. Test commands per service:
- explore: go test -race ./...
- read (content): cd content && go test -race ./...
- read (fetcher): cd fetcher && go test -race ./...
- read/email: go test -race ./...
- users: go test -race ./internal/... ./pkg/...

Exclude integration test tags. Use -count=1 to disable test caching in CI. Add -timeout 5m as safety net.

Tooling status: go test is built into Go. Race detector needs CGO_ENABLED=1 (default on Linux).
Runs standalone: YES for unit tests (no Docker/Postgres required). Integration tests are separate (P3, future task).

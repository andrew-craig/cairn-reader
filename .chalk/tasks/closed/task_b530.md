---
id: task_b530
title: Add gofmt check job to go-checks.yml
type: task
status: closed
priority: 2
labels: [ci,backend]
blocked_by: []
parent: epic_d014
created_at: 2026-04-01T03:38:20Z
updated_at: 2026-04-02T00:49:41Z
---
Add a gofmt verification job to go-checks.yml. This should fail if any Go files are not properly formatted. Use the standard pattern:
  gofmt -l . | grep . && exit 1 || true
or use gofmt -d and check for non-empty output.

Run across all service directories in the matrix. Could also be a single job that checks all Go files repo-wide since gofmt has no module awareness needed.

Tooling status: gofmt is built into Go. Already in all Makefiles as `go fmt ./...` but that auto-fixes rather than checking.
Runs standalone: YES.

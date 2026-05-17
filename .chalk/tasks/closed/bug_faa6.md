---
id: bug_faa6
title: Users Service: make test fails due to nonexistent ./pkg/... path in Makefile
type: bug
status: closed
priority: 2
labels: []
blocked_by: []
parent: null
remote_task_url: null
created_at: 2026-05-17T10:14:24Z
updated_at: 2026-05-17T10:15:26Z
---

## Bug Description

`make test` and `make test-coverage` in `services/users/` fail with exit code 1 because the Makefile includes `./pkg/...` in the `go test` glob, but no `pkg/` directory exists (and never has in git history).

**Error:**
```
# ./pkg/...
pattern ./pkg/...: lstat ./pkg/: no such file or directory
FAIL    ./pkg/... [setup failed]
```

All actual unit tests pass — the failure is purely from the nonexistent path.

**Root cause:** The Makefile was written referencing a planned `pkg/auth/` shared library directory (mentioned in the service CLAUDE.md), but that directory was never created. The test commands reference it anyway.

**Affected lines:**
- `services/users/Makefile:43` — `test` target: `go test -v ./internal/... ./pkg/...`
- `services/users/Makefile:85` — `test-coverage` target: `go test -v -coverprofile=coverage.out ./internal/... ./pkg/...`

## Plan

- [x] Identify affected Makefile targets
- [x] Remove `./pkg/...` from `test` and `test-coverage` targets
- [x] Verify `make test` passes cleanly (exit code 0)
- [x] Verify no other Makefile targets reference `./pkg/...`

## Result

Removed `./pkg/...` from `test` (line 43) and `test-coverage` (line 85) targets in `services/users/Makefile`. All unit tests pass with exit code 0.


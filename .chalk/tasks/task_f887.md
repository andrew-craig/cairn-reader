---
id: task_f887
title: Add test coverage reporting to go-checks.yml
type: task
status: open
priority: 3
labels: [ci,backend,testing]
blocked_by: []
parent: epic_d014
created_at: 2026-04-01T03:38:21Z
updated_at: 2026-04-01T03:38:21Z
---
Add test coverage reporting to the unit test job. Generate coverage profiles and either:
- Post coverage summary as a PR comment
- Upload to a coverage service (codecov, coveralls)
- Set minimum coverage thresholds that fail the build

The users service already has a test-coverage Makefile target: go test -coverprofile=coverage.out ./internal/... ./pkg/...

Start with coverage report generation (-coverprofile) and a summary output. Coverage thresholds can be enforced later once baselines are established.

Tooling status: go test -coverprofile is built into Go. PR comment posting or coverage service integration needs setup.
Runs standalone: YES.

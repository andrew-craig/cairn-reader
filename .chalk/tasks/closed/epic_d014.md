---
id: epic_d014
title: Go Backend CI Checks
type: epic
status: closed
priority: 1
labels: []
blocked_by: []
parent: null
created_at: 2026-03-31T10:28:02Z
updated_at: 2026-04-03T22:38:01Z
---
Add GitHub Actions CI checks for the Go backend services (explore, read, users) to catch bugs and regressions on PRs. Currently backend PRs only verify Docker images build (docker-test.yml). The mobile app already has comprehensive PR checks (mobile-checks.yml). This epic covers adding go vet, gofmt verification, unit tests with race detection, go build verification, golangci-lint, and test coverage reporting as required PR checks for all Go backend services.

## Current State
- Only CI check: docker-test.yml (verifies Docker images build)
- No unit tests, linting, vetting, or format checking in CI
- All services have Makefile targets for test/fmt/vet/lint but they're not enforced
- No golangci-lint configuration exists (.golangci.yml)
- Services share pkg/ modules via replace directives in go.mod

## Go Module Structure
- services/explore/ - own go.mod (Go 1.24.7)
- services/read/ - own go.mod (Go 1.24.7), covers content/, fetcher/ sub-services
- services/read/email/ - own go.mod (Go 1.24.7)
- services/users/ - own go.mod (Go 1.24.0)
- pkg/* - shared modules referenced via replace directives

## Workflow Strategy
Single workflow file (go-checks.yml) with matrix strategy and path filters, following the mobile-checks.yml pattern. Each service gets independent job runs triggered by path changes. Jobs that can run standalone (no Docker/Postgres) are prioritized first.

## Checks (prioritized)
1. P1: go vet - catches common bugs, no setup needed
2. P1: go build verification - ensures compilation succeeds
3. P1: go test with race detection - unit tests already exist
4. P2: gofmt check - verify code is formatted (fail on diff)
5. P2: golangci-lint - requires new .golangci.yml config
6. P3: test coverage reporting - add coverage thresholds
7. P3: integration tests (future) - requires Docker/Postgres, separate workflow

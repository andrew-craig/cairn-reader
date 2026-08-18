---
id: task_f927
title: [Audit X9/Tier 0] No CI job reaches 4 standalone pkg/* modules — 85 tests never run, incl. the SSRF guard suite
type: task
status: open
priority: 1
labels: [quality,ci,security,audit-x9]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-17T10:12:17Z
updated_at: 2026-08-17T10:12:17Z
---
**Source:** Cairn Simplification Audit (read-only pass at HEAD `a6c56a1`, 2026-08-16) — https://claude.ai/code/artifact/286883fb-3f93-49c4-942f-4880251a409f
Read docs/QUALITY_REMEDIATION_STRATEGY.md §0 (rules of engagement) and §2.6 (definition of done) before starting. One finding, one branch, one PR. Re-verify before fixing — all file:line references below were confirmed at `a6c56a1`.

**Audit pattern:** X9 (tests that do not execute) | **Tier 0** — prerequisite.

## Problem
`pkg/*` modules are **separate Go modules** (each has its own go.mod), so no service-scoped test job reaches them. `.github/workflows/go-checks.yml` has exactly one pkg job — `test-pkg-auth`, gated on `pkg-auth` changes. Every other pkg module with tests runs **nowhere**:

| Module | Test funcs | CI job |
|---|---|---|
| pkg/rss | 57 | none |
| pkg/middleware | 20 | none |
| pkg/config | 6 | none |
| pkg/logging | 2 | none |
| **total** | **85** | — |

(pkg/api and pkg/models have no test files at all.)

**The security consequence:** the SSRF guard suite lives in `pkg/rss` — the guard that R2 added and that `url_detector.go` deliberately wires. **Its tests have never been enforced by CI.** `pkg/middleware` is where the rate-limiter tests live too (see the rate-limiter race task), so that fix has nowhere to prove itself either.

Note the `changes` filter already computes a `pkg` output and fans it out to the four service jobs — but a change to `pkg/rss` only triggers the *services*' suites, never pkg/rss's own.

## What to do
1. Add a matrix job over the pkg modules that have tests (rss, middleware, config, logging), each with `working-directory` and its own `cache-dependency-path`, mirroring `test-pkg-auth`'s shape.
2. Land with `fail-fast: false` and triage per module.

## Done when
- All 85 tests execute in CI on changes to their module, and the SSRF guard suite is enforced on every PR that touches pkg/rss.

## Expect red on first enablement
**That is the finding, not a regression** — these 85 tests have never run, so their pass state is unknown by construction. Triage per module; do not delete or skip a test to get green.

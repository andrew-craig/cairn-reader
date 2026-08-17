---
id: task_dc5b
title: [Audit/Tier 3] content worker rolls its own config loader instead of internal/config.Load()
type: task
status: open
priority: 3
labels: [quality,consolidation,audit]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-17T12:51:30Z
updated_at: 2026-08-17T12:51:30Z
---
**Source:** Cairn Simplification Audit — https://claude.ai/code/artifact/286883fb-3f93-49c4-942f-4880251a409f · file:line detail supplied by the audit author 2026-08-17 and re-verified at HEAD `a6c56a1`.
Read docs/QUALITY_REMEDIATION_STRATEGY.md §0 (rules of engagement) and §2.6 (definition of done) before starting. One finding, one branch, one PR.

**Audit tier:** 3 (structural) | **Verified.**

## Problem
`services/read/content/cmd/worker/main.go` reimplements configuration loading rather than using the service's own loader:
- its own `Config` struct — :108-110
- its own `loadConfig()` — :113-145
- its own `getEnv*` helpers — :147, :155, :169

Meanwhile the sibling entrypoint `services/read/content/cmd/content/main.go:34` calls `internal/config.Load()`. Two binaries in one service, two unrelated notions of how that service is configured.

**The finding is the mechanism, not the wiring** — the point is that the worker has a *parallel loader*, not merely that it reads different variables. So the fix is to delete the parallel mechanism, not to align the two variable lists.

## What to do
1. Diff the worker's `loadConfig()` against `internal/config.Load()`: which keys, which defaults, which types. Any key the worker needs that `Load()` lacks is an addition to `Load()`, not a reason to keep the copy.
2. Repoint the worker onto `internal/config.Load()` and delete its `Config`, `loadConfig`, and `getEnv*` helpers.
3. Verify the worker still starts with the same environment the deployment provides (check `infrastructure/docker/` compose env for the worker service).

## Done when
- One config loader per service; the worker's parallel loader is gone and the worker boots unchanged.

## Related
- **task_7ada** ([Env parsing] collapse `pkg/env` vs `pkg/config` vs two service-local copies) is the repo-wide version of this class. The worker's `getEnv*` helpers are plausibly among the "service-local copies" it refers to. Check task_7ada's scope before starting: if it is being worked, fold this in rather than landing two overlapping refactors of the same helpers.

---
id: task_7ada
title: [Env parsing] Collapse pkg/env vs pkg/config vs two service-local copies into one
type: task
status: open
priority: 3
labels: [quality,wave4,consolidation]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-09T06:53:56Z
updated_at: 2026-08-09T06:53:56Z
---
Read docs/QUALITY_REMEDIATION_STRATEGY.md §0 (rules of engagement) and §2.6 (definition of done) before starting. Read the full finding text in docs/CODE_QUALITY_REVIEW.md. One finding, one branch, one PR. Re-verify on main first — cited line numbers are from 2026-07-05 and drift.

**Finding:** Theme 3 (env parsing) | **Wave 4** | **Recipe:** R11 (strategy §2.5)
**Touches:** pkg/env, pkg/config, two service-local copies

## Problem
Env-var parsing is duplicated four times, **two of them inside `pkg/` itself** (`pkg/env` and `pkg/config`), with **divergent duration-parsing behavior**. Two competing "shared" packages is worse than one plus a copy — nobody knows which is canonical.

## What to do
1. **Characterize first:** diff all four implementations, especially the duration semantics. Pick one behavior **explicitly**, document the choice in the PR, and write tests for it on the survivor before repointing anything.
2. Choose the canonical package and repoint callers one group at a time.
3. Delete the other three in the same PR as their last repoint.
4. Watch for config values whose meaning silently changes under the new duration semantics — call these out in the PR.

## Done when
- One env-parsing implementation remains, with one documented duration behavior.


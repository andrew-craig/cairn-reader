---
id: task_de6d
title: [Audit/Tier 4] Deduplicate web's transformToArticle / transformDetailToArticle — they differ on one line
type: task
status: closed
priority: 3
labels: [quality,consolidation,audit,web]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-17T12:52:21Z
updated_at: 2026-09-05T09:20:14Z
---
**Source:** Cairn Simplification Audit — https://claude.ai/code/artifact/286883fb-3f93-49c4-942f-4880251a409f · file:line detail supplied by the audit author 2026-08-17 and re-verified at HEAD `a6c56a1`.
Read docs/QUALITY_REMEDIATION_STRATEGY.md §0 (rules of engagement) and §2.6 (definition of done) before starting. One finding, one branch, one PR.

**Audit tier:** 4 (frontend) | **Verified.**

## Problem
`apps/web/src/services/read.ts` carries two mappers that differ on **a single line**:
- `transformToArticle` — :35-74
- `transformDetailToArticle` — :80-117

**This is web, not mobile** — do not go looking in `apps/mobile`.

## What to do
1. Diff the two functions and identify the one differing line precisely.
2. Collapse to one mapper, with the difference expressed as a parameter or by composing the detail case on top of the base case — whichever reads better at the two call sites.
3. If the difference turns out to be a **bug** rather than an intentional distinction, say so in the PR and fix it deliberately.

## Done when
- One mapper; both call sites behave exactly as before, or a deliberate behaviour change is documented.

## Restraint
This is a line-count win on stable code, which the audit rates low value. If collapsing them requires a parameter that makes either call site harder to read, leave them and close this task with that finding — the audit explicitly declined similar refactors (e.g. a shared focus-trap module) on exactly those grounds.

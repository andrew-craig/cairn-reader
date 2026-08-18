---
id: task_6fe1
title: [Audit/Tier 3] Collapse the triplicated content create pipeline in content_service.go
type: task
status: open
priority: 2
labels: [quality,consolidation,audit]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-17T12:50:50Z
updated_at: 2026-08-17T12:50:50Z
---
**Source:** Cairn Simplification Audit — https://claude.ai/code/artifact/286883fb-3f93-49c4-942f-4880251a409f · file:line detail supplied by the audit author 2026-08-17 and re-verified at HEAD `a6c56a1`.
Read docs/QUALITY_REMEDIATION_STRATEGY.md §0 (rules of engagement) and §2.6 (definition of done) before starting. One finding, one branch, one PR.

**Audit tier:** 3 (structural simplification) | **Verified.**

## Problem
`services/read/content/internal/service/content_service.go` implements the same create pipeline three times, plus a fourth divergent variant:
- `CreateFromURL` — :85-147
- `CreateFromHTML` — :151-213
- `BulkCreateFromHTML` — :280-372
- `UpdateContent` — :243-258 — a **fourth, divergent field-promotion variant**

The fourth is the reason this matters: field promotion drifting between the create paths and the update path means a field promoted on create is not necessarily promoted on update, so the same content ends up shaped differently depending on which door it came through.

## What to do
1. **Characterize first.** Tests on the current behaviour of all four paths, especially which fields each promotes and under what conditions. Where they diverge, decide **explicitly** which behaviour is correct and say so in the PR — do not silently pick one.
2. Extract the shared pipeline; keep the genuinely per-path differences as parameters or small hooks, not as a config object with a flag per caller.
3. Repoint all four, including `UpdateContent`'s promotion step.

## Done when
- One pipeline; each caller's behaviour either unchanged or changed deliberately and documented; divergences resolved rather than preserved.

## Restraint
The audit's own guidance applies: do not add parameterisation points speculatively. If collapsing all four requires more than about two switches, collapse three and say why the fourth stayed out.

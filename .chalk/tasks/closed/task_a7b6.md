---
id: task_a7b6
title: [Audit/Tier 3] Collapse the five near-identical fetch-outcome recording blocks in explore/fetcher FetchSingleFeed
type: task
status: closed
priority: 2
labels: [quality,consolidation,audit,explore]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-17T12:51:30Z
updated_at: 2026-08-29T23:46:39Z
---
**Source:** Cairn Simplification Audit — https://claude.ai/code/artifact/286883fb-3f93-49c4-942f-4880251a409f · file:line detail supplied by the audit author 2026-08-17 and re-verified at HEAD `a6c56a1`.
Read docs/QUALITY_REMEDIATION_STRATEGY.md §0 (rules of engagement) and §2.6 (definition of done) before starting. One finding, one branch, one PR.

**Audit tier:** 3 (structural) | **Verified.**

## Problem
`services/explore/fetcher/internal/fetcher/fetcher.go`, `FetchSingleFeed` (:62-145) contains **five near-identical blocks** that each pair a `UpdateFetchResult` call with a `RecordFetchHistory` call — one per exit path (success, and four failure shapes). Every early return has to remember to do both, in the right order, with the right arguments.

The failure mode is omission: a new exit path records one and forgets the other, and nothing catches it.

## Scope note — `FetchOutcome` does not exist
The audit's proposed shape names a `FetchOutcome` type. **That is a proposal, not a symbol in the tree** — do not go looking for it. Introducing it (or an equivalent) is the work.

## What to do
1. Characterize the five blocks: tabulate what each passes to `UpdateFetchResult` and `RecordFetchHistory`, including the empty-string `etag`/`lastModified` conventions (`internal/db/feed_repository.go:81-83` documents that on failure they are ignored).
2. Introduce one outcome value computed on each path, recorded **once** at a single exit point, so the two writes cannot diverge.
3. Preserve behaviour exactly; where two blocks already differ in a way that looks unintentional, flag it in the PR rather than silently normalising.

## Done when
- `UpdateFetchResult` and `RecordFetchHistory` are called from one place, and adding a new failure path cannot record one without the other.

## Blocked in practice by test coverage
`services/explore/fetcher`'s tests are integration-tagged and **do not currently compile** (bug_96d7 items 1-3), and no CI job runs them (task_527a covers the adjacent scoping gap). Refactoring this without a working suite is risky — land bug_96d7 first, or write unit coverage for `FetchSingleFeed`'s exit paths as part of this task.

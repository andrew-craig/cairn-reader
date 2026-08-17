---
id: task_fbb9
title: [Audit/Tier 5] Drop the duplicate constraint-backed indexes in the recommender and users migrations
type: task
status: open
priority: 3
labels: [quality,database,audit]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-17T12:52:21Z
updated_at: 2026-08-17T12:52:21Z
---
**Source:** Cairn Simplification Audit — https://claude.ai/code/artifact/286883fb-3f93-49c4-942f-4880251a409f · file:line detail supplied by the audit author 2026-08-17 and re-verified at HEAD `a6c56a1`.
Read docs/QUALITY_REMEDIATION_STRATEGY.md §0 (rules of engagement) and §2.6 (definition of done) before starting. One finding, one branch, one PR.

**Audit tier:** 5 (low priority — hazard reduction, no live defect).

## Problem
Several migrations create indexes that duplicate an index PostgreSQL already provides for a UNIQUE constraint or PRIMARY KEY. Each duplicate costs write throughput and disk for no read benefit.

**explore/recommender** (`services/explore/recommender/migrations/`):
- `000006_use_external_user_ids.up.sql:101`, `:98`, `:94` — redundant against the UNIQUE at `:106-107`, the UNIQUE at `:43`, and the PK at `:27` respectively
- `000003_fetcher_schema_updates.up.sql:24` — redundant against `000001_initial_schema.up.sql:35`

**users** (`services/users/migrations/`):
- `000002:25` vs `:20`
- `000005:21` vs `:17`
- `000006:18` vs `:15`

## What to do
1. Verify each pair with `\d+ <table>` (or `pg_indexes`) against a migrated database — confirm the constraint's implicit index genuinely covers the redundant one, **including column order**. A multi-column index is only redundant if the leading columns match.
2. **Needs real migrations, not edits to landed files.** Add new `DROP INDEX` migrations; do not modify the historical up-files, or already-migrated databases will diverge from fresh ones.
3. Drop with `DROP INDEX IF EXISTS` so re-running is safe.

## Done when
- Each confirmed duplicate is dropped by a forward migration, with a working down-migration, and no constraint loses its backing index.

## Priority note
No live defect — pure write-path overhead. Tier 5; do not preempt earlier tiers. If any pair turns out **not** to be redundant on inspection, leave it and record why.

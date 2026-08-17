---
id: task_fd42
title: [Readiness] Self-host /health/ready checks only 3 of 6 DBs → healthy while half the system is down
type: task
status: open
priority: 2
labels: [quality,wave3,ops]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-09T06:53:56Z
updated_at: 2026-08-09T06:53:56Z
---
Read docs/QUALITY_REMEDIATION_STRATEGY.md §0 (rules of engagement) and §2.6 (definition of done) before starting. Read the full finding text in docs/CODE_QUALITY_REVIEW.md. One finding, one branch, one PR. Re-verify on main first — cited line numbers are from 2026-07-05 and drift.

**Finding:** readiness probe lies (Part 1 shared-#5, confirmed from the deploy side in Part 2) | **Wave 3** | **Recipe:** R6 (strategy §2.5)
**Touches:** cmd/selfhost health wiring, .github/workflows/docker-test.yml

## Problem
`addDB` is called for only **3 of the 6** self-host databases. Outages of users, explore-recommender or explore-fetcher still report `healthy`. Worse, the CI smoke test in `docker-test.yml` curls that same endpoint — so CI reports a **false green** while half the system is down.

## What to do
1. Test first: take one of the three unchecked DBs offline and assert `/health/ready` reports unhealthy. Fails on main.
2. Register all 6 databases with `addDB`.
3. Confirm the compose smoke test in `docker-test.yml` actually fails when a DB is down — otherwise the ratchet is still fake.

## Done when
- `/health/ready` reflects all 6 databases and the CI smoke test detects a downed one.

---

## Re-confirmed by the Cairn Simplification Audit (2026-08-17)

Independently re-verified at HEAD `a6c56a1` and listed under the audit's Tier 1 (correctness & security). No new task was created — this one owns the finding.
**Audit report:** https://claude.ai/code/artifact/286883fb-3f93-49c4-942f-4880251a409f

**One addition to step 3.** The audit found *why* the CI ratchet is fake, and it is worse than "the smoke test doesn't assert enough": the smoke job **never runs at all**. `.github/workflows/docker-test.yml:326-329` declares `needs: [build-selfhost]` but gates on `if: needs.changes.outputs.selfhost == 'true'` — `changes` is absent from `needs:`, so that expression resolves to empty and the condition can never be true.

So step 3 ("confirm the compose smoke test actually fails when a DB is down") cannot be satisfied until that job is fixed. That fix is tracked separately as **task_7722**. Sequence task_7722 first, or this task's ratchet remains unverifiable.


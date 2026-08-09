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


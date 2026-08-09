---
id: task_4f8a
title: [P2-C4] Decide the prod deploy story: fix infrastructure/docker/prod or delete it (needs owner decision)
type: task
status: open
priority: 2
labels: [quality,wave3,infra,decision]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-09T06:53:56Z
updated_at: 2026-08-09T06:53:56Z
---
Read docs/QUALITY_REMEDIATION_STRATEGY.md §0 (rules of engagement) and §2.6 (definition of done) before starting. Read the full finding text in docs/CODE_QUALITY_REVIEW.md. One finding, one branch, one PR. Re-verify on main first — cited line numbers are from 2026-07-05 and drift.

**Finding:** P2-C4 | **Wave 3** | **Needs an owner decision — do not implement before it is made.**
**Touches:** infrastructure/docker/prod/, .github/workflows/docker-build-*.yml, scripts/init-vault-prod.sh, dev compose

## Problem
The multi-container production deployment is not shippable as checked in. Three stacked, independently verified blockers:

1. All 10 `docker-build-*.yml` workflows that publish images have `build-and-push: if: false`, so `prod/docker-compose.yml` has **no images to pull**.
2. `init-vault-prod.sh` never creates AppRoles for content-service or email-ingest, so those two **crash-loop on `os.Exit(1)`** with empty Vault credentials.
3. The dev compose is already broken for content-service — missing `VAULT_*` vars cause `os.Exit(1)` before DB/migrations.

Either the project pivoted to the `selfhost` binary — making `prod/` dead and actively misleading — or the prod path has not been run end-to-end in a while.

## What to do
1. Re-verify all three blockers on current main.
2. Write both options into this task with costs: **(a)** fix `prod/` (unblock image publishing, add the two AppRoles, repair dev compose) or **(b)** delete `prod/` and document selfhost as the only supported path, updating `docs/DEPLOYMENT.md` and `docs/ARCHITECTURE.md`.
3. **Ask the owner and get an explicit answer before writing code.**

## Done when
- The decision is recorded in this task and the repo matches it — no half-supported deployment path is left in tree.


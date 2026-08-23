---
id: task_d910
title: [Docs] .github/workflows/README.md is stale: documents a non-existent docker-build.yml, omits real per-service workflows
type: task
status: open
priority: 3
labels: [quality,docs,ci]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-23T06:34:00Z
updated_at: 2026-08-23T06:34:00Z
---
**Source:** noticed while working task_7722 (PR #342) and flagged by that PR's reviewer as out-of-scope-but-real; not fixed there to keep that PR surgical.

## Problem
`.github/workflows/README.md` no longer matches the workflows directory:
- Documents `docker-build.yml` as *the* image-build workflow (builds "all 7 Docker images" in one matrix job). That file doesn't exist — image builds are now 11 separate per-service `docker-build-*.yml` files (see task_4f8a: 8 of those 9 non-selfhost ones currently have `build-and-push: if: false`, so the README's description of what actually publishes images is doubly wrong).
- Describes `docker-test.yml` as the PR-time build-verification workflow. That file still exists but is disabled at the GitHub Actions platform level (`state: disabled_manually` — see task_4f8a for the confirmed evidence), so this section describes a workflow that never runs.
- Doesn't mention `.github/workflows/selfhost-compose-smoke.yml` (added in task_7722) at all.
- References `ios-testflight.yml` / `TESTFLIGHT_SETUP.md` under "💰 Alternative: EAS Build" — no such workflow file exists in `.github/workflows/` (only `ios-testflight-fastlane.yml` and `ios-testflight-local-build.yml` do).

## What to do
1. Re-verify current workflow files (`ls .github/workflows/`) against the README's claims.
2. Rewrite the "Workflows" section to describe the actual per-service `docker-build-*.yml` files (and their `if: false` state, or its resolution once task_4f8a lands) plus `selfhost-compose-smoke.yml`.
3. Remove or correct the `docker-build.yml` and `ios-testflight.yml` references.
4. Likely sequence this after task_4f8a lands (so the README describes the final, decided state rather than needing a second pass).

## Done when
- The README accurately lists every file actually in `.github/workflows/` and what triggers/does run, with no reference to nonexistent files.
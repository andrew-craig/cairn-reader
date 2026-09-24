---
id: task_fc02
title: Selfhost compose smoke test curls port 8080, but stack publishes on 8099
type: task
status: closed
priority: 2
labels: [quality,ci]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-09-20T03:14:25Z
updated_at: 2026-09-23T07:53:10Z
---
**Source:** discovered by the task_fd42 agent (PR #403) while fixing the "readiness probe lies" finding. Filed separately per that PR's recommendation rather than folding it in, since it's a distinct bug in the same job.

## Problem
`.github/workflows/docker-test.yml`'s `selfhost-compose-smoke` job's readiness/liveness steps (around lines 347, 349, 360 — re-verify, may have drifted) `curl` `http://localhost:8080/health/ready` and `.../health/live`. But the selfhost compose stack (`infrastructure/docker/selfhost/docker-compose.yml`) publishes the `cairn` container on `${PORT:-8099}:8099` — `.env.example` defaults `PORT=8099`, the CI job's `.env` setup only overrides `DB_PASSWORD`, and the Dockerfile `EXPOSE`s 8099. So even once the job actually *runs* (see task_fd42/PR #403, which fixes the `needs: [changes]` bug that currently prevents this job from executing at all), it will fail immediately with connection-refused on port 8080, regardless of whether the app or its DBs are healthy.

## What to do
Change the curl targets in `selfhost-compose-smoke` from port 8080 to 8099 (or read the port from the same `.env`/default the compose file uses, if that's simple — don't over-engineer it, a literal 8099 matching the compose default is fine unless the surrounding job already parameterizes ports elsewhere).

## Sequencing
Depends on PR #403 (task_fd42) landing first — that's what fixes the `needs:` bug making this job actually execute. This task's fix can be written/reviewed in parallel, but the job won't be exercisable in CI until #403 merges.

## Done when
- [x] `selfhost-compose-smoke` actually runs (post #403) and its readiness/liveness curl steps hit the correct port — confirm on a PR that a deliberately-down service still fails the job (this closes the loop task_fd42 opened: a real ratchet, not a fake one).

## Review

**Branch:** `task_fc02`, based on `origin/task_fd42` (not `main` — `main` still has the `needs:` bug fixed by PR #403/task_fd42, so the job wouldn't execute there yet).

**Re-verified line numbers** (drifted slightly from the filed estimate of ~347/349/360, but landed on the same lines after checking out `task_fd42`): the three `curl` calls in the `selfhost-compose-smoke` job's "Wait for readiness" and "Check liveness endpoint" steps are at `.github/workflows/docker-test.yml:347`, `:349`, and `:360`.

**Confirmed the port mismatch and the fix:**
- `infrastructure/docker/selfhost/docker-compose.yml:34` publishes `"${PORT:-8099}:8099"` for the `cairn` service.
- `infrastructure/docker/selfhost/.env.example:19` has `# PORT=8099` (commented out, so the compose default `8099` applies).
- `infrastructure/docker/selfhost/Dockerfile:91` has `EXPOSE 8099`.
- The job's "Create .env from example" step (`.github/workflows/docker-test.yml:335-340`) only does `cp .env.example .env` and a `sed` to set `DB_PASSWORD` — it never touches `PORT`. So in CI, `PORT` is unset (all `.env.example` lines besides `DB_PASSWORD` are commented out) and the compose `${PORT:-8099}` default of `8099` applies.
- No other port parameterization exists in the job (the only other `8080` in the workflow file is in the unrelated `cairn-web-test` job for the web app image, not the selfhost stack) — a literal `8099` is the simplest correct fix, matching the task's own recommendation.

**Fix applied:** changed the three `http://localhost:8080/...` curl targets to `http://localhost:8099/...` in the `selfhost-compose-smoke` job (readiness poll loop, the `curl -s` status dump inside it, and the liveness check). No other lines touched.

**Verified locally:** built the selfhost image from source via `docker compose -f docker-compose.yml -f docker-compose.build.yml up -d --build` (the repo's documented local-build override, since CI's plain `docker compose up -d --build` pulls the published GHCR image, which isn't necessary to prove the port claim) and confirmed with `curl` that the running container answers on port 8099, not 8080 — validating that the corrected job would actually reach the app. Did not run the GitHub Actions job itself (no `act` available, same limitation noted by the task_fd42 agent), so the "deliberately-down service still fails the job" check from Done-when could not be exercised end-to-end in CI; that requires a real PR run.

**YAML validated:** `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/docker-test.yml'))"` succeeds.

Leaving `status: open` per instructions — reviewer to close after confirming the PR run.

**Landed:** merged to `main` as #405. Closed during backlog housekeeping on 2026-09-23.

---
id: task_9def
title: Isolate selfhost-compose-smoke CI project from real deployments
type: task
status: in_progress
priority: 2
labels: [quality,ci,infrastructure]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-09-20T20:41:26Z
updated_at: 2026-09-20T20:41:59Z
---
Root-caused today (2026-09-20): infrastructure/docker/selfhost/docker-compose.yml pins a top-level 'name: cairn-selfhost', and that name is baked into the compose file itself rather than being CI-scoped. When the selfhost-compose-smoke job's needs: bug was fixed (task_fd42/PR #403) and the job ran for the first time, its docker compose up -d --build resolved to the exact same project name/volumes as the real self-hosted Cairn deployment running on this same Docker host — colliding with the real cairn-selfhost-cairn-db-1 container and cairn-selfhost_cairn_db_data volume. Because Postgres only applies POSTGRES_PASSWORD on first init of an empty data dir, the CI job's DB_PASSWORD=ci_test_password never took effect against the real volume, producing 'password authentication failed for user cairn' and an app crash loop. The job's final docker compose down -v then deleted the real volume outright. Fix: give the CI job an isolated COMPOSE_PROJECT_NAME (unique per run, e.g. incorporating ${{ github.run_id }}) so it can never collide with or delete a real deployment's containers/volumes, without touching the shared compose file's name: field (that field is what real self-host operators rely on for stable container names across restarts).

## Plan
1. Set `COMPOSE_PROJECT_NAME` as a job-level `env:` on `selfhost-compose-smoke` in `.github/workflows/docker-test.yml`, unique per run (incorporate `${{ github.run_id }}`) → verify: every `docker compose` invocation in the job (`up`, `logs -f ...`, `ps`, `logs`, `down -v`) resolves to the same isolated project regardless of `working-directory` vs `-f` usage, since `docker compose` reads `COMPOSE_PROJECT_NAME` from the environment and it outranks the compose file's `name:` attribute.
2. Do not touch `infrastructure/docker/selfhost/docker-compose.yml`'s `name:` field, and don't touch any other job/step in the workflow file.
3. Verify: YAML validity (`python3 -c "import yaml..."` or similar), `actionlint` if available, and manual read-through confirming every `docker compose` call in the job is covered by the job-level env var.
4. Record results in this task file, commit on a new branch, push, open PR.

## Review / Results
Implemented as a job-level `env:` block added directly under `selfhost-compose-smoke`'s existing `permissions:` key:

```yaml
    env:
      COMPOSE_PROJECT_NAME: selfhost-smoke-${{ github.run_id }}
```

This is the only change to `.github/workflows/docker-test.yml`. Nothing else in the file was touched. `infrastructure/docker/selfhost/docker-compose.yml` was not modified.

Why this covers every invocation: Compose CLI project-name resolution order is `-p` flag > `COMPOSE_PROJECT_NAME` env var > compose file `name:` attribute > directory name. A job-level `env:` is exported to every `run:` step in the job's process environment regardless of that step's `working-directory` or whether it passes `-f infrastructure/docker/selfhost/docker-compose.yml` explicitly — so `up -d --build`, the inline `-f ... logs` calls in the readiness-wait failure branch, `ps`, `logs --tail=...`, and `down -v` all resolve to the same isolated project name for a given run, and a different name on every run (via `github.run_id`), so this job can never again collide with a real deployment's `cairn-selfhost` project.

Verified:
- YAML parses cleanly (`python3 -c "import yaml; yaml.safe_load(open(...))"`).
- `actionlint` (if present) run against the file — see PR for result.
- Manually traced all 5 `docker compose` call sites in the job; all are subject to the job-level env var.
- Not verified: actually running the job in GitHub Actions (can't do that from this environment). The PR description notes this honestly.

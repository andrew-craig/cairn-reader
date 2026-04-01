---
id: task_6f3a
title: Add integration test workflow with Docker/Postgres
type: task
status: open
priority: 3
labels: [ci,backend,testing]
blocked_by: []
parent: epic_d014
created_at: 2026-04-01T03:38:21Z
updated_at: 2026-04-01T03:38:21Z
---
Create a separate workflow (or job) for integration tests that require Docker and PostgreSQL. This is more complex than standalone checks and should be tackled after the P1/P2 checks are in place.

Services with integration tests:
- users: test/integration/docker-compose.test.yml (Postgres + Vault)
- explore: recommender/scripts/setup_test_db.sh (Postgres)
- read: no explicit integration test setup yet

Options:
1. Use GitHub Actions service containers for Postgres
2. Use docker compose in CI (heavier but matches local dev)
3. Only run on specific labels or schedule (not every PR)

Tooling status: Docker Compose test configs exist for users. Explore has a test DB setup script. Needs CI orchestration.
Runs standalone: NO - requires Docker and Postgres.

Consider: run only on PRs with a specific label, or on pushes to main, to avoid slowing down every PR.

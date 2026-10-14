---
id: bug_a701
title: Self-host and read-service config drift (env forwarding, DB names, Makefile)
type: bug
status: open
priority: 2
labels: [infra]
blocked_by: []
parent: null
remote_task_url: null
created_at: 2026-10-14T20:28:44Z
updated_at: 2026-10-14T20:28:44Z
---
Found in docs review chore_65f1.
- infrastructure/docker/selfhost/docker-compose.yml has no env_file, so .env settings JWT_*_EXPIRY, INTERNAL_API_KEY, BCRYPT_COST, LOG_*, WEB_DIR never reach the container.
- Fetcher DB_NAME defaults differ: API cairn_rss, worker cairn_rss_fetcher, compose rss_fetcher_service.
- services/read/Makefile: migrate targets hardcode cairn_content and cairn_rss. The up/down/logs/clean targets run docker compose with no compose file in that directory.
- services/read/scripts/run-integration-tests.sh refers to content-db and rss-db containers, which don't exist.
- Email INGEST_API_KEY is required at startup in multi-container mode but only selfhost uses it (it seeds api_keys). Make it optional outside selfhost, or document why it's required.
- .github/workflows/README.md describes a docker-build.yml that no longer exists and says 7 images are built.

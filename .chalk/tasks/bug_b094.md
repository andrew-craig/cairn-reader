---
id: bug_b094
title: DB SSL mode env var name differs between processes; prod and dev compose each set the wrong one
type: bug
status: open
priority: 1
labels: [infra,read]
blocked_by: []
parent: epic_6e4d
remote_task_url: null
created_at: 2026-10-13T20:28:03Z
updated_at: 2026-10-13T20:39:47Z
---
The DB SSL mode env var has two names, and the compose files get it wrong in both directions.

- Readers of DB_SSLMODE (default require): the shared pkg/config loader used by the Content and Fetcher (ingest-rss) APIs. The User service reads DB_SSLMODE too (default disable).
- Readers of DB_SSL_MODE (default disable): the content worker (services/read/content/cmd/worker/main.go:121), the ingest_rss worker (services/read/fetcher/cmd/ingest_rss_worker/main.go:195) and the Email Ingest service (API and worker).

Compose mismatches:
- prod compose sets DB_SSLMODE on content-worker and ingest-rss-worker, which read DB_SSL_MODE, so it is ignored (harmless today, both default to disable).
- dev compose sets DB_SSL_MODE on content-service and ingest-rss, which read DB_SSLMODE, so it is ignored and the APIs connect with sslmode=require. The bundled cairn-db has no TLS, so these APIs fail to connect at startup in dev (migrations run on startup). The dev compose has no env_file and the dev .env.example has no SSL variable, so nothing else sets DB_SSLMODE.

Pick one name (DB_SSLMODE via pkg/config), update the workers and the email service, fix both compose files, and update docs/CONFIGURATION.md and docs/DEPLOYMENT.md, which currently document the mismatch.

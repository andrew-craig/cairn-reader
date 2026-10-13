---
id: bug_b094
title: Workers read DB_SSL_MODE but prod/dev compose set DB_SSLMODE
type: bug
status: open
priority: 2
labels: [infra,read]
blocked_by: []
parent: epic_6e4d
remote_task_url: null
created_at: 2026-10-13T20:28:03Z
updated_at: 2026-10-13T20:28:03Z
---
The content worker (services/read/content/cmd/worker/main.go:121) and ingest_rss worker (services/read/fetcher/cmd/ingest_rss_worker/main.go:195) read DB_SSL_MODE, while the APIs use pkg/config DB_SSLMODE (default require). The compose files set DB_SSLMODE on the workers, so the setting is silently ignored (both default to disable today). Pick one name (DB_SSLMODE via pkg/config), update the workers and compose files, and update docs/CONFIGURATION.md, which currently documents the mismatch.

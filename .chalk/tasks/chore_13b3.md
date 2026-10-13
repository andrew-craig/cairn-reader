---
id: chore_13b3
title: Users service: remove unused Redis config and VAULT_DB_CREDS_PATH
type: chore
status: open
priority: 3
labels: [users]
blocked_by: []
parent: epic_6e4d
remote_task_url: null
created_at: 2026-10-13T20:28:03Z
updated_at: 2026-10-13T20:28:03Z
---
services/users config parses REDIS_* and VAULT_DB_CREDS_PATH but nothing uses them. Remove the dead config (and their docs entries in docs/CONFIGURATION.md and the .env examples).

---
id: bug_2f87
title: Dev compose: users service sets SERVER_ENVIRONMENT but config reads ENVIRONMENT
type: bug
status: closed
priority: 3
labels: [infra,users]
blocked_by: []
parent: epic_6e4d
remote_task_url: null
created_at: 2026-10-13T21:22:21Z
updated_at: 2026-10-13T21:23:14Z
---
infrastructure/docker/dev/docker-compose.yml sets SERVER_ENVIRONMENT=development on user-service, but services/users/internal/config reads ENVIRONMENT, so the setting is ignored (it is harmless only because the default is development). infrastructure/docker/README.md documents the wrong name too. Rename both to ENVIRONMENT.

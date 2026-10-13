---
id: chore_35cf
title: Users CLAUDE.md drift (refresh token lifetime etc.)
type: chore
status: open
priority: 3
labels: [docs,users]
blocked_by: []
parent: epic_6e4d
remote_task_url: null
created_at: 2026-10-13T20:28:03Z
updated_at: 2026-10-13T20:28:03Z
---
services/users/CLAUDE.md says the refresh token lifetime is 7 days; the code default is 30. Sweep the rest of the file against the code (config defaults, endpoints, schema) and fix the docs. The code is the source of truth.

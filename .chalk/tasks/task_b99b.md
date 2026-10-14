---
id: task_b99b
title: Align token lifetime and rate-limit defaults across standalone, self-host and dev compose
type: task
status: open
priority: 2
labels: [users,config]
blocked_by: []
parent: null
remote_task_url: null
created_at: 2026-10-14T20:28:44Z
updated_at: 2026-10-14T20:28:44Z
---
Owner decision 2026-10-15 (chore_65f1): align the defaults. Today the standalone users service defaults to 60m access, 30d refresh and 100 req/min (services/users/internal/config/config.go:108-116). Self-host defaults to 15m, 7d and 10/min (cmd/selfhost/config.go:94-95, services/users/selfhost/users.go:140), and the dev compose uses 15m/7d. Password complexity is on in standalone and off in self-host (selfhost/users.go:112). Pick one set in code, then update docs/CONFIGURATION.md, the users openapi.yaml, docs/ARCHITECTURE.md and docs/architecture/http-frameworks.md, which currently describe both.

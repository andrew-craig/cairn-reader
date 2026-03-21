---
id: bug_4b52
title: Remove hardcoded DB password fallback in dev docker-compose
type: bug
status: open
priority: 2
labels: []
blocked_by: []
parent: null
created_at: 2026-03-21T04:20:36Z
updated_at: 2026-03-21T04:20:36Z
---
infrastructure/docker/dev/docker-compose.yml has a weak fallback that gets used if .env is missing. Remove the default so compose fails fast.

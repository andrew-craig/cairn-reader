---
id: task_d23e
title: Align Gin middleware patterns across services
type: task
status: open
priority: 1
labels: []
blocked_by: []
parent: null
created_at: 2026-03-21T04:20:36Z
updated_at: 2026-03-21T04:20:36Z
---
User Service uses Gin with pkg/auth.GinMiddleware while Read/Explore services use net/http with pkg/auth.Middleware. Both use the same underlying validator but have different middleware patterns.

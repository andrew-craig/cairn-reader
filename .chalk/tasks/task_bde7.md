---
id: task_bde7
title: Apply security headers to all services (CORS not needed)
type: task
status: in_progress
priority: 3
labels: [middleware, security]
blocked_by: []
parent: task_d23e
created_at: 2026-03-28T01:54:21Z
updated_at: 2026-03-28T18:56:56Z
---

## Description

~~Apply CORS middleware to public-facing services.~~

**Update**: The only client is a React Native mobile app, which uses native HTTP clients — not browser fetch. CORS is a browser-only security mechanism and is **not needed**. The existing CORS middleware in User Service can remain for potential future web clients, but no other service needs it.

**Remaining work**: Apply security headers (`SecureHeadersRelaxed`) to all services. These headers (X-Content-Type-Options, X-Frame-Options, etc.) are defense-in-depth and worth having regardless of client type.

## Requirements

- Apply `SecureHeadersRelaxed` middleware to Read Content, Explore Recommender, and Explore Fetcher
- Do NOT add CORS middleware to services that don't already have it (not needed for native mobile clients)
- Keep User Service CORS as-is (no change)

## Acceptance Criteria

- [ ] All services have security headers middleware
- [ ] No unnecessary CORS middleware added

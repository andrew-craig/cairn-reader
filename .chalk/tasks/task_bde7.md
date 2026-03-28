---
id: task_bde7
title: Apply CORS and security headers to public-facing services
type: task
status: open
priority: 3
labels: [middleware, security]
blocked_by: [task_6995]
parent: task_d23e
created_at: 2026-03-28T01:54:21Z
updated_at: 2026-03-28T01:54:21Z
---

## Description

Only the User Service currently applies CORS and security headers. Public-facing services (Read Content, Explore Recommender) that serve browser clients need these too. Internal-only services (Explore Fetcher, Read Fetcher) likely don't need CORS but should still have security headers.

## Requirements

- Apply CORS middleware to Read Content and Explore Recommender (they serve browser clients via the mobile app's API calls)
- Apply security headers (`SecureHeadersRelaxed`) to all services
- Make CORS origins configurable per service via environment variables
- Internal-only services should skip CORS but still use security headers

## Open Questions

- Does the mobile app make direct API calls to Read/Explore, or does it go through a gateway? This determines which services need CORS.
- Are there any browser-based admin tools that need CORS access?

## Acceptance Criteria

- [ ] Public-facing services have CORS middleware
- [ ] All services have security headers
- [ ] CORS origins are configurable

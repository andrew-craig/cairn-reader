---
id: chore_35cf
title: Users CLAUDE.md drift (refresh token lifetime etc.)
type: chore
status: closed
priority: 3
labels: [docs,users]
blocked_by: []
parent: epic_6e4d
remote_task_url: null
created_at: 2026-10-13T20:28:03Z
updated_at: 2026-10-13T21:05:05Z
---
services/users/CLAUDE.md says the refresh token lifetime is 7 days; the code default is 30. Sweep the rest of the file against the code (config defaults, endpoints, schema) and fix the docs. The code is the source of truth.

## Plan
- [x] Fix refresh token lifetime (7 -> 30 days) and access token lifetime (15m -> 60m default) everywhere in services/users/CLAUDE.md
- [x] Env var section: use code defaults, add missing vars, fix invalid `7d` duration (720h), bcrypt range 10-14
- [x] Ports (container 8080 / host 8082), Makefile comments, directory tree
- [x] DB schema vs migrations (NOT NULLs, constraints, unique token_hash, extra indexes, email_verification_tokens columns)
- [x] Endpoints vs router; response envelope and RequireHTTPS note; account lockout; examples (migration numbers, repository interface, handler pattern, route registration)
- [x] Redis/rate limiter wording; account-type "password recovery" claim
- [x] Record openapi/code mismatches (not fixed)

## Review
Docs-only change to services/users/AGENTS.md (CLAUDE.md is a symlink to it). Code is the source of truth; no code touched.

Fixes: refresh token lifetime 7d -> 30d (also Security Best Practices); access token 15m -> 60m default; env var block rewritten from config.go (added DB pool, Vault AppRole, key rotation, SHUTDOWN_TIMEOUT, LOG_* vars; `JWT_REFRESH_TOKEN_EXPIRY=7d` was not even a valid Go duration, now 720h); bcrypt range is 10-14 not "minimum 12"; port is 8080 (8082 only the compose host mapping); register/get-user response examples now match the real shape inside the `{data, meta}` envelope; documented that all /api/v1 routes require TLS or X-Forwarded-Proto: https; documented 5-failure/15-minute lockout; schema block now matches migrations (constraints, NOT NULLs, unique token_hash, idx_users_created_at, idx_refresh_tokens_revoked_at, email_verification_tokens columns); email-only accounts no longer claim password recovery; directory tree gained cmd/migrate, errors.go, limits.go, email_verification_service.go, audit.go, validation/; new-migration examples use 000009; repository interface / handler / route snippets match real names; Redis wording clarified (pkg/middleware has a Redis limiter but users does not wire it up); sqlmock mention removed (not a dependency).

openapi/code mismatches found (NOT fixed):
- openapi has no entries for GET/POST /api/v1/auth/verify-email or POST /api/v1/auth/resend-verification (router has them).
- AuthResponse describes access token as 15 min default (example expires_in 900) and refresh token as 7 days; code defaults are 60 min / 30 days.
- Success responses in openapi are not wrapped in the `{data, meta}` envelope the handlers emit (pkg/api WriteSuccess); error shape (`error`, `message`, `details`, `meta`) also differs from the spec's `Error` schema (`error` only).
- openapi does not mention the RequireHTTPS 403 on /api/v1.
Other code/config drift noticed: infrastructure/docker/dev/docker-compose.yml sets SERVER_ENVIRONMENT=development, but users config reads ENVIRONMENT; RouterConfig comments say auth rate limit default is 10 while main.go passes the config default of 100.

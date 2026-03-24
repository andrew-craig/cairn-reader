---
id: bug_515f
title: Add authentication to internal API routes
type: bug
status: closed
priority: 2
labels: []
blocked_by: []
parent: null
created_at: 2026-03-21T04:20:36Z
updated_at: 2026-03-21T04:20:36Z
---
Internal route /api/v1/internal/content/user/bulk in content service has no authentication. Need service-to-service auth.

## Analysis

### Root Cause
The internal API route `/api/v1/internal/content/user/bulk` in the content service has no authentication middleware applied. Any client with network access can call this endpoint to add content to any user's reading list. Currently, only Docker network isolation prevents external access.

### Additional Issue Found
The fetcher client (`content_service_client.go:263`) calls `/api/v1/users/bulk/contents` but the router defines the endpoint at `/api/v1/internal/content/user/bulk` — this is a URL mismatch that would result in 404s.

### Affected Files
- `services/read/content/internal/api/router.go` — internal route group (line 110-113)
- `services/read/content/internal/api/handlers/bulk_handler.go` — BulkAddToUsersInternal handler
- `services/read/fetcher/internal/client/content_service_client.go` — fetcher HTTP client
- `services/read/content/internal/config/config.go` — content service config
- `services/read/fetcher/internal/config/config.go` — fetcher config
- `pkg/auth/middleware.go` — shared auth middleware
- `infrastructure/docker/dev/docker-compose.yml` — dev compose
- `infrastructure/docker/prod/docker-compose.yml` — prod compose
- `infrastructure/docker/dev/.env.example` — env example

## Plan

Approach: API key-based authentication via `X-Internal-API-Key` header. Simple, appropriate for service-to-service communication within a trusted network.

- [x] Add `RequireInternalAPIKey` middleware to `pkg/auth/internal_auth.go`
- [x] Add `InternalAPIKey` field to content service config
- [x] Wire middleware into content service router for internal routes
- [x] Pass API key config through `main.go` → `NewRouter()`
- [x] Update fetcher client to send `X-Internal-API-Key` header
- [x] Add `InternalAPIKey` + `ContentServiceURL` to fetcher config
- [x] Fix URL mismatch: fetcher calls wrong endpoint path
- [x] Add `INTERNAL_API_KEY` env var to both docker-compose files and .env.example
- [x] Write tests for internal auth middleware
- [x] Run existing tests to verify no regressions

## Review

### Changes Made

**New files:**
- `pkg/auth/internal_auth.go` — `InternalAuthMiddleware` with `RequireInternalAPIKey` method using constant-time comparison
- `pkg/auth/internal_auth_test.go` — Tests for valid key, missing key, invalid key, empty key

**Modified files:**
- `services/read/content/internal/api/router.go` — Applied `RequireInternalAPIKey` middleware to `/api/v1/internal` route group; added `internalAuthMiddleware` parameter to `NewRouter`
- `services/read/content/internal/config/config.go` — Added `InternalAPIKey` field, loaded from `INTERNAL_API_KEY` env var, required validation
- `services/read/content/cmd/content/main.go` — Creates `InternalAuthMiddleware` and passes to router
- `services/read/fetcher/internal/client/content_service_client.go` — Added `InternalAPIKey` to config/client, sends `X-Internal-API-Key` header on all requests; fixed URL from `/api/v1/users/bulk/contents` to `/api/v1/internal/content/user/bulk`
- `services/read/fetcher/cmd/ingest_rss_worker/main.go` — Reads `INTERNAL_API_KEY` env var, passes to client config
- `services/read/fetcher/internal/client/content_service_client_test.go` — Updated URL assertion to match corrected path
- `infrastructure/docker/dev/docker-compose.yml` — Added `INTERNAL_API_KEY` to content-service and ingest-rss-worker
- `infrastructure/docker/prod/docker-compose.yml` — Added `INTERNAL_API_KEY` to content-service and ingest-rss-worker
- `infrastructure/docker/dev/.env.example` — Added `INTERNAL_API_KEY` placeholder
- `infrastructure/docker/prod/.env.example` — Added `INTERNAL_API_KEY` placeholder

### Verification
- All auth package tests pass (including 4 new internal auth tests)
- All content service handler tests pass
- All fetcher client tests pass (including updated URL path assertion)
- Both services compile cleanly
- Pre-existing failures (external URL integration tests, feed worker stop test) are unrelated

---
id: task_d23e
title: Standardize middleware patterns across all services
type: epic
status: open
priority: 1
labels: [middleware, architecture, consistency]
blocked_by: []
parent: null
created_at: 2026-03-21T04:20:36Z
updated_at: 2026-03-28T00:00:00Z
---

## Problem Statement

The three backend services (User, Read, Explore) have inconsistent middleware patterns, leading to duplicated code, missing cross-cutting concerns, and maintenance burden.

**Note**: The original task description mentioned "Gin middleware" but this was inaccurate. No service uses Gin — all services use `net/http` compatible middleware. The real issue is middleware inconsistency across services.

## Current State Analysis

### Router Framework Inconsistency

| Service | Router | Middleware Chain |
|---------|--------|-----------------|
| User Service | chi v5 | ✅ Full chain |
| Read Content | chi v5 | ✅ Full chain |
| Read Fetcher | chi v5 | ✅ Inline |
| Read Email | chi v5 | ✅ Full chain |
| Explore Fetcher | **http.NewServeMux()** | ❌ No chain |
| Explore Recommender | chi v5 | ✅ Full chain |

**Explore Fetcher** is the only service using stdlib `http.NewServeMux()` instead of chi, meaning it cannot use the standard middleware chain pattern.

### Cross-Cutting Concern Coverage

| Middleware | User Svc | Read Content | Read Fetcher | Read Email | Explore Fetcher | Explore Recommender |
|-----------|----------|-------------|-------------|-----------|----------------|-------------------|
| Panic Recovery | ✅ (6 variants) | ✅ | ✅ | ? | ❌ | ? |
| Request Logging | ✅ | ✅ | ? | ? | ❌ | ? |
| CORS | ✅ (full) | ❌ | ❌ | ❌ | ❌ | ❌ |
| Security Headers | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| Rate Limiting | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| JWT Auth | ✅ | ✅ | N/A | ✅ | N/A | ✅ |
| JSON Validation | ❌ | ✅ | ❌ | ❌ | ❌ | ❌ |

### Key Problems

1. **Explore Fetcher uses stdlib mux** — cannot participate in middleware chain patterns, no method routing, middleware applied inline
2. **User Service middleware is not shared** — CORS, security headers, recovery, rate limiting all live in `services/users/internal/middleware/` instead of a shared package
3. **Missing panic recovery in Explore Fetcher** — unhandled panics will crash the service
4. **No CORS in Read/Explore** — will fail for any browser-based clients
5. **No security headers in Read/Explore** — missing HSTS, X-Frame-Options, etc.
6. **Duplicated middleware code** — recovery middleware reimplemented in User Service, Read Content, and Read Fetcher separately
7. **No request logging in Explore Fetcher** — blind spot for debugging

### Shared Auth Package (pkg/auth)

The `pkg/auth` package is already well-designed and shared across services:
- `Middleware` (JWT auth) — used by User, Read Content, Read Email, Explore Recommender
- `InternalAuthMiddleware` (API key auth) — used by Read Content
- Context utilities (`GetUserIDFromContext`, etc.) — consistent across all services

This is the model for how other middleware should be organized.

## Solution Direction

Extract common middleware from User Service into a shared `pkg/middleware` package, migrate Explore Fetcher to chi router, and apply consistent middleware stacks across all services.

### Target Architecture

```
pkg/middleware/           # Shared middleware (new)
├── recovery.go          # Panic recovery (from User Service)
├── cors.go             # CORS handling (from User Service)
├── security.go         # Security headers (from User Service)
└── rate_limit.go       # Rate limiting (from User Service)

pkg/auth/               # Already shared — no changes needed
├── middleware.go       # JWT auth
├── internal_auth.go    # API key auth
└── ...

pkg/logging/            # Already shared — no changes needed
└── ...                 # ChiRequestLogger
```

## CORS Analysis (Resolved)

The mobile app (React Native) is the **only client**. All services sit behind the same domain (`cairn.seatrain.net`) with path-based routing via a reverse proxy. Since React Native uses native HTTP clients (not browser fetch), **CORS is not needed** — it's purely a browser security mechanism. The existing CORS middleware in User Service can stay for future web client support, but there's no urgency to add it to other services.

## Risks & Considerations

- **Explore Fetcher chi migration**: Low risk, but need to preserve all existing routes and behavior
- **Shared middleware extraction**: Must not break User Service when moving code out
- **Rate limiting scope**: Only auth endpoints need rate limiting currently, but the middleware should be available for all services

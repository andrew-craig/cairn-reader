# API Standardization and Migration Plan

**Status**: Draft
**Last Updated**: 2026-01-01
**Owner**: Engineering

---

## Executive Summary

This document outlines the plan to standardize API endpoints across all Cairn backend services. Currently, services use **three different API design patterns**, creating inconsistency for API consumers and complicating deployment/routing.

**Goals**:
1. Establish consistent API versioning across all services
2. Standardize path structure, naming conventions, and HTTP methods
3. Improve API discoverability and documentation
4. Enable clean service segregation for deployment/routing

**Approach**: Single-phase migration (no existing users to migrate)

---

## Current State Analysis

### Service Overview

| Service | Port | Versioning | Prefix | Framework | Endpoints |
|---------|------|------------|--------|-----------|-----------|
| Explore Fetcher | 8080 | ❌ None | None | stdlib | 2 |
| Explore Recommender | 8087 | ❌ None | `/explore` | stdlib | 7 |
| User Service | 8082 | ❌ None | `/auth`, `/user` | Gin | 11 |
| Read Content | 8083 | ✅ v1 | `/api/v1` | Chi | 12 |
| Read Ingest RSS | 8085 | ✅ v1 | `/api/v1` | Chi | 4 |

### Key Inconsistencies

#### 1. **API Versioning**
- **Problem**: Only Read services use versioning (`/api/v1`)
- **Impact**: Breaking changes will affect clients, no migration path

#### 2. **Path Structure**
```
Current State:
- Explore Fetcher:    /fetch
- Explore Recommender: /explore/recommendation/{userID}
- User Service:       /auth/register
- Read Content:       /api/v1/user/{user_id}/content
- Read Ingest RSS:    /api/v1/user/{user_id}/feed/subscription
```

#### 3. **Health Check Endpoints**
```
Explore Fetcher:      /health (liveness only)
Explore Recommender:  /health, /ready
User Service:         /health, /ready
Read Content:         /health (includes DB check)
Read Ingest RSS:      /health (includes DB check)
Workers:              /health/live, /health/ready
```

#### 4. **Path Parameter Naming**
- Inconsistent: `{id}`, `{user_id}`, `{userID}`, `{articleID}`, `{content_id}`
- No standard for snake_case vs camelCase

#### 5. **Resource Naming**
- Inconsistent use of singular vs plural resource names

---

## Target API Structure

### Standard API Pattern

All services will adopt the following structure:

```
/health/live                          → Liveness check (simple)
/health/ready                         → Readiness check (includes dependencies)
/api/v1/{service}/{resource}          → Versioned API endpoints
```

### Service-Specific Prefixes

Each service will use a unique prefix under `/api/v1`:

| Service | Current Port | API Prefix | Example Endpoint |
|---------|-------------|------------|------------------|
| Explore Fetcher | 8080 | `/api/v1/explore/feed` | `/api/v1/explore/feed/fetch` |
| Explore Recommender | 8081 | `/api/v1/explore` | `/api/v1/explore/recommendation/{user_id}` |
| User Service | 8082 | `/api/v1/auth`, `/api/v1/user` | `/api/v1/auth/register` |
| Read Content | 8083 | `/api/v1/content` | `/api/v1/content/{content_id}` |
| Read Ingest RSS | 8085 | `/api/v1/source/rss` | `/api/v1/source/rss/user/{user_id}/subscription` |

**Rationale**:
- Clear service boundaries for routing/deployment
- Enables future API gateway implementation (Content Service will act as gateway to source services)
- Consistent versioning strategy across all services
- `/source/*` prefix used for content source services (RSS, webhooks, integrations) not directly exposed to users
- Content Service will aggregate and serve content from multiple source services

---

## Detailed Migration by Service

### 1. Explore Fetcher Service

#### Current Endpoints
```
GET  /health              → Health check
POST /fetch               → Manual fetch trigger
```

#### Target Endpoints (v1)
```
GET  /health/live                      → Liveness check
GET  /health/ready                     → Readiness check (DB)
POST /api/v1/explore/feed/fetch        → Manual fetch trigger
```

#### Migration Steps
1. Add new `/api/v1/explore/feed/*` routes
2. Split `/health` into `/health/live` and `/health/ready`
3. Update OpenAPI spec
4. Update mobile app to use new paths

#### Breaking Changes
- `/fetch` → `/api/v1/explore/feed/fetch`
- `/health` → `/health/live` + `/health/ready`

---

### 2. Explore Recommender Service

#### Current Endpoints
```
GET  /health                               → Liveness
GET  /ready                                → Readiness
POST /explore/article                      → Submit articles (internal)
GET  /explore/recommendation/{userID}      → Get recommendations
POST /explore/article/read                 → Mark as read
POST /explore/article/{articleID}/vote     → Vote on article
DELETE /explore/article/{articleID}/vote   → Remove vote
GET /explore/article/{articleID}/vote      → Get vote counts
```

#### Target Endpoints (v1)
```
GET  /health/live                                 → Liveness
GET  /health/ready                                → Readiness
POST /api/v1/explore/article                      → Submit articles (internal)
GET  /api/v1/explore/recommendation/{user_id}     → Get recommendations
POST /api/v1/explore/article/{article_id}/read    → Mark as read
POST /api/v1/explore/article/{article_id}/vote    → Vote on article
DELETE /api/v1/explore/article/{article_id}/vote  → Remove vote
GET  /api/v1/explore/article/{article_id}/vote    → Get vote counts
```

#### Migration Steps
1. Add new `/api/v1/explore/*` routes
2. Standardize path parameters: `{userID}` → `{user_id}`, `{articleID}` → `{article_id}`
3. Standardize resource naming: `/articles` → `/article`, `/recommendations` → `/recommendation` (singular)
4. Update OpenAPI spec
5. Update mobile app to use new paths

#### Breaking Changes
- Path prefix: `/explore/*` → `/api/v1/explore/*`
- Resource naming: plural → singular (`/article`, `/recommendation`)
- Path parameters: camelCase → snake_case
- Health checks: `/health`, `/ready` → `/health/live`, `/health/ready`

---

### 3. User Service

#### Current Endpoints
```
GET    /health                      → Liveness
GET    /ready                       → Readiness
POST   /auth/register               → Register with email/password
POST   /auth/register/mobile        → Register with device ID
POST   /auth/login                  → Login with email/password
POST   /auth/login/mobile           → Login with device ID
POST   /auth/refresh                → Refresh access token
POST   /auth/logout                 → Logout (revoke token)
POST   /auth/logout-all             → Logout all devices
GET    /user/{id}                   → Get user profile
PATCH  /user/{id}                   → Update user profile
POST   /user/{id}/upgrade           → Upgrade mobile account
DELETE /user/{id}                   → Delete user account
```

#### Target Endpoints (v1)
```
GET    /health/live                        → Liveness
GET    /health/ready                       → Readiness
POST   /api/v1/auth/register               → Register with email/password
POST   /api/v1/auth/register/mobile        → Register with device ID
POST   /api/v1/auth/login                  → Login with email/password
POST   /api/v1/auth/login/mobile           → Login with device ID
POST   /api/v1/auth/refresh                → Refresh access token
POST   /api/v1/auth/logout                 → Logout (revoke token)
POST   /api/v1/auth/logout-all             → Logout all devices
GET    /api/v1/user/{user_id}              → Get user profile
PATCH  /api/v1/user/{user_id}              → Update user profile
POST   /api/v1/user/{user_id}/upgrade      → Upgrade mobile account
DELETE /api/v1/user/{user_id}              → Delete user account
```

#### Migration Steps
1. Add `/api/v1` prefix to all routes
2. Rename path parameter: `{id}` → `{user_id}`
3. Update Gin router configuration
4. Update JWT middleware to work with new routes
5. Update OpenAPI spec
6. Update mobile app to use new paths

#### Breaking Changes
- All routes gain `/api/v1` prefix
- Path parameter: `{id}` → `{user_id}`
- Health checks: `/health`, `/ready` → `/health/live`, `/health/ready`

---

### 4. Read Content Service

#### Current Endpoints
```
GET  /health                                           → Liveness + DB
POST   /api/v1/content                                 → Create content
GET    /api/v1/content/{id}                            → Get content
PUT    /api/v1/content/{id}                            → Update content
POST   /api/v1/content/bulk                            → Bulk create/update
POST   /api/v1/content/check-duplicate                 → Check duplicates
GET    /api/v1/user/{user_id}/content                  → List user contents
POST   /api/v1/user/{user_id}/content                  → Add content to user
GET    /api/v1/user/{user_id}/content/search           → Search user contents
PATCH  /api/v1/user/{user_id}/content/{content_id}     → Update user content
DELETE /api/v1/user/{user_id}/content/{content_id}     → Delete user content
POST   /api/v1/user/bulk/content                       → Bulk add to users
```

#### Target Endpoints (v1)
```
GET    /health/live                                    → Liveness
GET    /health/ready                                   → Readiness (DB)
POST   /api/v1/content                                 → Create content
GET    /api/v1/content/{content_id}                    → Get content
PUT    /api/v1/content/{content_id}                    → Update content
POST   /api/v1/content/bulk                            → Bulk create/update
POST   /api/v1/content/check-duplicate                 → Check duplicates
GET    /api/v1/content/user/{user_id}                  → List user contents
POST   /api/v1/content/user/{user_id}                  → Add content to user
GET    /api/v1/content/user/{user_id}/search           → Search user contents
PATCH  /api/v1/content/user/{user_id}/{content_id}     → Update user content
DELETE /api/v1/content/user/{user_id}/{content_id}     → Delete user content
POST   /api/v1/content/user/bulk                       → Bulk add to users
```

#### Migration Steps
1. ✅ Already uses `/api/v1` prefix (compliant)
2. ✅ Already uses singular resource names (compliant)
3. Split `/health` endpoint into `/health/live` and `/health/ready`
4. Standardize path parameter: `{id}` → `{content_id}` (for consistency)
5. **Consolidate all routes under `/api/v1/content` prefix** (move user-content routes from `/api/v1/user/{user_id}/content` to `/api/v1/content/user/{user_id}`)
6. Update OpenAPI spec
7. Update mobile app API client

#### Changes Needed
- Health check split: `/health` → `/health/live` + `/health/ready`
- Path parameter consistency: `{id}` → `{content_id}`
- **Route prefix consolidation: `/api/v1/user/{user_id}/content` → `/api/v1/content/user/{user_id}`** (ensures single service prefix)

**Rationale for prefix consolidation**:
- Single service prefix enables clean API gateway routing (all `/api/v1/content/*` routes to Content Service)
- Consistent with API standardization goal: one clear service boundary
- Avoids ambiguity between User Service (`/api/v1/user/*`) and Content Service routes

---

### 5. Read Ingest RSS Service

#### Current Endpoints
```
GET    /health                                         → Liveness + DB
POST   /api/v1/user/{user_id}/feed/subscription        → Subscribe to feed
GET    /api/v1/user/{user_id}/feed/subscription        → List subscriptions
DELETE /api/v1/user/{user_id}/feed/subscription/{feed_id}  → Unsubscribe
PATCH  /api/v1/feed/{feed_id}/enable                   → Re-enable feed
```

#### Target Endpoints (v1)
```
GET    /health/live                                              → Liveness
GET    /health/ready                                             → Readiness (DB)
POST   /api/v1/source/rss/user/{user_id}/subscription           → Subscribe to feed
GET    /api/v1/source/rss/user/{user_id}/subscription           → List subscriptions
DELETE /api/v1/source/rss/user/{user_id}/subscription/{feed_id} → Unsubscribe
PATCH  /api/v1/source/rss/feed/{feed_id}                         → Update feed (including enable/disable)
```

#### Migration Steps
1. ✅ Already uses `/api/v1` prefix (compliant)
2. ✅ Already uses singular resource names (compliant)
3. Split `/health` endpoint into `/health/live` and `/health/ready`
4. Add `/source/rss` prefix (prepares for future source services like webhooks, email, etc.)
5. Refactor `/api/v1/feed/{feed_id}/enable` to use generic `PATCH /api/v1/source/rss/feed/{feed_id}` with body: `{"enabled": true}`
6. Update OpenAPI spec

#### Changes Needed
- Health check split
- Add `/source/rss` prefix to all endpoints (not directly exposed to users - accessed via Content Service gateway)
- Refactor action-based endpoint (`/enable`) to RESTful PATCH
- Path structure: `/user/{user_id}/feed/subscription` to maintain consistency

**Rationale for `/source/rss` prefix**:
- Multiple content source services planned (RSS, webhooks, email integrations)
- Source services are internal-facing, accessed through Content Service gateway
- Clear separation between user-facing content API and source ingestion APIs

---

## Standard Conventions

### 1. Path Parameter Naming
**Standard**: Always use `snake_case`

```
✅ Correct:
/api/v1/user/{user_id}/content/{content_id}
/api/v1/explore/article/{article_id}/vote

❌ Incorrect:
/api/v1/user/{userId}/content/{contentId}
/api/v1/explore/article/{articleID}/vote
```

### 2. Resource Naming
**Standard**: Always use singular nouns for resources

**Rationale**: Singular resource names are more intuitive when reading the path as a sentence and align with RESTful best practices for addressable resources.

```
✅ Correct:
/api/v1/user/{user_id}
/api/v1/article/{article_id}
/api/v1/content/{content_id}
/api/v1/feed/{feed_id}

❌ Incorrect:
/api/v1/users/{user_id}
/api/v1/articles/{article_id}
/api/v1/contents/{content_id}
/api/v1/feeds/{feed_id}
```

### 3. HTTP Methods

| Operation | HTTP Method | Example |
|-----------|-------------|---------|
| Create | `POST` | `POST /api/v1/content` |
| Read (single) | `GET` | `GET /api/v1/content/{content_id}` |
| Read (list) | `GET` | `GET /api/v1/user/{user_id}/content` |
| Update (partial) | `PATCH` | `PATCH /api/v1/user/{user_id}/content/{content_id}` |
| Update (full) | `PUT` | `PUT /api/v1/content/{content_id}` |
| Delete | `DELETE` | `DELETE /api/v1/user/{user_id}/content/{content_id}` |

**Prefer `PATCH` over `PUT`** for most update operations.

### 4. Action-Based Endpoints

**Avoid** action-based endpoints when possible:

```
❌ Avoid:
POST /api/v1/feed/{feed_id}/enable
POST /api/v1/article/{article_id}/archive

✅ Prefer:
PATCH /api/v1/feed/{feed_id}
  Body: {"enabled": true}

PATCH /api/v1/article/{article_id}
  Body: {"status": "archived"}
```

**Exception**: When the action creates a distinct resource or event (e.g., `POST /article/{id}/vote` creates a vote resource, `POST /article/{id}/read` creates a read event).

### 5. Health Check Endpoints

**Standard**: Two endpoints for all services (Kubernetes-compatible)

```
GET /health/live        → Liveness probe
  - Returns 200 if service process is running
  - No dependency checks
  - Fast response (<100ms)
  - Used by orchestrators to restart crashed services

GET /health/ready       → Readiness probe
  - Returns 200 if service is ready to accept traffic
  - Checks database connectivity
  - Checks Vault connectivity (if applicable for User/Explore services)
  - Checks critical dependencies
  - May take longer (up to 5s timeout)
  - Used by load balancers to route traffic
```

**Response Format**:
```json
{
  "status": "healthy|degraded|unhealthy",
  "timestamp": "2026-01-01T12:00:00Z",
  "checks": {
    "database": "ok|error",
    "vault": "ok|error"
  }
}
```

### 6. Authentication

**Standard**: JWT Bearer token in `Authorization` header

```
Authorization: Bearer <access_token>
```

**Error Responses**:
```json
{
  "error": "unauthorized",
  "message": "Missing or invalid authentication token"
}
```

HTTP Status Codes:
- `401 Unauthorized`: Missing or invalid token
- `403 Forbidden`: Valid token but insufficient permissions

---

## Migration Approach

**Strategy**: Single-phase migration (all changes deployed together)

**Rationale**:
- No existing production users to migrate
- Clean slate implementation
- Simpler deployment and testing
- No need for backward compatibility or deprecation period

**Implementation Steps**:
1. Update all service routes to new API structure
2. Implement `/health/live` and `/health/ready` for all services
3. Update all OpenAPI specifications
4. Update mobile app API clients
5. Deploy all services together
6. Verify integration tests pass

**Rollback Plan**:
- Keep pre-migration code in version control
- Monitor service health checks post-deployment
- Quick rollback capability if critical issues arise

---

## API Gateway Considerations

### Future-Proofing for API Gateway

The standardized API structure enables future API gateway deployment:

```
Client Request → API Gateway → Backend Service
                    ↓
            Route by prefix:
            /api/v1/auth/*        → User Service
            /api/v1/user/*        → User Service
            /api/v1/explore/*     → Explore Services
            /api/v1/content/*     → Read Content Service
            /api/v1/source/rss/*  → Read Ingest RSS Service (internal)
```

**Benefits**:
- Centralized authentication/authorization
- Rate limiting per service
- Request/response logging
- API versioning management
- Circuit breaking and retry logic

**Gateway Options**:
- Kong Gateway
- AWS API Gateway
- NGINX with OpenResty
- Traefik

---

## OpenAPI Specification Updates

### Current State
- ✅ All services have OpenAPI 3.0 specs
- ⚠️ Explore spec combines Fetcher and Recommender (confusing)
- ⚠️ Some specs don't match implementation exactly

### Required Updates

#### 1. Split Explore Service Spec
**Current**: Single `services/explore/api/openapi.yaml`

**Target**:
- `services/explore/fetcher/api/openapi.yaml` (Fetcher-specific)
- `services/explore/recommender/api/openapi.yaml` (Recommender-specific)

#### 2. Add Version Information
Update all specs to include:
```yaml
info:
  version: "1.0.0"
  title: "Cairn {Service} API"
  description: |
    Version 1 of the {Service} API.

    **Migration Guide**: See /docs/API_MIGRATION_PLAN.md
```

#### 3. Update Path Structures
```yaml
paths:
  /api/v1/explore/feed/fetch:
    post:
      summary: Manually trigger feed fetch
      description: |
        Triggers an immediate fetch of RSS feeds.
      responses:
        '200':
          description: Fetch triggered successfully
```

#### 4. Standardize Error Responses
Add common error schemas to all specs:
```yaml
components:
  schemas:
    ErrorResponse:
      type: object
      required:
        - error
        - message
      properties:
        error:
          type: string
          example: "invalid_request"
        message:
          type: string
          example: "Missing required field: email"
        details:
          type: object
          additionalProperties: true
```

---

## Testing Strategy

### 1. Integration Tests
**Goal**: Ensure all services work together with new API structure

```bash
# Test new endpoints
curl http://localhost:8081/api/v1/explore/recommendation/user123

# Test health checks
curl http://localhost:8082/health/live
curl http://localhost:8082/health/ready
```

**Automated Tests**:
- Integration tests for all new routes
- Contract tests to ensure response format matches OpenAPI specs
- Load tests to verify performance meets requirements

### 2. OpenAPI Validation
**Goal**: Ensure implementation matches specification

```bash
# Validate OpenAPI specs
npx @apidevtools/swagger-cli validate services/*/api/openapi.yaml

# Generate tests from OpenAPI specs
npm install -g dredd
dredd services/explore/fetcher/api/openapi.yaml http://localhost:8080
```

### 3. End-to-End Testing
**Goal**: Verify complete user workflows work with new API

**Test Scenarios**:
1. User registration and authentication flow
2. Content discovery and saving flow
3. RSS feed subscription and content delivery flow
4. Search and content retrieval flow

---

## Documentation Updates

### Files to Update

1. **CLAUDE.md**
   - Update "API Endpoints" section with new paths
   - Update all example curl commands
   - Reference this migration plan

2. **README.md**
   - Update quick start examples with new endpoints

3. **Service-Specific READMEs**
   - `services/explore/README.md`
   - `services/users/README.md`
   - `services/read/README.md`
   - Update all example `curl` commands

4. **OpenAPI Specs**
   - `services/explore/fetcher/api/openapi.yaml`
   - `services/explore/recommender/api/openapi.yaml`
   - `services/users/api/openapi.yaml`
   - `services/read/content/api/openapi.yaml`
   - `services/read/fetcher/api/openapi.yaml`

5. **Mobile App**
   - Update API client configurations
   - Update service URLs in `apps/mobile/src/config/api.ts`
   - Update all API service methods to use new endpoints

---

## Success Metrics

### Key Performance Indicators (KPIs)

1. **API Consistency**
   - Target: 100% of endpoints follow standardized conventions
   - Measure: OpenAPI spec validation passing

2. **Service Health**
   - Target: All services pass health checks post-deployment
   - Measure: `/health/live` and `/health/ready` endpoints return 200

3. **Integration Tests**
   - Target: 100% of integration tests pass
   - Measure: Automated test suite execution

4. **Documentation Accuracy**
   - Target: 100% of OpenAPI specs match implementation
   - Measure: Automated contract testing passing

---

## Rollback Plan

### If Migration Causes Issues

**Scenario 1: Service Health Check Failures**
- Investigate health check failures
- Rollback to previous version if critical
- Fix issues and redeploy

**Scenario 2: Performance Degradation**
- Profile new endpoint implementation
- Optimize or rollback if severe
- Re-test performance before redeploying

**Scenario 3: Integration Failures**
- Debug service-to-service communication
- Verify all services deployed correctly
- Rollback all services together if needed

### Rollback Procedure
1. Stop all services
2. Revert to previous version in version control
3. Rebuild and redeploy all services
4. Verify health checks pass
5. Run integration tests
6. Investigate root cause and fix
7. Re-test in staging environment before redeploying

---

## Open Questions

1. **API Gateway Timeline**: When should we introduce an API gateway?
   - Recommendation: After migration completes (once standardization is complete)

2. **Framework Consolidation**: Should we migrate all services to Chi router?
   - Recommendation: Not urgent, but consider for future refactoring

3. **gRPC vs REST**: Should internal service-to-service communication use gRPC?
   - Recommendation: Evaluate after migration completes

4. **Rate Limiting**: Where should rate limiting be implemented?
   - Recommendation: At API gateway level (future work)

---

## Appendix A: Complete Endpoint Mapping

### Explore Fetcher

| Old Endpoint | New Endpoint | Method | Auth |
|--------------|--------------|--------|------|
| `/health` | `/health/live` | GET | No |
| N/A | `/health/ready` | GET | No |
| `/fetch` | `/api/v1/explore/feed/fetch` | POST | No |

### Explore Recommender

| Old Endpoint | New Endpoint | Method | Auth |
|--------------|--------------|--------|------|
| `/health` | `/health/live` | GET | No |
| `/ready` | `/health/ready` | GET | No |
| `/explore/article` | `/api/v1/explore/article` | POST | No |
| `/explore/recommendation/{userID}` | `/api/v1/explore/recommendation/{user_id}` | GET | Yes |
| `/explore/article/read` | `/api/v1/explore/article/{article_id}/read` | POST | Yes |
| `/explore/article/{articleID}/vote` | `/api/v1/explore/article/{article_id}/vote` | POST | Yes |
| `/explore/article/{articleID}/vote` | `/api/v1/explore/article/{article_id}/vote` | DELETE | Yes |
| `/explore/article/{articleID}/vote` | `/api/v1/explore/article/{article_id}/vote` | GET | Yes |

### User Service

| Old Endpoint | New Endpoint | Method | Auth |
|--------------|--------------|--------|------|
| `/health` | `/health/live` | GET | No |
| `/ready` | `/health/ready` | GET | No |
| `/auth/register` | `/api/v1/auth/register` | POST | No |
| `/auth/register/mobile` | `/api/v1/auth/register/mobile` | POST | No |
| `/auth/login` | `/api/v1/auth/login` | POST | No |
| `/auth/login/mobile` | `/api/v1/auth/login/mobile` | POST | No |
| `/auth/refresh` | `/api/v1/auth/refresh` | POST | No |
| `/auth/logout` | `/api/v1/auth/logout` | POST | No |
| `/auth/logout-all` | `/api/v1/auth/logout-all` | POST | Yes |
| `/user/{id}` | `/api/v1/user/{user_id}` | GET | Yes |
| `/user/{id}` | `/api/v1/user/{user_id}` | PATCH | Yes |
| `/user/{id}/upgrade` | `/api/v1/user/{user_id}/upgrade` | POST | Yes |
| `/user/{id}` | `/api/v1/user/{user_id}` | DELETE | Yes |

### Read Content Service

| Old Endpoint | New Endpoint | Method | Auth |
|--------------|--------------|--------|------|
| `/health` | `/health/live`, `/health/ready` | GET | No |
| `/api/v1/content` | `/api/v1/content` | POST | No |
| `/api/v1/content/{id}` | `/api/v1/content/{content_id}` | GET | No |
| `/api/v1/content/{id}` | `/api/v1/content/{content_id}` | PUT | No |
| `/api/v1/content/bulk` | `/api/v1/content/bulk` | POST | No |
| `/api/v1/content/check-duplicate` | `/api/v1/content/check-duplicate` | POST | No |
| `/api/v1/user/{user_id}/content` | `/api/v1/content/user/{user_id}` | GET | Yes |
| `/api/v1/user/{user_id}/content` | `/api/v1/content/user/{user_id}` | POST | Yes |
| `/api/v1/user/{user_id}/content/search` | `/api/v1/content/user/{user_id}/search` | GET | Yes |
| `/api/v1/user/{user_id}/content/{content_id}` | `/api/v1/content/user/{user_id}/{content_id}` | PATCH | Yes |
| `/api/v1/user/{user_id}/content/{content_id}` | `/api/v1/content/user/{user_id}/{content_id}` | DELETE | Yes |
| `/api/v1/user/bulk/content` | `/api/v1/content/user/bulk` | POST | No |

### Read Ingest RSS Service

| Old Endpoint | New Endpoint | Method | Auth |
|--------------|--------------|--------|------|
| `/health` | `/health/live`, `/health/ready` | GET | No |
| `/api/v1/user/{user_id}/feed/subscription` | `/api/v1/source/rss/user/{user_id}/subscription` | POST | Yes |
| `/api/v1/user/{user_id}/feed/subscription` | `/api/v1/source/rss/user/{user_id}/subscription` | GET | Yes |
| `/api/v1/user/{user_id}/feed/subscription/{feed_id}` | `/api/v1/source/rss/user/{user_id}/subscription/{feed_id}` | DELETE | Yes |
| `/api/v1/feed/{feed_id}/enable` | `/api/v1/source/rss/feed/{feed_id}` (with body) | PATCH | Yes |

---

## Appendix B: Response Format Standards

### Success Response
```json
{
  "data": {
    // Resource data
  },
  "meta": {
    "timestamp": "2026-01-01T12:00:00Z",
    "version": "v1"
  }
}
```

### Error Response
```json
{
  "error": "resource_not_found",
  "message": "Content with ID abc123 not found",
  "details": {
    "content_id": "abc123"
  },
  "meta": {
    "timestamp": "2026-01-01T12:00:00Z",
    "version": "v1"
  }
}
```

### Paginated Response
```json
{
  "data": [
    // Array of resources
  ],
  "pagination": {
    "cursor": "eyJpZCI6MTIzfQ==",
    "has_more": true,
    "limit": 20
  },
  "meta": {
    "timestamp": "2026-01-01T12:00:00Z",
    "version": "v1"
  }
}
```

---

## Revision History

| Date | Version | Author | Changes |
|------|---------|--------|---------|
| 2026-01-01 | 1.0 | Engineering | Initial migration plan |

# Read Content Service API Route Standardization Plan

**Status**: Implementation Ready
**Created**: 2026-01-01
**Owner**: Engineering

---

## Problem Statement

The Read Content Service currently uses a mixed routing structure that splits endpoints across `/api/v1/content` and `/api/v1/user/{user_id}/content` prefixes. This violates the API standardization goal of having one clear service prefix per service.

**Current Issues:**
1. Routes split across two top-level prefixes (`/content` and `/user`)
2. Inconsistent with other services that use single prefix
3. Makes API gateway routing configuration more complex
4. Confusing for API consumers to determine the correct prefix

---

## Proposed Solution

**Consolidate all Read Content Service routes under `/api/v1/content` prefix**, using nested user paths where appropriate.

### Route Mapping

| Old Route | New Route | Change Type |
|-----------|-----------|-------------|
| `GET /health` | `GET /health/live` | Split health check |
| N/A | `GET /health/ready` | New readiness check |
| `POST /api/v1/content` | `POST /api/v1/content` | ✅ No change |
| `GET /api/v1/content/{id}` | `GET /api/v1/content/{content_id}` | Parameter rename |
| `PUT /api/v1/content/{id}` | `PUT /api/v1/content/{content_id}` | Parameter rename |
| `POST /api/v1/content/bulk` | `POST /api/v1/content/bulk` | ✅ No change |
| `POST /api/v1/content/check-duplicate` | `POST /api/v1/content/check-duplicate` | ✅ No change |
| `GET /api/v1/user/{user_id}/content` | `GET /api/v1/content/user/{user_id}` | **Prefix change** |
| `POST /api/v1/user/{user_id}/content` | `POST /api/v1/content/user/{user_id}` | **Prefix change** |
| `GET /api/v1/user/{user_id}/content/search` | `GET /api/v1/content/user/{user_id}/search` | **Prefix change** |
| `PATCH /api/v1/user/{user_id}/content/{content_id}` | `PATCH /api/v1/content/user/{user_id}/{content_id}` | **Prefix change** |
| `DELETE /api/v1/user/{user_id}/content/{content_id}` | `DELETE /api/v1/content/user/{user_id}/{content_id}` | **Prefix change** |
| `POST /api/v1/user/bulk/content` | `POST /api/v1/content/user/bulk` | **Prefix change** |

---

## Implementation Checklist

### 1. Backend Changes

#### 1.1 Router Updates (`services/read/content/internal/api/router.go`)

**Changes Required:**
- [x] Split `/health` into `/health/live` (liveness) and `/health/ready` (readiness with DB check)
- [x] Change parameter names: `{id}` → `{content_id}` in all routes
- [x] Move user-content routes from `/api/v1/user/{user_id}/content` to `/api/v1/content/user/{user_id}`
- [x] Update route registration calls to use new paths

**Example:**
```go
// OLD
r.Route("/api/v1", func(r chi.Router) {
    r.Post("/content", contentHandler.CreateContent)
    r.Get("/content/{id}", contentHandler.GetContent)

    r.Route("/user/{user_id}/content", func(r chi.Router) {
        r.Get("/", userContentHandler.ListUserContents)
        r.Post("/", userContentHandler.AddContentToUser)
    })
})

// NEW
r.Route("/api/v1/content", func(r chi.Router) {
    r.Post("/", contentHandler.CreateContent)
    r.Get("/{content_id}", contentHandler.GetContent)

    r.Route("/user/{user_id}", func(r chi.Router) {
        r.Get("/", userContentHandler.ListUserContents)
        r.Post("/", userContentHandler.AddContentToUser)
    })
})
```

#### 1.2 Handler Updates

**Files to Update:**
- `services/read/content/internal/api/handlers/content_handler.go`
- `services/read/content/internal/api/handlers/user_content_handler.go`
- `services/read/content/internal/api/handlers/bulk_handler.go`

**Changes Required:**
- [x] Update URL parameter extraction: `chi.URLParam(r, "id")` → `chi.URLParam(r, "content_id")`
- [x] No business logic changes needed (only parameter name changes)

#### 1.3 Health Check Implementation

**File:** `services/read/content/internal/api/router.go`

**Add two endpoints:**
```go
// Liveness check - simple process check
r.Get("/health/live", func(w http.ResponseWriter, r *http.Request) {
    WriteJSON(w, http.StatusOK, map[string]string{
        "status": "healthy",
        "timestamp": time.Now().Format(time.RFC3339),
    })
})

// Readiness check - includes DB connectivity
r.Get("/health/ready", func(w http.ResponseWriter, r *http.Request) {
    // Check DB connectivity
    if err := db.PingContext(r.Context()); err != nil {
        WriteError(w, http.StatusServiceUnavailable, "database_unavailable",
            "Database connection failed", nil)
        return
    }

    WriteJSON(w, http.StatusOK, map[string]interface{}{
        "status": "healthy",
        "timestamp": time.Now().Format(time.RFC3339),
        "checks": map[string]string{
            "database": "ok",
        },
    })
})
```

#### 1.4 OpenAPI Specification Update

**File:** `services/read/content/api/openapi.yaml`

**Changes Required:**
- [x] Update all path definitions to use new route structure
- [x] Change parameter names: `id` → `content_id`
- [x] Add `/health/live` and `/health/ready` endpoints
- [x] Update examples and descriptions
- [x] Fix plural/singular inconsistency (use singular: `/content`, not `/contents`)

---

### 2. Mobile App Changes

#### 2.1 API Client Update (`apps/mobile/src/services/read.ts`)

**Changes Required:**
- [x] Update all endpoint strings to use new route structure
- [x] No function signature changes (internal implementation only)

**Example:**
```typescript
// OLD
const endpoint = `/api/v1/user/${userId}/content`;

// NEW
const endpoint = `/api/v1/content/user/${userId}`;
```

**Affected Methods:**
- `listUserContents()` - Update endpoint construction
- `searchUserContents()` - Update endpoint construction
- `addContentToUser()` - Update endpoint construction
- `updateUserContent()` - Update endpoint construction
- `deleteUserContent()` - Update endpoint construction

#### 2.2 Type Definitions

**File:** `apps/mobile/src/types/read.ts`

**Changes:** ✅ No changes needed (types are endpoint-agnostic)

#### 2.3 Screen Updates

**Files:**
- `apps/mobile/src/screens/ReadScreen.tsx`
- `apps/mobile/src/screens/FavoritesScreen.tsx` (future)
- `apps/mobile/src/screens/ArchiveScreen.tsx` (future)

**Changes:** ✅ No changes needed (screens call service methods, not endpoints directly)

---

### 3. Documentation Updates

#### 3.1 API Migration Plan

**File:** `docs/API_MIGRATION_PLAN.md`

**Changes:**
- [x] Update "4. Read Content Service" section with corrected target endpoints
- [x] Update Appendix A endpoint mapping table
- [x] Update response format examples if needed

#### 3.2 CLAUDE.md

**File:** `CLAUDE.md`

**Changes:**
- [x] Update "Read Service - Content Service" endpoint table
- [x] Update example curl commands
- [x] Update health check documentation

#### 3.3 Read Service README

**File:** `services/read/README.md`

**Changes:**
- [x] Update API endpoint documentation
- [x] Update example curl commands
- [x] Update quick start guide

---

## Testing Strategy

### 1. Backend Testing

**Manual Tests:**
```bash
# Start service
cd services/read
docker-compose up --build

# Test health checks
curl http://localhost:8083/health/live
curl http://localhost:8083/health/ready

# Test content endpoints
curl -X POST http://localhost:8083/api/v1/content \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com", "title": "Test"}'

curl http://localhost:8083/api/v1/content/{content_id}

# Test user-content endpoints (with auth)
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8083/api/v1/content/user/{user_id}

curl -X POST -H "Authorization: Bearer $TOKEN" \
  http://localhost:8083/api/v1/content/user/{user_id} \
  -d '{"url": "https://example.com"}'

curl -X PATCH -H "Authorization: Bearer $TOKEN" \
  http://localhost:8083/api/v1/content/user/{user_id}/{content_id} \
  -d '{"status": "reading"}'
```

**Automated Tests:**
- Run existing integration tests (should pass with route updates)
- Add new health check tests

### 2. Mobile App Testing

**Test Scenarios:**
1. Add article via ReadScreen modal
2. Load articles in ReadScreen (pagination)
3. Pull to refresh in ReadScreen
4. Mark article as favorite (when integrated)
5. Search articles (when UI implemented)
6. Update article status (when integrated)

**Testing Commands:**
```bash
cd apps/mobile
npm start
# Test on iOS simulator: press 'i'
# Test on Android emulator: press 'a'
```

### 3. Integration Testing

**Full Flow Test:**
1. Start all services via centralized docker-compose
2. Register mobile user via User Service
3. Add content via Read Service
4. Fetch content list via mobile app
5. Update content status via mobile app
6. Verify changes persist across app restarts

---

## Rollback Plan

**If issues are detected:**

1. **Backend:** Revert commits to `services/read/content/internal/api/router.go`
2. **Mobile:** Revert commits to `apps/mobile/src/services/read.ts`
3. **Rebuild and redeploy** affected services
4. **Investigate root cause** before re-attempting

**Version Control:**
- All changes will be in a feature branch
- Can easily revert individual commits
- No database schema changes (safe to rollback)

---

## Migration Impact Analysis

### Breaking Changes
- ✅ **YES** - All user-content endpoints change paths
- ✅ **YES** - Health check endpoint changes

### Non-Breaking Changes
- ✅ Content creation/retrieval endpoints (same paths, just parameter rename)
- ✅ Bulk operation endpoints (same paths)

### Deployment Requirements
- **Backend and mobile app must be deployed together**
- **No database migrations needed**
- **No data migration needed**
- **Services can be updated independently** (mobile app is only consumer)

### Affected Systems
1. **Read Content Service** - Route definitions
2. **Mobile App** - API client endpoints
3. **Documentation** - API references
4. **Future API Gateway** - Routing configuration (not yet deployed)

---

## Success Criteria

- [x] All backend routes follow `/api/v1/content/*` structure
- [x] Health checks split into `/health/live` and `/health/ready`
- [x] Mobile app successfully calls all endpoints with new paths
- [x] Integration tests pass
- [x] OpenAPI specification validates and matches implementation
- [x] Documentation updated and accurate

---

## Timeline

**Estimated Time:** 2-3 hours

1. Backend changes: 45 minutes
2. Mobile app changes: 30 minutes
3. Testing: 30 minutes
4. Documentation updates: 30 minutes
5. Integration testing: 15 minutes

---

## Open Questions

1. ✅ **Resolved:** Should we maintain backward compatibility?
   - **Answer:** No, single-phase migration (no existing production users per API_MIGRATION_PLAN.md)

2. ✅ **Resolved:** Should bulk endpoint move to `/api/v1/content/bulk/user`?
   - **Answer:** No, keep as `/api/v1/content/user/bulk` for consistency with user-scoped pattern

3. ✅ **Resolved:** Do we need to update Ingest RSS service calls?
   - **Answer:** Yes, Ingest RSS service calls Content Service bulk endpoint - will need update

---

## Related Documents

- [API Migration Plan](docs/API_MIGRATION_PLAN.md)
- [CLAUDE.md](CLAUDE.md)
- [Read Service README](services/read/README.md)
- [Engineering Principles](docs/ENGINEERING_PRINCIPLES.md)

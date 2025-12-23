# API Endpoint Migration Summary

## Overview
This document summarizes the migration of all Explore service API endpoints from `/api/v1/` to `/explore/` prefix pattern.

## Date
December 23, 2025

## Changes

### Recommender Service Endpoints

All endpoints now use the `/explore/` prefix instead of `/api/v1/`:

#### Public Endpoints (No Authentication)
- **Health Check**: `/health` (unchanged)
- **Submit Articles**: `/api/v1/articles` → `/explore/articles`

#### Protected Endpoints (Require JWT Authentication)
- **Get Recommendations**: `/api/v1/recommendations/{userID}` → `/explore/recommendations/{userID}`
- **Mark Article as Read**: `/api/v1/articles/read` → `/explore/articles/read`
- **Vote on Article**: `/api/v1/articles/{articleID}/vote` → `/explore/articles/{articleID}/vote`
- **Remove Vote**: `/api/v1/articles/{articleID}/vote` → `/explore/articles/{articleID}/vote` (DELETE)
- **Get Vote Counts**: `/api/v1/articles/{articleID}/votes` → `/explore/articles/{articleID}/votes`

## Files Modified

### Code Changes
1. **[recommender/internal/api/server.go](recommender/internal/api/server.go)** - Updated route registrations
2. **[recommender/internal/api/handlers.go](recommender/internal/api/handlers.go)** - Updated path parsing in handlers
3. **[fetcher/internal/client/recommender_client.go](fetcher/internal/client/recommender_client.go)** - Updated article submission endpoint

### Test Updates
4. **[recommender/integration_test.go](recommender/integration_test.go)** - Updated all test HTTP calls
5. **[fetcher/integration_test.go](fetcher/integration_test.go)** - Updated mock server endpoint

### Documentation Updates
6. **[README.md](README.md)** - Updated API endpoint documentation
7. **[CLAUDE.md](CLAUDE.md)** - Updated endpoint references and examples
8. **[/Users/andrew/projects/cairn/CLAUDE.md](/Users/andrew/projects/cairn/CLAUDE.md)** - Should be updated with new endpoints

## Migration Impact

### Breaking Changes
⚠️ **All clients calling the recommender service must update their endpoints:**

**Before:**
```bash
curl http://localhost:8081/api/v1/recommendations/user123
curl -X POST http://localhost:8081/api/v1/articles/read -d '{"user_id":"user123","article_id":"abc"}'
```

**After:**
```bash
curl -H "Authorization: Bearer <JWT>" http://localhost:8081/explore/recommendations/user123
curl -X POST -H "Authorization: Bearer <JWT>" http://localhost:8081/explore/articles/read -d '{"article_id":"abc"}'
```

### Authentication Changes
- Protected endpoints now require JWT authentication via `Authorization: Bearer <token>` header
- User ID is extracted from JWT token context, not request body/path
- This improves security and follows REST best practices

## Testing

To verify the changes work correctly:

```bash
# Start services
cd services/explore
docker-compose up --build

# Run integration tests
go test -v ./recommender/... ./fetcher/...

# Manual testing (requires JWT token from user service)
TOKEN="<your-jwt-token>"
curl -H "Authorization: Bearer $TOKEN" http://localhost:8081/explore/recommendations/user123
curl -X POST -H "Authorization: Bearer $TOKEN" http://localhost:8081/explore/articles/read \
  -d '{"article_id":"test-article-id"}'
```

## Next Steps

1. ✅ Update all endpoint routes in code
2. ✅ Update all test files
3. ✅ Update documentation
4. ⏳ Update mobile app to use new endpoints (pending)
5. ⏳ Update main CLAUDE.md with new endpoint patterns (pending)

## Rollback Plan

If needed, the changes can be reverted by:
1. Replacing `/explore/` with `/api/v1/` in the modified files
2. Reverting the path parsing logic in handlers
3. Running tests to verify rollback

All changes are tracked in git commit history for easy rollback.

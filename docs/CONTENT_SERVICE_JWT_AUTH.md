# Content Service JWT Authentication Implementation

## Overview

The Content Service currently lacks JWT authentication, allowing any client to access any user's content by providing a user ID in the URL. This document describes the implementation plan to add proper authentication following the established patterns from the Explore Recommender Service.

## Problem Statement

**Current State**:
- Content Service endpoints are completely unprotected
- Any client can list content for ANY user by providing their user ID
- Any client can add, update, or delete content for ANY user
- Handlers accept `{user_id}` from the URL without verifying the requesting client is that user

**Why It Happened**:
- Shared JWT infrastructure exists in `pkg/auth/` but was never integrated
- Content Service has no Vault configuration
- No authentication middleware in router
- No authorization checks in handlers

**Expected State**:
- Frontend sends JWT token in `Authorization: Bearer <token>` header
- Content Service validates token using RequireAuth middleware
- Token contains user_id claim, compared against `{user_id}` in request
- User can only access their own content

## Implementation Phases

### Phase 1: Configuration & Vault Setup

**1.1 Update config.go**
- Add `VaultConfig` struct with:
  - `Address` - Vault server address (e.g., "http://localhost:8200")
  - `Token` - Development token-based auth
  - `RoleID` - Production AppRole role ID
  - `SecretID` - Production AppRole secret ID
  - `AuthPath` - AppRole auth mount path (e.g., "approle")
  - `PublicKeyPath` - Path to JWT public key in Vault (e.g., "secret/data/jwt/public-key")

- Load from environment variables:
  - `VAULT_ADDR`
  - `VAULT_TOKEN` (development)
  - `VAULT_ROLE_ID` (production)
  - `VAULT_SECRET_ID` (production)
  - `VAULT_AUTH_PATH`
  - `JWT_PUBLIC_KEY_PATH`

- Add validation:
  - Vault address must be set
  - Auth method must be either token OR AppRole (not both, not neither)
  - Public key path must be set

**1.2 Update main.go**
- After database initialization:
  1. Create Vault client with configured credentials
  2. Verify Vault connectivity via `client.Health()`
  3. Fetch JWT public key from Vault using `client.GetPublicKey()`
  4. Create `auth.Validator` with the public key
  5. Create `auth.Middleware` with the validator
  6. Pass middleware to `api.NewRouter()`

- Error handling:
  - Exit with error if Vault is unreachable
  - Exit with error if public key cannot be retrieved
  - Log successful initialization at startup

### Phase 2: Router & Middleware Integration

**2.1 Update api/router.go**
- Modify `NewRouter()` signature:
  ```go
  func NewRouter(db *database.DB, ingestRSSServiceURL string, authMiddleware *auth.Middleware) http.Handler
  ```

- Apply `RequireAuth` middleware to protected user-specific routes:
  ```
  ✓ GET /api/v1/content/user/{user_id}                    - List user contents
  ✓ POST /api/v1/content/user/{user_id}                   - Add content to user
  ✓ GET /api/v1/content/user/{user_id}/search             - Search user contents
  ✓ PATCH /api/v1/content/user/{user_id}/{content_id}     - Update user content
  ✓ DELETE /api/v1/content/user/{user_id}/{content_id}    - Delete from user
  ✓ POST /api/v1/content/user/bulk                        - Bulk add to users
  ```

- Keep unprotected routes:
  ```
  ✗ GET /health/live                                       - Liveness probe
  ✗ GET /health/ready                                      - Readiness probe
  ✗ GET /                                                  - Root endpoint
  ✗ POST /api/v1/content/detect                            - URL detection
  ✗ POST /api/v1/content/                                  - Create content (internal)
  ✗ GET /api/v1/content/{content_id}                       - Get content (internal)
  ✗ PUT /api/v1/content/{content_id}                       - Update content (internal)
  ```

**2.2 Internal Service Routes**
- Create separate route group for internal operations:
  ```go
  r.Route("/api/v1/internal", func(r chi.Router) {
    // No auth required - these are internal-only
    r.Post("/content/user/bulk", bulkHandler.BulkAddToUsersInternal)
    // other internal endpoints as needed
  })
  ```

- Ingest RSS Service uses these internal endpoints instead of user-authenticated ones
- Document these endpoints as internal-only in OpenAPI spec

## Phase 3: Handler Authorization

**3.1 User Content Handler**

For all user-specific routes, implement the authorization check:

```go
// Extract authenticated user ID from context
authenticatedUserID := auth.MustGetUserID(r.Context())

// Extract requested user ID from URL
requestedUserID, err := uuid.Parse(chi.URLParam(r, "user_id"))
if err != nil {
    // Return 400 Bad Request
}

// Verify user owns the content
if authenticatedUserID != requestedUserID {
    // Return 403 Forbidden with error message
}

// Proceed with operation
```

**Affected Methods**:
- `ListUserContents()` - GET /api/v1/content/user/{user_id}
- `AddContentToUser()` - POST /api/v1/content/user/{user_id}
- `SearchUserContents()` - GET /api/v1/content/user/{user_id}/search
- `UpdateUserContent()` - PATCH /api/v1/content/user/{user_id}/{content_id}
- `DeleteUserContent()` - DELETE /api/v1/content/user/{user_id}/{content_id}

**3.2 Content Handler - No Changes**
- `CreateContent()` and `GetContent()` don't take user_id parameter
- These remain unprotected (used by internal services like Ingest RSS)
- Can be called without authentication

**3.3 Bulk Handler - Authorization Logic**
- `BulkCreateContent()` - Creates content records, no user_id involved
  - Keep unprotected (internal use)

- `CheckDuplicates()` - Checks for duplicate content
  - Keep unprotected (internal use)

- `BulkAddToUsers()` - Adds existing content to user
  - **PROTECTED** - Requires authentication
  - Extract authenticated user ID from context
  - Compare with each user_id in request body
  - Return 403 if trying to add content to other users
  - Create separate internal endpoint for internal service use

## Phase 4: Error Response Format

All authentication/authorization failures use standard error format:

**401 Unauthorized** - Missing or invalid token:
```json
{
  "error": "unauthorized",
  "message": "Missing or invalid authentication token"
}
```

**403 Forbidden** - User lacks permission:
```json
{
  "error": "forbidden",
  "message": "User can only access their own content"
}
```

**400 Bad Request** - Invalid user ID format:
```json
{
  "error": "bad_request",
  "message": "Invalid user ID format"
}
```

## Phase 5: Testing

**5.1 Add Authentication Tests**
- Test missing JWT token returns 401
- Test malformed JWT returns 401
- Test expired JWT returns 401
- Test invalid signature returns 401
- Test user cannot access other user's content (403)
- Test authenticated user can access own content (200)

**5.2 Update Handler Tests**
- Mock `auth.MustGetUserID()` in handler tests
- Inject authentication context into test requests
- Add test fixtures for valid JWT tokens
- Update existing tests to include auth context where needed
- Test authorization failure cases

**5.3 Integration Tests**
- Test full request flow with valid JWT token
- Test full request flow with missing token
- Verify Vault integration works correctly
- Test token validation against actual public key

## Phase 6: OpenAPI Specification Updates

**6.1 Update api/openapi.yaml**
- Add `SecurityScheme` definition for Bearer JWT:
  ```yaml
  components:
    securitySchemes:
      BearerAuth:
        type: http
        scheme: bearer
        bearerFormat: JWT
        description: JWT token from User Service
  ```

- Add `security` requirement to protected endpoints:
  ```yaml
  /api/v1/content/user/{user_id}:
    get:
      security:
        - BearerAuth: []
      responses:
        401:
          description: Missing or invalid authentication token
        403:
          description: User can only access their own content
  ```

- Document all protected endpoints with 401/403 responses
- Mark internal endpoints with `x-internal: true` extension

## Implementation Details

### Environment Variables

**Development**:
```bash
VAULT_ADDR=http://localhost:8200
VAULT_TOKEN=dev-root-token
JWT_PUBLIC_KEY_PATH=secret/data/jwt/public-key
VAULT_AUTH_PATH=approle
```

**Production**:
```bash
VAULT_ADDR=https://vault.example.com:8200
VAULT_ROLE_ID=<role-id>
VAULT_SECRET_ID=<secret-id>
JWT_PUBLIC_KEY_PATH=secret/data/jwt/public-key
VAULT_AUTH_PATH=approle
```

### Middleware Chain

```go
r := chi.NewRouter()

// Global middleware (no auth required)
r.Use(middleware.Recovery)
r.Use(logging.ChiRequestLogger(slog.Default()))
r.Use(middleware.ValidateJSON)

// Routes are protected individually using RequireAuth
r.Route("/api/v1/content", func(r chi.Router) {
  // Public routes
  r.Post("/detect", detectionHandler.DetectURL)
  r.Post("/", contentHandler.CreateContent)

  // Protected routes
  r.Route("/user/{user_id}", func(r chi.Router) {
    r.Use(authMiddleware.RequireAuth(http.MethodGet))
    r.Get("/", userContentHandler.ListUserContents)
    // ... other protected routes
  })
})
```

### Token Validation Flow

1. Client includes `Authorization: Bearer <JWT>` in request header
2. Middleware calls `auth.Validator.ValidateToken(tokenString)`
3. Validator:
   - Parses JWT with RS256 public key
   - Verifies signature is valid
   - Checks token is not expired
   - Validates issuer claim ("cairn-user-service")
   - Validates audience claim ("cairn-api")
4. Middleware extracts user_id from token claims
5. Stores user_id in request context via `auth.SetUserIDInContext()`
6. Handler retrieves user_id via `auth.MustGetUserID(r.Context())`

### Integration with Ingest RSS Service

The Ingest RSS Service currently makes requests to Content Service to add content to users.

**Before Implementation**:
- Calls `POST /api/v1/content/user/{user_id}`
- No authentication required

**After Implementation**:
- Creates internal endpoint: `POST /api/v1/internal/content/user/bulk`
- Ingest RSS Service calls this endpoint instead
- No JWT required for internal endpoints
- No changes needed to Ingest RSS Service behavior, only endpoint URL

**Internal Endpoint Response** - Same format as user endpoint, just at different path

## Files to Modify

| File | Changes |
|------|---------|
| `services/read/content/internal/config/config.go` | Add VaultConfig struct, environment variable loading, validation |
| `services/read/content/cmd/content/main.go` | Initialize Vault client, fetch public key, create middleware, pass to router |
| `services/read/content/internal/api/router.go` | Update NewRouter signature, apply RequireAuth middleware, create internal routes |
| `services/read/content/internal/api/handlers/user_content_handler.go` | Add user ID extraction and comparison logic to all methods |
| `services/read/content/internal/api/handlers/bulk_handler.go` | Add authorization check to BulkAddToUsers, create BulkAddToUsersInternal |
| `services/read/content/internal/api/handlers/*_test.go` | Add auth context to tests, test authorization failures |
| `services/read/content/api/openapi.yaml` | Add SecurityScheme, mark protected endpoints, document error responses |
| `.env.example` (if exists) | Add Vault configuration variables |

## Security Considerations

1. **Token Validation**: Uses `pkg/auth.Validator` which validates:
   - RS256 signature correctness
   - Token expiration
   - Issuer claim matches "cairn-user-service"
   - Audience claim matches "cairn-api"

2. **User ID Comparison**: Always compare authenticated user_id from JWT with URL parameter before allowing access

3. **Internal Routes**: Clearly marked with `/api/v1/internal/` prefix to distinguish from user-facing API

4. **Token Logging**: Never log token values; use opaque error messages to prevent token leakage

5. **CORS Security**: Ensure CORS headers don't expose or grant access to internal routes

6. **Key Rotation**: Public key is fetched from Vault at startup; restart service to pick up new keys

## Success Criteria

✅ User can only access their own content (403 Forbidden on unauthorized attempts)
✅ Missing JWT token returns 401 Unauthorized
✅ Invalid/expired JWT returns 401 Unauthorized
✅ Authenticated users can fully manage their content
✅ Internal services can still add content via internal endpoints
✅ Health checks remain unauthenticated
✅ All existing tests pass with auth context added
✅ New authorization tests cover failure scenarios
✅ OpenAPI spec documents all authentication requirements

## Related Documentation

- [JWT Authentication in pkg/auth](ARCHITECTURE.md#jwt-authentication)
- [Explore Recommender Service Auth Implementation](services/explore/CLAUDE.md)
- [Security Principles](ENGINEERING_PRINCIPLES.md#security)
- [Vault Configuration](CONFIGURATION.md)

## Rollout Notes

This is a complete replacement of the current authentication model (none) with JWT-based authentication. All services calling Content Service endpoints must:

1. **Frontend**: Send JWT token in `Authorization: Bearer <token>` header for user-specific endpoints
2. **Internal Services** (e.g., Ingest RSS): Update endpoint URLs from `/api/v1/content/user/{user_id}` to `/api/v1/internal/content/user/{user_id}` or use new bulk internal endpoint
3. **Monitoring**: Update health checks to continue using `/health/live` and `/health/ready` (unauthenticated)

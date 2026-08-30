# CLAUDE.md - User Service

This file provides guidance to Claude Code (claude.ai/code) when working with the User service in this directory.

> 📖 **For project-wide context and conventions, see [/CLAUDE.md](/CLAUDE.md) and [/docs/ENGINEERING_PRINCIPLES.md](/docs/ENGINEERING_PRINCIPLES.md)**

## Service Overview

The User Service is responsible for user authentication, authorization, and account management for the Cairn read-it-later application. It implements stateless JWT-based authentication with refresh token rotation and supports both email/password and mobile device authentication.

**Location**: `/services/users/`

**Port**: 8082

**Database**: `cairn_users` (PostgreSQL)

**Key Responsibilities**:
- User registration and account management
- Stateless JWT authentication with RS256 signing
- Refresh token management with automatic rotation
- Mobile device authentication via Expo device ID
- Account upgrade from device-only to email/password
- Authorization middleware ensuring users can only access their own data
- Secure secrets management with HashiCorp Vault

## Quick Start

### Running with Centralized Docker Compose (Recommended)

The easiest way to run all Cairn backend services (including User Service) is using the centralized setup:

```bash
# From repository root
cd infrastructure/docker/dev

# Copy and configure environment variables
cp .env.example .env
# Edit .env and set secure passwords

# Start all services (includes Vault, databases, all microservices)
docker compose up --build -d

# Check User Service health
curl http://localhost:8082/health/ready

# View User Service logs
docker compose logs -f user-service

# Stop services
docker compose down
```

### Running User Service Locally

For development focused on the User Service:

```bash
cd services/users

# 1. Set up environment variables
# No .env.example ships in this directory - set the variables listed under
# "Environment Variables" below directly (a .env file is picked up automatically
# via godotenv if present)

# 2. Set up PostgreSQL database
createdb cairn_users

# 3. Set up HashiCorp Vault (required for JWT keys)
# Option A: Use Vault in dev mode
vault server -dev

# Option B: Use Docker
docker run -d --name vault -p 8200:8200 \
  -e VAULT_DEV_ROOT_TOKEN_ID=dev-root-token \
  hashicorp/vault:latest server -dev

# 4. Initialize Vault with JWT keys
# See infrastructure/docker/scripts/init-vault.sh for reference

# 5. Run database migrations
make migrate-up

# 6. Run the service
make run
# Or with live reload (requires air)
make dev
```

### Makefile Commands

```bash
# Build
make build                   # Build user-service binary to bin/

# Run
make run                     # Run user service (port 8082)
make dev                     # Run with live reload (requires air)

# Testing
make test                    # Run all tests
make test-coverage           # Generate coverage report (coverage.html)

# Database migrations
make migrate-up              # Apply pending migrations
make migrate-down            # Rollback last migration
make migrate-version         # Check current migration version
make db-create               # Create database and user (requires psql)
make db-drop                 # Drop database

# Code quality
make fmt                     # Format Go code (go fmt ./...)
make vet                     # Run go vet
make lint                    # Run golangci-lint (requires installation)
make deps                    # Download and tidy dependencies

# Docker
make docker-build            # Build Docker image
make docker-run              # Run in Docker container
```

## Architecture

### System Architecture

```
┌──────────────────────────────────────────────────────┐
│              Client Applications                      │
│        (Mobile App, Web App, etc.)                   │
└────────────────────┬─────────────────────────────────┘
                     │ HTTPS
                     ▼
      ┌──────────────────────────────┐
      │      User Service (8082)     │
      │                              │
      │  - Registration              │
      │  - Login/Logout              │
      │  - Token Management          │
      │  - User Profile              │
      │  - Account Upgrade           │
      └──────┬───────────────┬───────┘
             │               │
             │               │
       ┌─────▼─────┐   ┌────▼────────┐
       │PostgreSQL │   │ Vault       │
       │(users,    │   │(JWT keys,   │
       │ tokens)   │   │ secrets)    │
       └───────────┘   └─────────────┘

       ┌──────────────────────────────┐
       │   Other Services             │
       │   (Content, Explore, Read)   │
       │                              │
       │   Validate JWT independently │
       │   using public key from Vault│
       └──────────────────────────────┘
```

**Key Principles**:
1. **Stateless Authentication**: JWTs validated without database lookups
2. **Refresh Token Rotation**: New refresh token issued on each use
3. **Vault Integration**: Secure secrets management and JWT key distribution
4. **Authorization Middleware**: Users can only access their own resources
5. **Multi-Device Support**: Email/password accounts work across devices

### Directory Structure

```
services/users/
├── cmd/
│   └── user-service/          # Application entrypoint
│       └── main.go
├── internal/                  # Private application code
│   ├── auth/                 # JWT and token management
│   │   ├── jwt.go           # JWT signing and signing-key management (validation lives in pkg/auth)
│   │   ├── password.go      # Password hashing (bcrypt)
│   │   ├── refresh_token.go # Refresh token generation
│   │   └── vault.go         # Vault integration for keys
│   ├── config/              # Configuration management
│   │   └── config.go
│   ├── database/            # Database connection and repositories
│   │   ├── db.go           # Database connection
│   │   ├── migrate.go      # Migration runner
│   │   ├── user_repository.go
│   │   └── refresh_token_repository.go
│   ├── handlers/            # HTTP request handlers
│   │   ├── auth_handler.go  # Auth endpoints
│   │   ├── user_handler.go  # User management endpoints
│   │   ├── health.go        # Health checks
│   │   └── router.go        # Route setup
│   ├── models/              # Data models
│   │   ├── user.go
│   │   └── refresh_token.go
│   └── services/            # Business logic layer
│       ├── auth_service.go
│       └── user_service.go
├── migrations/             # Database migrations (golang-migrate up/down pairs)
│   ├── 000001_create_users_table.up.sql
│   ├── 000001_create_users_table.down.sql
│   └── ...
├── api/
│   └── openapi.yaml        # OpenAPI 3.0 specification
├── Dockerfile              # Multi-stage Docker build
├── Makefile                # Build and development commands
├── README.md               # Service documentation
└── CLAUDE.md              # This file
```

CORS, rate limiting, security headers, and panic recovery come from the shared `pkg/middleware/` module at the repo root (`sharedmw.CORS`, `sharedmw.RateLimit`, `sharedmw.SecureHeadersRelaxed`, `sharedmw.Recovery` in `router.go`), not from a package inside `services/users/`. Likewise, JWT validation for other services is the shared `pkg/auth/` module at the repo root — there is no `pkg/` directory inside `services/users/`.

## Data Models

### Database Schema

**users** table:
```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE,            -- Email (nullable for mobile-only)
    password_hash VARCHAR(255),           -- bcrypt hash (nullable for mobile-only)
    expo_device_id VARCHAR(255) UNIQUE,   -- Expo device ID (nullable for email-only)
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_login_at TIMESTAMP WITH TIME ZONE,
    email_verified BOOLEAN NOT NULL DEFAULT FALSE,
    failed_login_attempts INTEGER NOT NULL DEFAULT 0,
    locked_until TIMESTAMP WITH TIME ZONE,
    last_failed_login_at TIMESTAMP WITH TIME ZONE
);

-- Indexes
CREATE INDEX idx_users_email ON users(email) WHERE email IS NOT NULL;
CREATE INDEX idx_users_expo_device_id ON users(expo_device_id) WHERE expo_device_id IS NOT NULL;
```

**refresh_tokens** table:
```sql
CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL,     -- SHA-256 hash of refresh token
    device_info TEXT,
    ip_address VARCHAR(45),
    token_family UUID,                    -- Groups tokens from the same rotation chain (reuse detection)
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_used_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    revoked_at TIMESTAMP WITH TIME ZONE   -- Set on rotation/logout; NULL = active. Row is kept until expiry so replay can be detected as reuse.
);

-- Indexes
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);
CREATE INDEX idx_refresh_tokens_token_family ON refresh_tokens(token_family) WHERE token_family IS NOT NULL;
```

There is also an `email_verification_tokens` table (added in migration 000005) backing the email verification endpoints below. A `password_reset_tokens` table existed briefly (migration 000006) but was dropped (migration 000008) — see the note under Password reset below.

### Account Types

Account types are determined by which fields are populated (no separate `account_type` field):

**Mobile-only account**:
- `expo_device_id` is NOT NULL
- `email` and `password_hash` are NULL
- Single-device only
- Requires app reinstall to recover

**Email-only account**:
- `email` and `password_hash` are NOT NULL
- `expo_device_id` is NULL
- Multi-device support
- Password recovery available

**Hybrid account** (after upgrade):
- All three fields (`email`, `password_hash`, `expo_device_id`) are NOT NULL
- Multi-device support
- MUST use email/password for login (device ID login rejected)

## API Endpoints

### Health Checks
```bash
# Liveness check
curl http://localhost:8082/health/live

# Readiness check (includes DB and Vault connectivity)
curl http://localhost:8082/health/ready
```

### Authentication Endpoints

**Register with email/password**:
```bash
curl -X POST http://localhost:8082/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "securepass123"
  }'

# Response:
{
  "user_id": "uuid-here",
  "access_token": "eyJhbGciOiJSUzI1NiIs...",
  "refresh_token": "refresh-token-here",
  "expires_in": 3600
}
```

**Register mobile-only account**:
```bash
curl -X POST http://localhost:8082/api/v1/auth/register/mobile \
  -H "Content-Type: application/json" \
  -d '{
    "expo_device_id": "device-id-from-expo-application"
  }'

# Response: Same as above
```

**Login with email/password**:
```bash
curl -X POST http://localhost:8082/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "securepass123"
  }'
```

**Login with device ID**:
```bash
curl -X POST http://localhost:8082/api/v1/auth/login/mobile \
  -H "Content-Type: application/json" \
  -d '{
    "expo_device_id": "device-id-from-expo-application"
  }'
```

**Refresh access token**:
```bash
curl -X POST http://localhost:8082/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "refresh-token-here"
  }'

# Response: New access token and new refresh token (rotation)
```

**Logout (revoke refresh token)**:
```bash
curl -X POST http://localhost:8082/api/v1/auth/logout \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <JWT>" \
  -d '{
    "refresh_token": "refresh-token-here"
  }'
```

**Logout all devices (revoke all refresh tokens)**:
```bash
curl -X POST http://localhost:8082/api/v1/auth/logout-all \
  -H "Authorization: Bearer <JWT>"
```

**Email verification**:
```bash
# Resend verification email (requires auth)
curl -X POST http://localhost:8082/api/v1/auth/resend-verification \
  -H "Authorization: Bearer <JWT>"

# Verify email with token (also accepts GET with ?token= for email links)
curl -X POST http://localhost:8082/api/v1/auth/verify-email \
  -H "Content-Type: application/json" \
  -d '{"token": "verification-token-here"}'
```

**Password reset**: not implemented. `/forgot-password` and `/reset-password` were removed (H3+H4, `epic_fefa`) — the tokens they generated were never delivered to the user by any mailer, and the self-host build panicked on both routes because it never wired up the token repository. Rather than ship a route that appears to work but silently does nothing, the routes were deleted; users with an email/password account who forget their password have no self-service recovery today. Account deletion + re-registration is the only path until a real delivery mechanism (and the routes) are added back deliberately.

### User Management Endpoints (Authenticated)

**Get user profile**:
```bash
curl -H "Authorization: Bearer <JWT>" \
  http://localhost:8082/api/v1/user/{user_id}

# Response:
{
  "id": "user-id",
  "email": "user@example.com",
  "created_at": "2025-01-09T10:00:00Z",
  "last_login_at": "2025-01-09T11:00:00Z"
}
```

**Update user profile**:
```bash
curl -X PATCH http://localhost:8082/api/v1/user/{user_id} \
  -H "Authorization: Bearer <JWT>" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "newemail@example.com"
  }'
```

**Upgrade mobile-only account to email/password**:
```bash
curl -X POST http://localhost:8082/api/v1/user/{user_id}/upgrade \
  -H "Authorization: Bearer <JWT>" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "securepass123"
  }'

# Note: After upgrade, device ID login is disabled
# User must authenticate with email/password
```

**Change password**:
```bash
curl -X PUT http://localhost:8082/api/v1/user/{user_id}/password \
  -H "Authorization: Bearer <JWT>" \
  -H "Content-Type: application/json" \
  -d '{
    "current_password": "securepass123",
    "new_password": "newsecurepass123"
  }'
```

**Delete account**:
```bash
curl -X DELETE http://localhost:8082/api/v1/user/{user_id} \
  -H "Authorization: Bearer <JWT>"
```

## Key Implementation Details

### JWT Authentication (`internal/auth/jwt.go`)

`internal/auth/jwt.go` (`JWTManager`) only **signs** access tokens and manages the
signing key pair (rotation via `UpdateKeys`). It does **not** validate tokens —
the users service validates through the shared `pkg/auth` `Validator` (wired onto
the `/api/v1/user` route group in `router.go`), exactly like every other service.

**Token Generation**:
- Algorithm: RS256 (2048-bit RSA keys), with a `kid` header identifying the signing key for rotation
- Lifetime: 15 minutes (configurable via `JWT_ACCESS_TOKEN_EXPIRY`; code default is 60 minutes if unset)
- Claims: `user_id` (custom) plus standard registered claims - `iss`, `aud`, `sub`, `iat`, `exp`, `nbf`
- Keys stored in HashiCorp Vault

**Token Validation** (`pkg/auth`):
- Stateless validation using public key
- No database lookups required
- Issuer (`cairn-user-service`) and audience (`cairn-api`) are validated, not just the signature
- Other services validate independently using shared public key

**JWT Structure**:
```json
{
  "user_id": "uuid-here",
  "iss": "cairn-user-service",
  "aud": ["cairn-api"],
  "sub": "uuid-here",
  "iat": 1704801600,
  "nbf": 1704801600,
  "exp": 1704805200
}
```

### Refresh Token Management (`internal/auth/refresh_token.go`)

**Token Generation**:
- Cryptographically random 32-byte token (base64-encoded)
- SHA-256 hashed before database storage
- Lifetime: 7 days (configurable via `JWT_REFRESH_TOKEN_EXPIRY`; code default is 30 days if unset)

**Token Rotation**:
1. Client sends refresh token to `/auth/refresh`
2. Service validates token hash in database
3. Service issues new access token AND new refresh token
4. Old refresh token is marked revoked (`revoked_at` set; the row is retained
   until its natural expiry, not deleted)
5. New refresh token returned to client

**Why Rotation?**
- Limits window of exposure if refresh token is compromised
- Enables detection of token theft: replaying an already-revoked token (any
  token whose `revoked_at` is set) is treated as reuse and revokes the whole
  token family, regardless of how long ago it was revoked
- Follows OAuth 2.0 best practices

**Metadata Tracking**:
- `device_info`: User agent string
- `ip_address`: Client IP address
- `last_used_at`: Updated on each refresh
- Enables security monitoring and suspicious activity detection

### Password Security (`internal/auth/password.go`)

**Hashing**:
- Algorithm: bcrypt
- Cost factor: 12 (configurable, minimum 12 for production)
- Password requirements:
  - Minimum length: 8 characters
  - Complexity required by default (uppercase, lowercase, digit, and special character; toggle via `REQUIRE_PASSWORD_COMPLEXITY`)

**Validation**:
```go
// Hash password on registration
hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)

// Verify password on login
err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password))
```

### HashiCorp Vault Integration (`internal/auth/vault.go`)

**Purpose**:
- Secure storage of JWT private/public key pairs
- Distribution of JWT public key to other services
- Secret rotation capabilities

**Vault Operations**:
1. **Generate RSA Key Pair** (2048-bit):
   ```bash
   openssl genrsa -out private.pem 2048
   openssl rsa -in private.pem -pubout -out public.pem
   ```

2. **Store in Vault**:
   ```bash
   vault kv put secret/jwt/private-key value=@private.pem
   vault kv put secret/jwt/public-key value=@public.pem
   ```

3. **Retrieve Keys** (on service startup):
   ```go
   privateKey, err := vault.GetPrivateKey()
   publicKey, err := vault.GetPublicKey()
   ```

**Environment Variables**:
```bash
VAULT_ADDR=http://localhost:8200
VAULT_TOKEN=dev-root-token
JWT_PRIVATE_KEY_PATH=secret/data/jwt/private-key
JWT_PUBLIC_KEY_PATH=secret/data/jwt/public-key
```

### Authorization

**Purpose**: Ensures users can only access their own resources

**Flow** (split across two layers, not a single middleware):
1. `pkg/auth`'s `RequireAuth` middleware (wired onto the `/api/v1/user` route group in `router.go`) extracts the JWT from `Authorization: Bearer <token>`, validates its RS256 signature against the public key, and stores `user_id` from the claims on the request context.
2. Each `UserHandler` method (`internal/handlers/user_handler.go`) reads that authenticated `user_id` and the `{user_id}` path parameter, then calls into `UserService`, passing both. The service layer compares them and returns `services.ErrUnauthorized`, which the handler maps to `403 Forbidden`, on mismatch.

There is no `services/users` middleware package — same-user enforcement lives entirely in the service layer as described above.

**Example**:
```go
// Request: GET /api/v1/user/user-123
// JWT claims: {"user_id": "user-456", ...}
// Result: 403 Forbidden (user-456 cannot access user-123)

// Request: GET /api/v1/user/user-456
// JWT claims: {"user_id": "user-456", ...}
// Result: 200 OK (user can access own data)
```

### Rate Limiting (`pkg/middleware/rate_limit.go`)

**Purpose**: Prevent brute force attacks on authentication endpoints

**Configuration**:
- Applies to the entire `/api/v1/auth/*` route group per IP (single limit, not split by endpoint); `/api/v1/user/*` endpoints are not rate limited
- Limit and window are configurable via `RATE_LIMIT_REQUESTS` / `RATE_LIMIT_WINDOW` (default: 100 requests per 1 minute); the router falls back to 10/minute only if `RATE_LIMIT_REQUESTS` resolves to 0
- Uses in-memory store (consider Redis for multi-instance deployments)

**Protected Endpoints** (all routes under `/api/v1/auth`):
- `/register`, `/register/mobile`, `/login`, `/login/mobile`, `/refresh`, `/logout`, `/logout-all`
- `/verify-email`, `/resend-verification`

### Mobile Device Authentication

**Expo Device ID**:
- Obtained via `expo-application` package: `Application.androidId` or `Application.getIosIdForVendorAsync()`
- Unique per app installation
- Treated as credential (transmitted over HTTPS only)

**Security Considerations**:
- Loss of device ID (app reinstall) requires new account creation
- Single-device only (no multi-device sync)
- Users requiring multi-device access must upgrade to email/password

**Account Upgrade Flow**:
1. Mobile-only user requests upgrade via `/user/{id}/upgrade`
2. Service validates email is unique
3. Service hashes password and stores email/password
4. Account becomes hybrid (all three fields populated)
5. Future logins MUST use email/password (device ID login rejected)

**Why disable device ID login after upgrade?**
- Ensures users have recoverable credentials
- Prevents single-device-only access pattern
- Enforces proper multi-device authentication

## Testing and Development Workflow

### Running Tests

```bash
# All tests
make test

# With coverage
make test-coverage

# Specific package
go test -v ./internal/auth/...
go test -v ./internal/handlers/...

# Table-driven tests with verbose output
go test -v ./internal/services/...

# Run with race detection
go test -race ./...
```

### Test Organization

**Unit Tests** (no external dependencies):
- `*_test.go` files alongside source code
- Mock database with `testify/mock` or `sqlmock`
- Test coverage target: 80%+

**Test Files**:
```
internal/
├── auth/
│   ├── jwt.go
│   ├── jwt_test.go
│   ├── password.go
│   ├── password_test.go
│   └── ...
├── handlers/
│   ├── auth_handler.go
│   ├── auth_handler_test.go
│   └── ...
└── services/
    ├── auth_service.go
    ├── auth_service_test.go
    └── ...
```

**Testing Patterns**:
```go
// Table-driven tests
func TestHashPassword(t *testing.T) {
    tests := []struct {
        name     string
        password string
        wantErr  bool
    }{
        {"valid password", "securepass123", false},
        {"empty password", "", true},
        {"short password", "short", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            hash, err := HashPassword(tt.password)
            if (err != nil) != tt.wantErr {
                t.Errorf("HashPassword() error = %v, wantErr %v", err, tt.wantErr)
            }
            if !tt.wantErr && hash == "" {
                t.Error("HashPassword() returned empty hash")
            }
        })
    }
}
```

### Testing Authentication Flow

**1. Register user**:
```bash
curl -X POST http://localhost:8082/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"testpass123"}'
```

**2. Save tokens** from response:
```bash
export ACCESS_TOKEN="eyJhbGci..."
export REFRESH_TOKEN="refresh-token-here"
```

**3. Test protected endpoint**:
```bash
curl -H "Authorization: Bearer $ACCESS_TOKEN" \
  http://localhost:8082/api/v1/user/{user_id}
```

**4. Test token refresh**:
```bash
curl -X POST http://localhost:8082/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d "{\"refresh_token\":\"$REFRESH_TOKEN\"}"
```

**5. Test authorization** (should fail with 403):
```bash
curl -H "Authorization: Bearer $ACCESS_TOKEN" \
  http://localhost:8082/api/v1/user/different-user-id
```

### Database Operations

**Run migrations**:
```bash
make migrate-up
```

**Rollback migration**:
```bash
make migrate-down
```

**Check migration status**:
```bash
make migrate-version
```

**Create new migration**:
```bash
# Add a numbered up/down pair to migrations/ (golang-migrate convention)
# migrations/000007_add_new_column.up.sql
ALTER TABLE users ADD COLUMN new_column TEXT;

# migrations/000007_add_new_column.down.sql
ALTER TABLE users DROP COLUMN new_column;
```

**Access database directly**:
```bash
psql -U cairn_user -d cairn_users

# View users
SELECT id, email, expo_device_id, created_at FROM users;

# View active refresh tokens
SELECT user_id, device_info, created_at, expires_at
FROM refresh_tokens
WHERE expires_at > NOW();
```

## Code Conventions

### Go Code Style

**Error Handling**:
```go
// Wrap errors with context using %w
if err != nil {
    return fmt.Errorf("failed to create user: %w", err)
}

// Return errors up the call stack
// Let handlers format HTTP responses
func (s *AuthService) Register(email, password string) (*User, error) {
    // ... business logic ...
    if err != nil {
        return nil, fmt.Errorf("register user: %w", err)
    }
    return user, nil
}
```

**Database Operations**:
```go
// Always use context.Context for database calls
func (r *UserRepository) GetByID(ctx context.Context, id string) (*User, error) {
    // Implementation
}

// Use transactions for multi-step operations
tx, err := r.db.BeginTx(ctx, nil)
if err != nil {
    return fmt.Errorf("begin transaction: %w", err)
}
defer tx.Rollback() // Safe to call even after Commit

// ... perform operations with tx ...

if err := tx.Commit(); err != nil {
    return fmt.Errorf("commit transaction: %w", err)
}
```

**Password Validation**:
```go
// Validate before hashing
if len(password) < 8 {
    return nil, errors.New("password must be at least 8 characters")
}

// Hash with appropriate cost
hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
```

**JWT Handling**:
```go
// Create token with all required claims
claims := jwt.MapClaims{
    "user_id": user.ID,
    "iat":     time.Now().Unix(),
    "exp":     time.Now().Add(tokenLifetime).Unix(),
}

// Validate and extract claims
token, err := jwt.ParseWithClaims(tokenString, &jwt.MapClaims{}, keyFunc)
if err != nil {
    return "", fmt.Errorf("parse token: %w", err)
}
```

### Repository Pattern

**Interface**:
```go
type UserRepository interface {
    Create(ctx context.Context, user *User) error
    GetByID(ctx context.Context, id string) (*User, error)
    GetByEmail(ctx context.Context, email string) (*User, error)
    GetByDeviceID(ctx context.Context, deviceID string) (*User, error)
    Update(ctx context.Context, user *User) error
    Delete(ctx context.Context, id string) error
}
```

**Implementation** (`internal/database/user_repository.go`):
- All database access isolated in repository files
- Handlers never directly import database code
- Services inject repository interfaces
- Enables easy mocking for tests

### Service Layer Pattern

**Purpose**: Business logic layer between handlers and repositories

**Structure**:
```go
type AuthService struct {
    userRepo         UserRepository
    refreshTokenRepo RefreshTokenRepository
    jwtManager       *JWTManager
    passwordHasher   PasswordHasher
}

func (s *AuthService) Register(ctx context.Context, email, password string) (*AuthResponse, error) {
    // 1. Validate input
    // 2. Check if user exists
    // 3. Hash password
    // 4. Create user
    // 5. Generate tokens
    // 6. Return response
}
```

**Benefits**:
- Separates business logic from HTTP layer
- Testable without HTTP context
- Reusable across different interfaces (REST, GraphQL, gRPC)

### Handler Pattern

**Structure**:
```go
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
    // 1. Parse request body
    var req RegisterRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    // 2. Call service layer
    resp, err := h.authService.Register(r.Context(), req.Email, req.Password)
    if err != nil {
        // Map service errors to HTTP status codes
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // 3. Return JSON response
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(resp)
}
```

## Environment Variables

```bash
# Server Configuration
PORT=8080                                    # HTTP server port (default: 8080; dev docker-compose maps host 8082 -> container 8080)
ENVIRONMENT=development                      # Environment mode (development, production)

# Database Configuration
DB_HOST=localhost                            # PostgreSQL host
DB_PORT=5432                                 # PostgreSQL port
DB_USER=cairn_user                           # Database user
DB_PASSWORD=secure_password                  # Database password
DB_NAME=cairn_users                          # Database name
DB_SSLMODE=disable                           # SSL mode (disable for local dev)

# HashiCorp Vault Configuration (REQUIRED)
VAULT_ADDR=http://localhost:8200             # Vault address
VAULT_TOKEN=dev-root-token                   # Vault token
JWT_PRIVATE_KEY_PATH=secret/data/jwt/private-key  # Vault path to private key
JWT_PUBLIC_KEY_PATH=secret/data/jwt/public-key    # Vault path to public key

# JWT Configuration
JWT_ACCESS_TOKEN_EXPIRY=15m                  # Access token lifetime (code default: 60 minutes if unset)
JWT_REFRESH_TOKEN_EXPIRY=7d                  # Refresh token lifetime (code default: 30 days if unset)

# Password Security
BCRYPT_COST=12                               # bcrypt cost factor (minimum 12)
MIN_PASSWORD_LENGTH=8                        # Minimum password length
REQUIRE_PASSWORD_COMPLEXITY=true             # Require upper/lower/digit/special char (code default: true)

# Rate Limiting (applies to the /api/v1/auth/* route group only; /api/v1/user/* is not rate limited)
RATE_LIMIT_REQUESTS=100                      # Requests per window per IP (code default: 100)
RATE_LIMIT_WINDOW=1m                         # Rate limit window (code default: 1 minute)
```

Note: CORS is currently hardcoded to a permissive development configuration (`pkg/middleware.DefaultCORSConfig()`, allow-origin `*`) and is not configurable via environment variables for this service.

## Common Development Tasks

### Adding a New Endpoint

**1. Define handler**:
```go
// internal/handlers/user_handler.go
func (h *UserHandler) NewEndpoint(w http.ResponseWriter, r *http.Request) {
    // Implementation
}
```

**2. Register route**:
```go
// internal/handlers/router.go
r.Route("/api/v1/user", func(r chi.Router) {
    r.Use(middleware.RequireAuth) // Apply auth middleware
    r.Get("/{id}/new-endpoint", h.NewEndpoint)
})
```

**3. Add service method**:
```go
// internal/services/user_service.go
func (s *UserService) NewOperation(ctx context.Context, userID string) error {
    // Business logic
}
```

**4. Add repository method if needed**:
```go
// internal/database/user_repository.go
func (r *UserRepository) NewQuery(ctx context.Context, id string) (*User, error) {
    // Database access
}
```

**5. Write tests**:
```go
// internal/handlers/user_handler_test.go
func TestNewEndpoint(t *testing.T) {
    // Test implementation
}
```

**6. Update OpenAPI spec**:
```yaml
# api/openapi.yaml
paths:
  /api/v1/user/{id}/new-endpoint:
    get:
      summary: Description
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
      responses:
        '200':
          description: Success
```

### Adding Database Migration

**1. Create migration files** (golang-migrate up/down pair):
```sql
-- migrations/000007_add_display_name.up.sql
ALTER TABLE users ADD COLUMN display_name VARCHAR(255);
```
```sql
-- migrations/000007_add_display_name.down.sql
ALTER TABLE users DROP COLUMN display_name;
```

**2. Test migration**:
```bash
# Apply migration
make migrate-up

# Verify schema
psql -U cairn_user -d cairn_users -c "\d users"

# Rollback if needed
make migrate-down
```

**3. Update models**:
```go
// internal/models/user.go
type User struct {
    ID           uuid.UUID
    Email        *string
    PasswordHash *string
    ExpoDeviceID *string
    DisplayName  *string    // New field
    CreatedAt    time.Time
    UpdatedAt    time.Time
}
```

### Adding Middleware

Cross-cutting middleware (CORS, rate limiting, recovery, auth) lives in the repo-root
`pkg/middleware` and `pkg/auth` modules and is wired in `internal/handlers/router.go`.
Add service-specific middleware there or in a new `internal/middleware` package.

**1. Create middleware function**:
```go
// internal/middleware/my_middleware.go (new package)
func MyMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Pre-processing

        next.ServeHTTP(w, r)

        // Post-processing
    })
}
```

**2. Apply to routes**:
```go
// internal/handlers/router.go
r.Use(middleware.MyMiddleware)

// Or apply to specific routes
r.Route("/api/v1/protected", func(r chi.Router) {
    r.Use(middleware.MyMiddleware)
    r.Get("/resource", handler)
})
```

**3. Write tests**:
```go
// internal/middleware/my_middleware_test.go
func TestMyMiddleware(t *testing.T) {
    // Test implementation
}
```

### Debugging

**View service logs**:
```bash
# Development (stdout)
make run

# Docker Compose
docker compose logs -f user-service

# Follow last 100 lines
docker compose logs --tail=100 user-service
```

**Test JWT validation**:
```bash
# Decode JWT (header.payload.signature)
echo "eyJhbGciOiJSUzI1NiIs..." | cut -d'.' -f2 | base64 -d | jq

# Validate JWT using online tools (jwt.io) or:
# Use public key from Vault to verify signature
```

**Check database state**:
```bash
psql -U cairn_user -d cairn_users

-- View users
SELECT id, email, expo_device_id, created_at, last_login_at FROM users;

-- View active refresh tokens
SELECT
    rt.id,
    rt.user_id,
    u.email,
    rt.device_info,
    rt.created_at,
    rt.last_used_at,
    rt.expires_at
FROM refresh_tokens rt
JOIN users u ON rt.user_id = u.id
WHERE rt.expires_at > NOW()
ORDER BY rt.created_at DESC;

-- View expired tokens
SELECT COUNT(*) FROM refresh_tokens WHERE expires_at < NOW();
```

**Test Vault connectivity**:
```bash
# Check Vault status
curl http://localhost:8200/v1/sys/health

# Read JWT public key
vault kv get secret/jwt/public-key

# Or via API
curl -H "X-Vault-Token: dev-root-token" \
  http://localhost:8200/v1/secret/data/jwt/public-key
```

## Important Notes

### HashiCorp Vault Dependency

**CRITICAL**: The User Service **requires** HashiCorp Vault for JWT key management.

**What Vault is used for:**
- Generates and stores RS256 JWT signing keys (2048-bit RSA)
- Distributes JWT public key to other services (Content, Explore, Read)
- Enables key rotation without service downtime
- Secure secret storage in production

**Development Setup:**

The centralized Docker Compose setup ([infrastructure/docker/dev/docker-compose.yml](../../infrastructure/docker/dev/docker-compose.yml)) includes:
1. Vault container running in dev mode (port 8200)
2. Automated `vault-init` service that generates RSA keys and stores them in Vault
3. All services configured to use the shared Vault instance

**Running without Docker Compose:**

1. Start Vault in dev mode:
   ```bash
   docker run -d --name vault -p 8200:8200 \
     -e VAULT_DEV_ROOT_TOKEN_ID=dev-root-token \
     hashicorp/vault:latest server -dev
   ```

2. Initialize Vault with JWT keys:
   ```bash
   # Generate RSA key pair
   openssl genrsa -out private.pem 2048
   openssl rsa -in private.pem -pubout -out public.pem

   # Store in Vault
   export VAULT_ADDR=http://localhost:8200
   export VAULT_TOKEN=dev-root-token

   vault kv put secret/jwt/private-key value=@private.pem
   vault kv put secret/jwt/public-key value=@public.pem
   ```

3. Configure User Service:
   ```bash
   export VAULT_ADDR=http://localhost:8200
   export VAULT_TOKEN=dev-root-token
   ```

**Production**: Use properly configured Vault cluster with:
- Persistent storage (not dev mode)
- TLS encryption
- Proper authentication (AppRole, Kubernetes, etc.)
- High availability setup
- Audit logging enabled

### Security Best Practices

**Password Security**:
- Minimum bcrypt cost: 12 (production)
- Never log passwords or hashes
- Enforce minimum password length (8+ characters)
- Password complexity (upper/lower/digit/special char) required by default; toggle via `REQUIRE_PASSWORD_COMPLEXITY`

**JWT Security**:
- RS256 algorithm only (not HS256)
- Short access token lifetime (15 minutes)
- Longer refresh token lifetime (7 days)
- Validate token signature, expiration, and claims

**Refresh Token Security**:
- SHA-256 hash before database storage
- Rotate on each use (issue new token)
- Track device_info and ip_address
- Implement token reuse detection
- Revoke all tokens on suspected compromise

**Transport Security**:
- HTTPS required in production (`RequireHTTPS` middleware)
- Tokens are returned in the JSON response body, not as cookies
- CORS currently hardcoded to a permissive dev configuration (see Environment Variables note above)
- Security headers (CSP, HSTS, X-Frame-Options)

**Rate Limiting**:
- 100 requests/minute (default) on the `/api/v1/auth/*` route group, IP-based (prevent brute force)
- `/api/v1/user/*` endpoints are not rate limited
- Use Redis for distributed rate limiting (not yet implemented; in-memory only today)

### Cross-Service Integration

**How other services validate JWTs**:

1. **Fetch public key from Vault** (on startup):
   ```go
   publicKey, err := vault.GetPublicKey()
   ```

2. **Validate JWT** (on each request):
   ```go
   token, err := jwt.ParseWithClaims(tokenString, &jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
       return publicKey, nil
   })
   ```

3. **Extract user_id** from claims:
   ```go
   claims := token.Claims.(*jwt.MapClaims)
   userID := (*claims)["user_id"].(string)
   ```

**Shared Auth Package** (`pkg/auth/`):
- Lightweight JWT validation library
- Middleware for protecting endpoints
- User context extraction utilities
- Distributed to other services as Go module

### Account Type Validation

**Mobile-only account**:
```go
func (u *User) IsMobileOnly() bool {
    return u.ExpoDeviceID != nil && u.Email == nil && u.PasswordHash == nil
}
```

**Email-only account**:
```go
func (u *User) IsEmailOnly() bool {
    return u.Email != nil && u.PasswordHash != nil && u.ExpoDeviceID == nil
}
```

**Hybrid account** (after upgrade):
```go
func (u *User) IsHybrid() bool {
    return u.Email != nil && u.PasswordHash != nil && u.ExpoDeviceID != nil
}
```

**Login validation**:
- Mobile-only and email-only: Accept respective auth method
- Hybrid: Accept ONLY email/password (reject device ID login)

### Token Cleanup

Already implemented: `main.go` runs `tokenCleanupScheduler` in the background, which calls `RefreshTokenService.CleanupExpiredTokens` every hour (with an initial run at startup) to delete expired refresh tokens.

## Technology Stack

### Core Dependencies

```go
// HTTP routing and middleware
github.com/go-chi/chi/v5               // HTTP router
// CORS is a hand-rolled middleware in pkg/middleware, not a third-party package

// JWT handling
github.com/golang-jwt/jwt/v5           // JWT creation and validation

// Password hashing
golang.org/x/crypto/bcrypt             // bcrypt password hashing

// Database
github.com/jackc/pgx/v5                // PostgreSQL driver (github.com/lib/pq is only an indirect dependency)
github.com/golang-migrate/migrate/v4   // Schema migrations

// HashiCorp Vault
github.com/hashicorp/vault/api         // Vault client

// Configuration
github.com/joho/godotenv               // .env file loading

// Testing
github.com/stretchr/testify            // Testing framework and assertions
github.com/golang/mock                 // Interface mocking for tests
```

## Current Implementation Status

**✅ COMPLETE - All Core Features Implemented**:

**Phase 1: Core Infrastructure** ✅
- ✅ Database schema and migrations
- ✅ User and refresh token repositories
- ✅ Configuration management
- ✅ Vault integration for JWT keys

**Phase 2: Authentication** ✅
- ✅ JWT generation and validation (RS256)
- ✅ Password hashing (bcrypt)
- ✅ Refresh token generation and rotation
- ✅ Email/password registration and login
- ✅ Mobile device registration and login

**Phase 3: User Management** ✅
- ✅ User profile retrieval
- ✅ User profile updates
- ✅ Account upgrade (mobile → email/password)
- ✅ Account deletion

**Phase 4: Security** ✅
- ✅ Authorization middleware
- ✅ Rate limiting
- ✅ CORS configuration
- ✅ Security headers
- ✅ Panic recovery

**Phase 5: Testing** ✅
- ✅ Unit tests for all components
- ✅ Integration tests for critical flows
- ✅ Test coverage > 80%

**Phase 6: Documentation** ✅
- ✅ README.md
- ✅ requirements.md
- ✅ OpenAPI specification
- ✅ CLAUDE.md (this file)

### Future Enhancements

Note: Email verification, password reset, refresh token reuse detection, and account lockout after failed attempts are already implemented (see API Endpoints and Data Models above) - they are not listed here.

**Medium Priority**:
- Multi-factor authentication (MFA)
- OAuth2/OpenID Connect support
- Session management improvements

**Low Priority**:
- Remember device functionality
- Audit logging
- Advanced rate limiting (Redis-based)
- Metrics and observability

## Documentation References

- **Main Project CLAUDE.md**: [/CLAUDE.md](/CLAUDE.md) - Project-wide context and conventions
- **Engineering Principles**: [/docs/ENGINEERING_PRINCIPLES.md](/docs/ENGINEERING_PRINCIPLES.md) - Standards and best practices
- **Service README**: [README.md](README.md) - Comprehensive service documentation
- **Requirements**: [requirements.md](/docs/detailed_requirements/users_service_requirements.md) - Detailed requirements and specifications
- **OpenAPI Spec**: [api/openapi.yaml](api/openapi.yaml) - Formal API specification

## Getting Help

For issues or questions:
- **Check service logs**: `docker compose logs -f user-service`
- **Review test cases**: Look at `*_test.go` files for usage examples
- **Consult OpenAPI spec**: [api/openapi.yaml](api/openapi.yaml) for API reference
- **Check database state**: `psql -U cairn_user -d cairn_users`
- **Verify Vault connectivity**: `curl http://localhost:8200/v1/sys/health`
- **Main documentation**: [/CLAUDE.md](/CLAUDE.md) and [/docs/ENGINEERING_PRINCIPLES.md](/docs/ENGINEERING_PRINCIPLES.md)

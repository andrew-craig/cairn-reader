# Cairn User Service - Implementation Plan

## Phase 1: Project Setup & Infrastructure

### 1.1 Project Structure
- [x] Initialize Go module for user service
- [x] Set up project directory structure (cmd, internal, pkg)
- [x] Create internal package structure (handlers, models, middleware, auth, database)
- [x] Set up configuration management (environment variables, config struct)
- [x] Add .env.example file with required configuration

### 1.2 Dependencies
- [x] Add core dependencies (go.mod):
  - Gin or Echo web framework
  - JWT library (golang-jwt/jwt)
  - Database driver (pgx for PostgreSQL)
  - Migration tool (golang-migrate)
  - bcrypt for password hashing
  - UUID library
  - HashiCorp Vault Go client
- [x] Add development dependencies (testing, mocking)
- [x] Set up dependency management and versioning

### 1.3 Database Setup
- [x] Create PostgreSQL schema design
- [x] Write initial migration for users table
- [x] Write initial migration for refresh_tokens table
- [x] Add indexes (email, expo_device_id, token_hash, expires_at)
- [x] Add foreign key constraints
- [x] Create database connection pool configuration
- [x] Implement database health check endpoint

## Phase 2: Data Layer

### 2.1 Database Models
- [x] Define User model struct with all fields
- [x] Define RefreshToken model struct with all fields
- [x] Add JSON tags and validation tags
- [x] Implement helper methods for account type detection (IsMobileOnly, IsEmailOnly, IsHybrid)

### 2.2 User Repository
- [x] Create UserRepository interface
- [x] Implement CreateUser (for email/password registration)
- [x] Implement CreateMobileUser (for device-only registration)
- [x] Implement GetUserByID
- [x] Implement GetUserByEmail
- [x] Implement GetUserByExpoDeviceID
- [x] Implement UpdateUser
- [x] Implement UpgradeAccount (add email/password to mobile account)
- [x] Implement DeleteUser
- [x] Implement UpdateLastLoginAt
- [x] Add proper error handling and SQL injection prevention

### 2.3 Refresh Token Repository
- [x] Create RefreshTokenRepository interface
- [x] Implement CreateRefreshToken
- [x] Implement GetRefreshTokenByHash
- [x] Implement UpdateLastUsedAt
- [x] Implement RevokeToken (delete specific token)
- [x] Implement RevokeAllUserTokens (delete all tokens for user)
- [x] Implement CleanupExpiredTokens (scheduled cleanup)
- [x] Add token family tracking for rotation

## Phase 3: Security & Authentication

### 3.1 HashiCorp Vault Integration
- [x] Set up Vault client configuration
- [x] Implement Vault authentication (AppRole or appropriate method)
- [x] Create secrets retrieval functions
- [x] Implement JWT key pair retrieval from Vault
- [x] Implement database credentials retrieval from Vault
- [x] Add Vault connection health check
- [x] Implement key rotation handling

### 3.2 Password Security
- [x] Implement password hashing with bcrypt (cost factor 12)
- [x] Create password validation function (length, complexity)
- [x] Implement secure password comparison
- [x] Add password strength checker

### 3.3 JWT Token Management
- [x] Load RSA private/public key pair from Vault
- [x] Implement JWT token generation with RS256
- [x] Add standard claims (user_id, iat, exp, nbf, iss, aud, sub)
- [x] Implement JWT token validation
- [x] Create token parsing utilities
- [x] Add token expiration configuration (default 60 minutes)

### 3.4 Refresh Token Management
- [x] Implement cryptographically secure random token generation
- [x] Implement token hashing (SHA-256)
- [x] Add token rotation logic (new token on refresh)
- [x] Implement refresh token reuse detection
- [x] Add token family tracking
- [x] Implement automatic revocation on compromise detection
- [x] Configure token lifetime (default 30 days)

## Phase 4: Business Logic & Services

### 4.1 Authentication Service
- [x] Create AuthService interface
- [x] Implement Register (email/password)
- [x] Implement RegisterMobile (expo device ID)
- [x] Implement Login (email/password)
- [x] Implement LoginMobile (expo device ID)
- [x] Implement RefreshAccessToken
- [x] Implement Logout (single token revocation)
- [x] Implement LogoutAll (revoke all user tokens)
- [x] Add device info and IP address tracking

### 4.2 User Service
- [x] Create UserService interface
- [x] Implement GetUser (with authorization check)
- [x] Implement UpdateUser (with authorization check)
- [x] Implement UpgradeAccount (mobile to hybrid)
- [x] Implement DeleteUser (with authorization check)
- [x] Add email uniqueness validation
- [x] Add device ID uniqueness validation
- [x] Prevent device ID login for hybrid accounts

## Phase 5: HTTP Layer

### 5.1 Middleware
- [x] Create JWT authentication middleware
- [x] Implement user ID extraction from token
- [x] Create authorization middleware (user can only access own ID)
- [x] Implement rate limiting middleware (authentication endpoints)
- [x] Add HTTPS enforcement middleware
- [x] Create request logging middleware
- [x] Add CORS middleware (if needed)
- [x] Implement panic recovery middleware

### 5.2 Authentication Handlers
- [x] POST /auth/register handler
- [x] POST /auth/register/mobile handler
- [x] POST /auth/login handler
- [x] POST /auth/login/mobile handler
- [x] POST /auth/refresh handler
- [x] POST /auth/logout handler
- [x] POST /auth/logout-all handler
- [x] Add request validation for all endpoints
- [x] Implement proper error responses

### 5.3 User Management Handlers
- [x] GET /users/{id} handler
- [x] PATCH /users/{id} handler
- [x] POST /users/{id}/upgrade handler
- [x] DELETE /users/{id} handler
- [x] Add request validation
- [x] Implement 403 Forbidden for unauthorized access

### 5.4 Health & Utility Endpoints
- [x] GET /health endpoint
- [x] GET /ready endpoint (database + Vault check)
- [ ] Add metrics endpoint (optional)

### 5.5 Application Wiring (cmd/user-service/main.go) - ✅ COMPLETE
**Status**: IMPLEMENTED - Service is now fully wired and runnable
**Priority**: COMPLETE

- [x] Initialize database connection
  - [x] Connect to PostgreSQL using config
  - [x] Set up connection pool with appropriate settings
  - [x] Test database connectivity on startup
  - [x] Implement graceful connection handling
- [x] Initialize Vault client
  - [x] Connect to Vault using config
  - [x] Authenticate with Vault (AppRole or token)
  - [x] Test Vault connectivity on startup
  - [x] Handle Vault connection failures gracefully
- [x] Load secrets from Vault
  - [x] Retrieve JWT key pair (private and public keys)
  - [x] Retrieve database credentials (if using Vault for DB creds)
  - [x] Handle secret retrieval errors
- [x] Run database migrations
  - [x] Execute pending migrations on startup
  - [x] Handle migration errors appropriately
  - [x] Log migration status
- [x] Initialize repositories
  - [x] Create UserRepository instance with DB connection
  - [x] Create RefreshTokenRepository instance with DB connection
- [x] Initialize services
  - [x] Create JWT manager with keys from Vault
  - [x] Create password hasher
  - [x] Create refresh token service with repository
  - [x] Create AuthService with dependencies
  - [x] Create UserService with dependencies
- [x] Initialize handlers
  - [x] Create AuthHandler with AuthService
  - [x] Create UserHandler with UserService
  - [x] Create HealthHandler with DB and Vault clients
- [x] Set up Gin router
  - [x] Replace stdlib ServeMux with Gin router
  - [x] Configure Gin mode (release/debug based on environment)
  - [x] Add global middleware (recovery, logging, CORS, security headers)
- [x] Register all routes
  - [x] Auth routes (POST /auth/register, /auth/login, etc.)
  - [x] User routes (GET/PATCH/DELETE /users/:id, POST /users/:id/upgrade)
  - [x] Health routes (GET /health, /ready)
  - [x] Apply middleware to appropriate route groups
  - [x] Apply rate limiting to auth endpoints
  - [x] Apply JWT auth middleware to protected routes
  - [x] Apply authorization middleware to user routes
- [x] Set up background tasks
  - [x] Start JWT key rotation manager
  - [x] Start expired token cleanup scheduler
  - [x] Ensure graceful shutdown of background tasks
- [x] Implement graceful shutdown
  - [x] Close database connections
  - [x] Stop background tasks
  - [x] Wait for in-flight requests to complete
  - [x] Close Vault client

**Current State**: main.go has full application wiring with:
- Database connection with health checks and connection pooling
- Vault client with authentication and health verification
- JWT key loading from Vault with automatic rotation
- All repositories, services, and handlers properly initialized
- Gin router with all routes and middleware configured
- Background tasks for key rotation and token cleanup
- Graceful shutdown handling for all resources

**Implementation**: See [cmd/user-service/main.go:1-307](cmd/user-service/main.go)

## Phase 6: Shared Authentication Package

### 6.1 JWT Validation Library
- [x] Create separate Go module for shared auth package
- [x] Implement JWT validation function using RS256 public key
- [x] Create middleware for protecting endpoints
- [x] Add user context extraction utilities
- [x] Add public key fetching from Vault
- [x] Include usage examples and documentation
- [x] Publish as internal package or module

## Phase 7: Testing

### 7.1 Unit Tests

#### 7.1.1 Auth Package Tests
**Password Security (internal/auth/password.go)** - ✅ COMPLETE
- [x] NewPasswordHasher constructor tests
- [x] HashPassword tests (valid passwords, edge cases, bcrypt cost verification)
- [x] Hash uniqueness tests (verify random salt)
- [x] ComparePassword tests (correct/incorrect passwords, case sensitivity)
- [x] Invalid hash handling tests
- [x] ValidatePasswordStrength tests (min length, complexity requirements)
- [x] Complexity validation tests (uppercase, lowercase, digits, special chars)
- [x] No complexity mode tests
- [x] Integration tests (full workflow)
- [x] Benchmark tests

**JWT Token Management (internal/auth/jwt.go)** - ✅ COMPLETE
- [x] NewJWTManager constructor tests
- [x] NewJWTManagerWithConfig tests (custom/default issuer and audience)
- [x] GenerateToken tests (valid token generation, claims verification)
- [x] ValidateToken tests (valid, expired, invalid signature, malformed, wrong issuer/audience)
- [x] ParseTokenWithoutValidation tests (expired tokens)
- [x] ExtractTokenFromHeader tests (Bearer format, case insensitive, edge cases)
- [x] GetTokenExpiry tests
- [x] UpdateKeys tests (key rotation)
- [x] GetPublicKey tests
- [x] GetTokenInfo tests (token metadata extraction, expired tokens)
- [x] Token claims verification (all standard claims)
- [x] RS256 signing method verification
- [x] NotBefore claim validation tests
- [x] Benchmark tests

**Refresh Token Service (internal/auth/refresh_token.go)** - ✅ COMPLETE
- [x] NewRefreshTokenService constructor tests
- [x] GenerateToken tests
  - [x] Cryptographically secure random generation
  - [x] Token length verification (32 bytes = 43 char base64)
  - [x] Hash generation (SHA-256)
  - [x] Uniqueness tests (multiple tokens should be different)
- [x] hashToken helper function tests
  - [x] Consistent hashing (same input = same hash)
  - [x] Different inputs produce different hashes
  - [x] SHA-256 output format verification
- [x] CreateRefreshToken tests
  - [x] Valid token creation with all parameters
  - [x] Token family creation (nil vs provided)
  - [x] Device info and IP address handling
  - [x] Expiration time calculation (default 30 days)
  - [x] Database storage verification
  - [x] Error handling (database errors)
- [x] ValidateAndRotateToken tests
  - [x] Valid token rotation flow
  - [x] Token expiration detection
  - [x] Token reuse detection (within grace period)
  - [x] Token family revocation on reuse
  - [x] Fallback to user token revocation (no family)
  - [x] LastUsedAt timestamp update
  - [x] Token not found error
  - [x] Multiple rotation cycles
  - [x] Error handling during all phases
- [x] isTokenReused tests
  - [x] First use (LastUsedAt == CreatedAt) returns false
  - [x] Within grace period (5s) returns true
  - [x] Outside grace period returns false
  - [x] Edge cases around grace period boundary
- [x] RevokeToken tests
  - [x] Single token revocation
  - [x] Token not found error
  - [x] Hash lookup verification
  - [x] Database error handling
- [x] RevokeAllUserTokens tests
  - [x] All user tokens revoked
  - [x] User with no tokens (no error)
  - [x] Database error handling
- [x] RevokeTokenFamily tests
  - [x] All family tokens revoked
  - [x] Invalid UUID handling
  - [x] Non-existent family (no error)
  - [x] Database error handling
- [x] CleanupExpiredTokens tests
  - [x] Expired tokens removed
  - [x] Valid tokens preserved
  - [x] Count verification
  - [x] Empty table handling
  - [x] Database error handling
- [x] Benchmark tests (token generation and hashing)

**Vault Integration (internal/auth/vault.go)** - ✅ COMPLETE (Unit tests - Integration tests require Vault server)
- [x] Key parsing tests (parsePrivateKey and parsePublicKey)
  - [x] PKCS1 and PKCS8 format support
  - [x] Error handling (invalid PEM, wrong key types)
  - [x] Edge cases (empty strings, malformed data)
- [x] Structure tests (VaultConfig, DatabaseCredentials, JWTKeyPair)
- [x] KeyRotationManager tests
  - [x] Thread-safe key pair access
  - [x] Context cancellation handling
  - [x] Start/Stop lifecycle
  - [x] Concurrent access verification
- [x] Configuration validation tests
- [x] Benchmark tests (key parsing performance)
- Note: Full integration tests (NewVaultClient, GetSecret, etc.) require actual Vault server

#### 7.1.2 Models Package Tests
**User Model (internal/models/user.go)** - ✅ COMPLETE
- [x] IsMobileOnly tests (mobile-only, email-only, hybrid)
- [x] IsEmailOnly tests (all account types)
- [x] IsHybrid tests (all account types)
- [x] CanLoginWithEmail tests
- [x] CanLoginWithDevice tests (mobile-only allowed, hybrid blocked)
- [x] AccountType tests (string representation)

**RefreshToken Model (internal/models/refresh_token.go)** - ✅ COMPLETE
- [x] IsExpired tests (expired and valid tokens)
- [x] ShouldRotate tests (always true per requirements)
- [x] TimeSinceLastUsed tests (needed for reuse detection)

#### 7.1.3 Database Package Tests
**User Repository (internal/database/user_repository.go)** - ✅ COMPLETE
- [x] CreateUser tests
  - [x] Valid email user creation
  - [x] Duplicate email handling (ErrUserAlreadyExists)
  - [x] Empty email/password validation
  - [x] UUID generation
  - [x] Timestamp setting
- [x] CreateMobileUser tests
  - [x] Valid mobile user creation
  - [x] Duplicate device ID handling
  - [x] Empty device ID validation
- [x] GetUserByID tests
  - [x] Existing user retrieval
  - [x] Non-existent user (ErrUserNotFound)
- [x] GetUserByEmail tests
  - [x] Existing user retrieval
  - [x] Non-existent user
  - [x] Empty email validation
- [x] GetUserByExpoDeviceID tests
  - [x] Existing user retrieval
  - [x] Non-existent user
  - [x] Empty device ID validation
- [x] UpdateUser tests
  - [x] Valid email update
  - [x] Duplicate email handling
  - [x] Non-existent user
  - [x] Timestamp update verification
- [x] UpgradeAccount tests
  - [x] Valid mobile-to-hybrid upgrade
  - [x] Non-mobile account rejection
  - [x] Duplicate email handling
  - [x] Empty email/password validation
  - [x] Account type verification after upgrade
- [x] DeleteUser tests
  - [x] Successful deletion
  - [x] Non-existent user
  - [x] Row count verification
- [x] UpdateLastLoginAt tests
  - [x] Timestamp update
  - [x] Non-existent user
  - [x] Both updated_at and last_login_at set

**RefreshToken Repository (internal/database/refresh_token_repository.go)** - ✅ COMPLETE
- [x] CreateRefreshToken tests
  - [x] Valid token creation with all fields
  - [x] Token family handling (nil and provided)
  - [x] Device info and IP address (nil and provided)
  - [x] Unique constraint violation (duplicate hash)
  - [x] Foreign key violation (invalid user ID)
  - [x] Empty token hash validation
  - [x] Nil user ID validation
- [x] GetRefreshTokenByHash tests
  - [x] Existing token retrieval
  - [x] Non-existent token (ErrTokenNotFound)
  - [x] Empty hash validation
  - [x] All fields populated correctly
- [x] UpdateLastUsedAt tests
  - [x] Timestamp update
  - [x] Non-existent token
  - [x] Timestamp precision verification
  - [x] Multiple updates verification
- [x] RevokeToken tests
  - [x] Successful deletion
  - [x] Non-existent token
  - [x] Row count verification
  - [x] Double revoke handling
- [x] RevokeAllUserTokens tests
  - [x] All user tokens deleted
  - [x] Multiple tokens for user
  - [x] User with no tokens (no error)
  - [x] Row count verification
- [x] RevokeTokenFamily tests
  - [x] All family tokens deleted
  - [x] Multiple tokens in family
  - [x] Non-existent family (no error)
  - [x] Nil UUID validation
  - [x] Row count verification
  - [x] Mixed expiration states in family
- [x] CleanupExpiredTokens tests
  - [x] Expired tokens removed
  - [x] Valid tokens preserved
  - [x] Multiple expired tokens
  - [x] Empty table handling
  - [x] Row count verification
  - [x] Multiple cleanup operations
- [x] Edge case tests
  - [x] Far future expiration dates
  - [x] Past expiration dates
  - [x] Long device info strings
  - [x] IPv6 addresses
  - [x] Concurrent token operations
  - [x] Token expiration detection via model methods

#### 7.1.4 Services Package Tests
**Auth Service (internal/services/auth_service.go)** - ✅ COMPLETE
- [x] NewAuthService constructor tests
  - [x] Default password settings (minLength=8)
  - [x] Custom password settings
- [x] Register tests
  - [x] Valid email/password registration
  - [x] Empty email/password validation
  - [x] Weak password rejection
  - [x] Duplicate email handling (ErrAccountExists)
  - [x] Password hashing verification
  - [x] Token generation (access + refresh)
  - [x] Response structure validation
- [x] RegisterMobile tests
  - [x] Valid device ID registration
  - [x] Empty device ID validation
  - [x] Duplicate device ID handling
  - [x] Token generation with device info
  - [x] IP address tracking
- [x] Login tests
  - [x] Valid email/password login
  - [x] Empty email/password validation
  - [x] Non-existent user (ErrInvalidCredentials)
  - [x] Incorrect password (ErrInvalidCredentials)
  - [x] Mobile-only user cannot login with email
  - [x] Password comparison verification
  - [x] LastLoginAt update
  - [x] Token generation
- [x] LoginMobile tests
  - [x] Valid device ID login
  - [x] Empty device ID validation
  - [x] Non-existent device (ErrInvalidCredentials)
  - [x] Hybrid account rejection (ErrHybridAccountDeviceLogin)
  - [x] Mobile-only account success
  - [x] LastLoginAt update
  - [x] Token generation with device info
- [x] RefreshAccessToken tests
  - [x] Valid token refresh
  - [x] Empty refresh token validation
  - [x] Invalid refresh token
  - [x] Expired refresh token
  - [x] Token reuse detection
  - [x] Token rotation verification
  - [x] New access token generation
  - [x] Device info and IP tracking
- [x] Logout tests
  - [x] Valid single token revocation
  - [x] Empty token handling
  - [x] Already revoked token (no error)
  - [x] Token not found (no error)
- [x] LogoutAll tests
  - [x] All user tokens revoked
  - [x] User with multiple tokens
  - [x] User with no tokens
- [x] generateAuthResponse helper tests
  - [x] Access token generation
  - [x] Refresh token generation
  - [x] Token family creation
  - [x] Device info propagation
  - [x] ExpiresIn calculation
  - [x] User data in response

**User Service (internal/services/user_service.go)** - ✅ COMPLETE
- [x] NewUserService constructor tests
  - [x] Default password settings
  - [x] Custom password settings
- [x] GetUser tests
  - [x] Authorized access (requesting user = target user)
  - [x] Unauthorized access (different user - ErrUnauthorized)
  - [x] Non-existent user (ErrUserNotFound)
  - [x] User data returned correctly
- [x] UpdateUser tests
  - [x] Authorized update
  - [x] Unauthorized update (ErrUnauthorized)
  - [x] Nil email validation
  - [x] Duplicate email handling (ErrAccountExists)
  - [x] Non-existent user
  - [x] Updated user returned
- [x] UpgradeAccount tests
  - [x] Authorized upgrade
  - [x] Unauthorized upgrade (ErrUnauthorized)
  - [x] Empty email/password validation
  - [x] Weak password rejection
  - [x] Non-mobile account rejection (ErrNotMobileAccount)
  - [x] Duplicate email handling
  - [x] Password hashing verification
  - [x] Account type verification after upgrade
  - [x] Non-existent user
- [x] DeleteUser tests
  - [x] Authorized deletion
  - [x] Unauthorized deletion (ErrUnauthorized)
  - [x] Refresh tokens revoked
  - [x] Non-existent user
  - [x] Cleanup verification

#### 7.1.5 Middleware Package Tests
**JWT Auth Middleware (internal/middleware/auth.go)** - ✅ COMPLETE
- [x] JWTAuth middleware tests
  - [x] Valid token allows request
  - [x] Missing Authorization header (401)
  - [x] Invalid header format (401)
  - [x] Empty token (401)
  - [x] Expired token (401)
  - [x] Invalid signature (401)
  - [x] Invalid issuer (401)
  - [x] Invalid audience (401)
  - [x] User ID stored in context
  - [x] Error response formats
- [x] GetUserIDFromContext tests
  - [x] Valid user ID retrieval
  - [x] Missing user ID in context
  - [x] Invalid type in context
- [x] RequireAuth alias tests
- [x] OptionalAuth middleware tests
  - [x] Valid token sets context
  - [x] Missing token continues
  - [x] Invalid token continues
  - [x] User ID in context when valid
- [x] IsAuthenticated tests
  - [x] Returns true when authenticated
  - [x] Returns false when not authenticated
- [x] MustGetUserID tests
  - [x] Returns user ID when present
  - [x] Panics when missing

**Authorization Middleware (internal/middleware/authorization.go)** - ✅ COMPLETE
- [x] RequireSameUser middleware tests
  - [x] Authorized access (same user)
  - [x] Unauthorized access (different user - 403)
  - [x] Missing authentication (401)
  - [x] Missing ID parameter (400)
  - [x] Invalid UUID format (400)
  - [x] Request continues when authorized
- [x] RequireSameUserWithCustomParam tests
  - [x] Custom parameter name handling
  - [x] All authorization checks
- [x] RequireOwnership tests
  - [x] Custom owner check function
  - [x] Authorized access
  - [x] Unauthorized access
  - [x] Owner check errors
- [x] CheckUserIDParam tests
  - [x] Valid UUID continues
  - [x] Missing parameter (400)
  - [x] Invalid UUID format (400)

**Rate Limiting Middleware (internal/middleware/rate_limit.go)** - ✅ COMPLETE
- [x] NewRateLimiter tests
  - [x] Constructor initialization
  - [x] Cleanup loop starts
- [x] RateLimiter.allow tests
  - [x] First request allowed
  - [x] Within limit allowed
  - [x] Exceed limit blocked
  - [x] Window refill after time passes
  - [x] Multiple keys independent
  - [x] Concurrent access safety
- [x] RateLimiter cleanup tests
  - [x] Old entries removed
  - [x] Active entries preserved
- [x] RateLimit middleware tests
  - [x] Requests within limit allowed
  - [x] Exceeded limit returns 429
  - [x] Rate limit headers set
  - [x] IP-based limiting
  - [x] Window reset after time
- [x] RateLimitByUser tests
  - [x] Authenticated user limiting
  - [x] Fallback to IP when unauthenticated
  - [x] User ID as key
  - [x] Rate limit headers
- [x] RateLimitWithKey tests
  - [x] Custom key function
  - [x] All rate limit behaviors
- [x] RateLimitAuth tests
  - [x] Same behavior as RateLimit

**Other Middleware Tests** - ❌ NOT STARTED
- [ ] CORS middleware tests (internal/middleware/cors.go)
  - [ ] CORS headers set correctly
  - [ ] Preflight requests handled
  - [ ] Allowed origins/methods/headers
- [ ] Recovery middleware tests (internal/middleware/recovery.go)
  - [ ] Panic recovery
  - [ ] 500 response on panic
  - [ ] Error logging
- [ ] Security middleware tests (internal/middleware/security.go)
  - [ ] HTTPS enforcement
  - [ ] Security headers set
- [ ] Logging middleware tests (internal/middleware/logging.go)
  - [ ] Request logging
  - [ ] Response logging
  - [ ] Duration tracking
  - [ ] Request ID tracking

#### 7.1.6 Handlers Package Tests
**Auth Handlers (internal/handlers/auth_handler.go)** - ❌ NOT STARTED
- [ ] POST /auth/register handler tests
  - [ ] Valid registration
  - [ ] Invalid JSON body (400)
  - [ ] Missing email/password (400)
  - [ ] Weak password (400)
  - [ ] Duplicate email (409)
  - [ ] Response structure validation
- [ ] POST /auth/register/mobile handler tests
  - [ ] Valid mobile registration
  - [ ] Invalid JSON body
  - [ ] Missing device ID
  - [ ] Duplicate device ID (409)
  - [ ] Device info and IP capture
- [ ] POST /auth/login handler tests
  - [ ] Valid email/password login
  - [ ] Invalid JSON body
  - [ ] Missing credentials (400)
  - [ ] Invalid credentials (401)
  - [ ] Response structure validation
- [ ] POST /auth/login/mobile handler tests
  - [ ] Valid mobile login
  - [ ] Invalid JSON body
  - [ ] Missing device ID
  - [ ] Invalid device ID (401)
  - [ ] Hybrid account blocked (403)
- [ ] POST /auth/refresh handler tests
  - [ ] Valid token refresh
  - [ ] Invalid JSON body
  - [ ] Missing refresh token (400)
  - [ ] Invalid refresh token (401)
  - [ ] Expired token (401)
  - [ ] Token reuse (401/403)
- [ ] POST /auth/logout handler tests
  - [ ] Valid logout
  - [ ] Missing refresh token
  - [ ] Already revoked token (no error)
- [ ] POST /auth/logout-all handler tests
  - [ ] Valid logout all
  - [ ] Authentication required
  - [ ] User ID from JWT token

**User Handlers (internal/handlers/user_handler.go)** - ❌ NOT STARTED
- [ ] GET /users/:id handler tests
  - [ ] Valid user retrieval
  - [ ] Authentication required (401)
  - [ ] Authorization required (403)
  - [ ] Non-existent user (404)
  - [ ] Response structure validation
- [ ] PATCH /users/:id handler tests
  - [ ] Valid user update
  - [ ] Invalid JSON body
  - [ ] Authentication required
  - [ ] Authorization required (403)
  - [ ] Duplicate email (409)
  - [ ] Non-existent user (404)
- [ ] POST /users/:id/upgrade handler tests
  - [ ] Valid account upgrade
  - [ ] Invalid JSON body
  - [ ] Authentication required
  - [ ] Authorization required
  - [ ] Weak password (400)
  - [ ] Non-mobile account (400)
  - [ ] Duplicate email (409)
  - [ ] Account type verification
- [ ] DELETE /users/:id handler tests
  - [ ] Valid user deletion
  - [ ] Authentication required
  - [ ] Authorization required
  - [ ] Non-existent user (404)

**Health Handlers (internal/handlers/health.go)** - ❌ NOT STARTED
- [ ] GET /health handler tests
  - [ ] Returns 200 OK
  - [ ] Response structure
  - [ ] Uptime calculation
- [ ] GET /ready handler tests
  - [ ] Returns 200 when all dependencies healthy
  - [ ] Returns 503 when database unavailable
  - [ ] Returns 503 when Vault unavailable
  - [ ] Dependency status in response

#### 7.1.7 Coverage Goals
- [ ] Achieve >80% overall code coverage
- [ ] 100% coverage for critical security code (auth, password, JWT)
- [ ] 100% coverage for business logic (services)
- [ ] >90% coverage for data layer (repositories)
- [ ] >80% coverage for HTTP layer (handlers, middleware)
- [ ] Generate coverage reports (HTML and text)
- [ ] Set up coverage thresholds in CI

### 7.2 Integration Tests
- [ ] Set up test database
- [ ] Write integration tests for registration flow
- [ ] Write integration tests for login flow
- [ ] Write integration tests for token refresh flow
- [ ] Write integration tests for logout flow
- [ ] Write integration tests for account upgrade flow
- [ ] Write integration tests for authorization checks
- [ ] Test token reuse detection
- [ ] Test rate limiting

### 7.3 End-to-End Tests
- [ ] Test complete registration -> login -> API access flow
- [ ] Test mobile registration -> upgrade -> email login flow
- [ ] Test refresh token rotation
- [ ] Test multi-device scenarios
- [ ] Test security scenarios (unauthorized access, expired tokens)

## Phase 8: Security Hardening

### 8.1 Security Review
- [ ] Review all endpoints for SQL injection vulnerabilities
- [ ] Review for XSS vulnerabilities
- [ ] Review for CSRF vulnerabilities (if applicable)
- [ ] Verify rate limiting effectiveness
- [ ] Test token expiration handling
- [ ] Test refresh token reuse detection
- [ ] Verify HTTPS enforcement
- [ ] Review logging for sensitive data exposure

### 8.2 Penetration Testing
- [ ] Test authentication bypass attempts
- [ ] Test authorization bypass attempts
- [ ] Test brute force protection
- [ ] Test token manipulation attempts
- [ ] Test concurrent request handling
- [ ] Verify database constraint enforcement

## Phase 9: Observability & Monitoring

### 9.1 Logging
- [ ] Implement structured logging
- [ ] Log all authentication attempts (success/failure)
- [ ] Log token refresh events
- [ ] Log token revocation events
- [ ] Log account upgrades
- [ ] Ensure no sensitive data in logs (passwords, tokens)
- [ ] Add request ID tracking

### 9.2 Metrics
- [ ] Add metrics for authentication requests
- [ ] Add metrics for token operations
- [ ] Add metrics for database operations
- [ ] Add metrics for Vault operations
- [ ] Track error rates by endpoint
- [ ] Track response times

### 9.3 Alerts
- [ ] Configure alerts for multiple failed login attempts
- [ ] Configure alerts for token reuse detection
- [ ] Configure alerts for service health issues
- [ ] Configure alerts for database connection failures
- [ ] Configure alerts for Vault connection failures

## Phase 10: Documentation

### 10.1 API Documentation
- [ ] Create OpenAPI/Swagger specification
- [ ] Document all endpoints with request/response examples
- [ ] Document error codes and responses
- [ ] Document authentication flow
- [ ] Document token refresh flow
- [ ] Document account upgrade flow

### 10.2 Developer Documentation
- [ ] Write setup instructions (local development)
- [ ] Document configuration options
- [ ] Document environment variables
- [ ] Document Vault setup requirements
- [ ] Document database migration process
- [ ] Create architecture diagrams
- [ ] Document security considerations

### 10.3 Operations Documentation
- [ ] Write deployment guide
- [ ] Document monitoring and alerting
- [ ] Document backup and recovery procedures
- [ ] Document key rotation procedures
- [ ] Document incident response procedures
- [ ] Create runbook for common issues

## Phase 11: Deployment Preparation

### 11.1 Configuration Management
- [ ] Create production configuration template
- [ ] Create staging configuration template
- [ ] Document all required environment variables
- [ ] Set up Vault policies for service authentication
- [ ] Create database migration deployment process

### 11.2 Container & Orchestration
- [ ] Create Dockerfile for user service
- [ ] Optimize Docker image size
- [ ] Create Kubernetes manifests (if applicable)
- [ ] Configure health check probes
- [ ] Configure resource limits
- [ ] Set up horizontal pod autoscaling (if applicable)

### 11.3 CI/CD Pipeline
- [ ] Set up automated testing in CI
- [ ] Add linting and static analysis
- [ ] Add security scanning
- [ ] Configure automated builds
- [ ] Set up deployment automation
- [ ] Add database migration automation

## Phase 12: Performance & Optimization

### 12.1 Performance Testing
- [ ] Load test authentication endpoints
- [ ] Load test token refresh endpoint
- [ ] Verify <100ms refresh operation requirement
- [ ] Test concurrent validation across multiple instances
- [ ] Profile database query performance
- [ ] Optimize slow queries

### 12.2 Caching Strategy
- [ ] Implement public key caching for validation
- [ ] Configure cache TTL and invalidation
- [ ] Test graceful degradation with database unavailable
- [ ] Implement connection pooling optimization

## Phase 13: Launch Checklist

- [ ] All tests passing
- [ ] Security review completed
- [ ] Performance requirements met
- [ ] Documentation complete
- [ ] Monitoring and alerts configured
- [ ] Vault integration tested in production environment
- [ ] Database migrations tested
- [ ] Rollback plan documented
- [ ] Team training completed
- [ ] Support procedures documented

## Future Enhancements (Post-MVP)

- [ ] OAuth2/OpenID Connect support
- [ ] Multi-factor authentication (MFA)
- [ ] Email verification workflow
- [ ] Password reset functionality
- [ ] Account lockout after failed attempts
- [ ] Remember device functionality
- [ ] Device management (list/revoke devices)
- [ ] Audit log for security events

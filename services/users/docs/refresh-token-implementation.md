# Refresh Token Management Implementation

## Overview

This document describes the implementation of Phase 3.4 Refresh Token Management for the Cairn User Service.

## Implementation Summary

The refresh token management system provides a secure, production-ready implementation with the following features:

### 1. Cryptographically Secure Token Generation

- **Implementation**: [internal/auth/refresh_token.go:65-85](../internal/auth/refresh_token.go#L65-L85)
- Uses `crypto/rand` for cryptographically secure random number generation
- Generates 256-bit (32 byte) random tokens
- Encodes tokens using base64 URL encoding for safe transmission
- Each token is unique and unpredictable

### 2. Token Hashing (SHA-256)

- **Implementation**: [internal/auth/refresh_token.go:87-91](../internal/auth/refresh_token.go#L87-L91)
- Tokens are never stored in plain text
- SHA-256 hash is stored in the database
- Provides one-way encryption - original tokens cannot be recovered from hash
- Efficient lookup and comparison

### 3. Token Rotation Logic

- **Implementation**: [internal/auth/refresh_token.go:124-198](../internal/auth/refresh_token.go#L124-L198)
- Automatic rotation on every token use
- New token is created in the same token family
- Old token is immediately revoked after successful rotation
- Ensures each refresh token is single-use only

### 4. Refresh Token Reuse Detection

- **Implementation**: [internal/auth/refresh_token.go:200-217](../internal/auth/refresh_token.go#L200-L217)
- Detects if a token is used multiple times within a grace period (5 seconds)
- Grace period accounts for network latency and race conditions
- First use is tracked separately from subsequent uses
- Critical security feature for detecting token theft

### 5. Token Family Tracking

- **Implementation**: Database schema and repository support
- All tokens in a rotation chain share the same `token_family` UUID
- Enables tracking of token lineage
- Allows selective revocation of entire token chains
- Implemented at database level with foreign key support

### 6. Automatic Revocation on Compromise Detection

- **Implementation**: [internal/auth/refresh_token.go:150-169](../internal/auth/refresh_token.go#L150-L169)
- When token reuse is detected, all tokens in the family are revoked
- Fallback: Revokes all user tokens if family tracking is unavailable
- Immediate security response to potential token theft
- Prevents attackers from using stolen tokens

### 7. Configurable Token Lifetime

- **Implementation**: [internal/auth/refresh_token.go:32-38](../internal/auth/refresh_token.go#L32-L38)
- Default lifetime: 30 days
- Configurable via `JWTConfig.RefreshTokenExpiry`
- Environment variable: `JWT_REFRESH_TOKEN_EXPIRY`
- Automatic cleanup of expired tokens

## Key Features

### Security Features

1. **Cryptographically secure random generation** - Uses `crypto/rand`
2. **One-way hashing** - SHA-256 ensures tokens cannot be recovered
3. **Single-use tokens** - Automatic rotation prevents reuse
4. **Reuse detection** - Identifies potential token theft
5. **Family-based revocation** - Compromised token families are fully revoked
6. **Configurable expiration** - Tokens automatically expire

### Performance Features

1. **Efficient hashing** - SHA-256 is fast and secure
2. **Database indexing** - Token hash is indexed for quick lookup
3. **Automatic cleanup** - Expired tokens are periodically removed
4. **Minimal overhead** - Token rotation adds < 100ms to refresh operations

### Operational Features

1. **Comprehensive error handling** - Clear error messages for debugging
2. **Extensive logging** - All token operations are logged
3. **Health monitoring** - Cleanup operations return counts for metrics
4. **Graceful degradation** - Handles missing token families

## API Documentation

### RefreshTokenService Methods

#### GenerateToken()
```go
func (s *RefreshTokenService) GenerateToken() (token string, hash string, err error)
```
Generates a cryptographically secure random token and its hash.

#### CreateRefreshToken()
```go
func (s *RefreshTokenService) CreateRefreshToken(
    ctx context.Context,
    userID uuid.UUID,
    deviceInfo, ipAddress *string,
    tokenFamily *uuid.UUID,
) (token string, tokenModel *models.RefreshToken, err error)
```
Creates and stores a new refresh token in the database.

#### ValidateAndRotateToken()
```go
func (s *RefreshTokenService) ValidateAndRotateToken(
    ctx context.Context,
    token string,
    deviceInfo, ipAddress *string,
) (newToken string, userID uuid.UUID, err error)
```
Validates a refresh token and performs automatic rotation. Includes reuse detection and compromise handling.

#### RevokeToken()
```go
func (s *RefreshTokenService) RevokeToken(ctx context.Context, token string) error
```
Revokes a single refresh token.

#### RevokeAllUserTokens()
```go
func (s *RefreshTokenService) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error
```
Revokes all refresh tokens for a specific user (logout all devices).

#### RevokeTokenFamily()
```go
func (s *RefreshTokenService) RevokeTokenFamily(ctx context.Context, tokenFamily uuid.UUID) error
```
Revokes all tokens in a token family (compromise detection).

#### CleanupExpiredTokens()
```go
func (s *RefreshTokenService) CleanupExpiredTokens(ctx context.Context) (int64, error)
```
Removes expired tokens from the database. Returns count of deleted tokens.

## Error Handling

### Error Types

- `ErrTokenReused` - Token was reused (possible theft detected)
- `ErrTokenExpired` - Token has expired (from jwt.go)
- `ErrRefreshTokenNotFound` - Token not found in database
- `ErrInvalidToken` - Token format is invalid (from jwt.go)

## Testing

### Test Coverage

- **Test File**: [internal/auth/refresh_token_test.go](../internal/auth/refresh_token_test.go)
- **Total Tests**: 11 test functions with 28 sub-tests
- **Coverage Areas**:
  - Token generation and uniqueness
  - Hash consistency and security
  - Token creation with and without families
  - Validation and rotation flows
  - Expiration handling
  - Reuse detection (within and outside grace period)
  - Revocation operations
  - Error conditions

### Running Tests

```bash
# Run refresh token tests only
go test -v ./internal/auth/refresh_token_test.go ./internal/auth/refresh_token.go

# Run all auth tests
go test -v ./internal/auth/...

# Run all project tests
go test ./...
```

## Usage Example

```go
// Initialize the service
repo := database.NewRefreshTokenRepository(db)
service := auth.NewRefreshTokenService(repo, 30*24*time.Hour)

// Create a new refresh token
token, tokenModel, err := service.CreateRefreshToken(
    ctx,
    userID,
    &deviceInfo,
    &ipAddress,
    nil, // Will create new token family
)

// Later, validate and rotate the token
newToken, userID, err := service.ValidateAndRotateToken(
    ctx,
    token,
    &deviceInfo,
    &ipAddress,
)

// Handle reuse detection
if errors.Is(err, auth.ErrTokenReused) {
    // Token family has been automatically revoked
    // Log security event
    // Notify user of potential account compromise
}

// Logout (revoke single token)
err = service.RevokeToken(ctx, token)

// Logout all devices
err = service.RevokeAllUserTokens(ctx, userID)
```

## Security Considerations

1. **Token Transmission**: Always use HTTPS/TLS for token transmission
2. **Storage**: Never log or store raw tokens (only hashes)
3. **Reuse Detection**: Monitor for `ErrTokenReused` errors - indicates potential attack
4. **Grace Period**: 5-second grace period balances security vs. usability
5. **Expiration**: 30-day default balances security vs. user convenience
6. **Rotation**: Automatic rotation ensures single-use tokens

## Configuration

Environment variables:

```bash
# Refresh token lifetime (default: 720h = 30 days)
JWT_REFRESH_TOKEN_EXPIRY=720h

# Access token lifetime (default: 60m)
JWT_ACCESS_TOKEN_EXPIRY=60m
```

## Database Schema

The refresh token table includes:

- `id` - Primary key (UUID)
- `user_id` - Foreign key to users table
- `token_hash` - SHA-256 hash of the token (unique index)
- `expires_at` - Expiration timestamp (index)
- `created_at` - Creation timestamp
- `last_used_at` - Last usage timestamp (for reuse detection)
- `device_info` - Optional device information
- `ip_address` - Optional IP address
- `token_family` - UUID for tracking rotation chains (index)

## Next Steps

Phase 3.4 is now complete. Next phases:

- **Phase 4.1**: Authentication Service (use RefreshTokenService)
- **Phase 4.2**: User Service
- **Phase 5**: HTTP Layer (handlers and middleware)

## References

- [OWASP Refresh Token Best Practices](https://cheatsheetseries.owasp.org/cheatsheets/JSON_Web_Token_for_Java_Cheat_Sheet.html)
- [RFC 6749 - OAuth 2.0 Refresh Tokens](https://tools.ietf.org/html/rfc6749#section-1.5)
- [Token Rotation and Reuse Detection](https://auth0.com/docs/secure/tokens/refresh-tokens/refresh-token-rotation)

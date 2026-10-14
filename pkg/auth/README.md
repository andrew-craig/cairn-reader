# Cairn Shared Authentication Package

This package provides JWT validation and HTTP middleware for protecting endpoints in Cairn microservices. It validates tokens issued by the Cairn User Service using RS256 public key cryptography.

## Features

- **JWT Validation**: Validate RS256 JWT tokens with issuer and audience verification
- **HTTP Middleware**: Ready-to-use middleware for `net/http` handlers
- **User Context**: Extract authenticated user ID from request context
- **Vault Integration**: Fetch JWT public keys from HashiCorp Vault
- **Key Rotation**: `KeyRefresher` polls for rotated keys and re-fetches on a signature failure
- **Internal API Key**: `InternalAuthMiddleware` for service-to-service endpoints (`X-Internal-API-Key`)
- **Token Inspection**: Debug utilities for examining token contents
- **Thread-Safe**: Safe for concurrent use across multiple goroutines

## Installation

```bash
go get github.com/andrew-craig/cairn-reader/pkg/auth
```

## Quick Start

### Basic Usage with `net/http`

```go
package main

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"

    "github.com/andrew-craig/cairn-reader/pkg/auth"
)

func main() {
    // 1. Fetch public key from Vault
    vaultClient, err := auth.NewVaultClient(&auth.VaultConfig{
        Address: "https://vault.example.com",
        Token:   "vault-token",
    })
    if err != nil {
        log.Fatal(err)
    }

    publicKey, err := vaultClient.GetPublicKey("secret/data/jwt/public-key")
    if err != nil {
        log.Fatal(err)
    }

    // 2. Create validator and middleware
    validator := auth.NewValidator(publicKey)
    middleware := auth.NewMiddleware(validator)

    // 3. Create your handlers
    protectedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Extract user ID from context
        userID := auth.MustGetUserID(r.Context())

        response := map[string]string{
            "message": "This is a protected endpoint",
            "user_id": userID.String(),
        }
        json.NewEncoder(w).Encode(response)
    })

    publicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Check if authenticated (optional auth). At this point the request is
        // either anonymous (no Authorization header was sent) or carries a
        // fully-validated token — OptionalAuth has already rejected invalid
        // tokens with 401 before the handler runs.
        if auth.IsAuthenticated(r.Context()) {
            userID, _ := auth.GetUserIDFromContext(r.Context())
            fmt.Fprintf(w, "Authenticated user: %s", userID)
        } else {
            fmt.Fprint(w, "Public access")
        }
    })

    // 4. Apply middleware
    mux := http.NewServeMux()
    mux.Handle("/api/protected", middleware.RequireAuth(protectedHandler))
    mux.Handle("/api/public", middleware.OptionalAuth(publicHandler))
    mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("OK"))
    })

    log.Println("Server starting on :8080")
    log.Fatal(http.ListenAndServe(":8080", mux))
}
```

### Using with `http.ServeMux` (Standard Library Router)

```go
mux := http.NewServeMux()

// Protected routes - require valid JWT
mux.Handle("/api/v1/content/", middleware.RequireAuth(contentHandler))
mux.Handle("/api/v1/articles/read", middleware.RequireAuth(markReadHandler))

// Optional auth - useful for endpoints that work with or without auth.
// Anonymous when no Authorization header is sent; 401 when a header is
// present but the token is invalid.
mux.Handle("/api/v1/articles/", middleware.OptionalAuth(articlesHandler))

// Public routes - no auth required
mux.HandleFunc("/health", healthHandler)

http.ListenAndServe(":8080", mux)
```

### Custom Configuration

```go
// NewValidator already defaults to issuer "cairn-user-service" and audience
// "cairn-api"; use NewValidatorWithConfig to override them or set a key ID
validator := auth.NewValidatorWithConfig(auth.ValidatorConfig{
    PublicKey: publicKey,
    Issuer:    "cairn-user-service",
    Audience:  "cairn-api",
})
```

### Chaining Multiple Middleware

```go
// Create a logging middleware
func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        log.Printf("%s %s", r.Method, r.URL.Path)
        next.ServeHTTP(w, r)
    })
}

// Chain middleware together
handler := auth.Chain(
    loggingMiddleware,
    middleware.RequireAuth,
)(yourHandler)

mux.Handle("/api/endpoint", handler)
```

## API Reference

### Validator

```go
// Create a new validator
validator := auth.NewValidator(publicKey *rsa.PublicKey)

// Validate a token string
claims, err := validator.ValidateToken(tokenString)

// Update public key (for key rotation; KeyRefresher does this for you)
validator.UpdatePublicKey(newPublicKey)

// Get token info without full validation (debugging only)
info, err := validator.GetTokenInfo(tokenString)
```

### Middleware

```go
middleware := auth.NewMiddleware(validator)

// Require authentication - returns 401 if invalid/missing token
protectedHandler := middleware.RequireAuth(handler)

// Optional authentication - anonymous when no Authorization header is sent,
// but returns 401 if a header is present and the token is malformed,
// expired, or otherwise invalid (fail-closed when credentials are presented).
optionalHandler := middleware.OptionalAuth(handler)
```

#### OptionalAuth security model

`OptionalAuth` is fail-closed when credentials are presented:

| Authorization header           | Result                                                  |
|--------------------------------|---------------------------------------------------------|
| absent                         | request continues anonymously                           |
| present, malformed             | `401 Unauthorized` (Warn-logged with method/path/IP)    |
| present, token invalid/expired | `401 Unauthorized` (Warn-logged with method/path/IP)    |
| present, token valid           | request continues with `user_id` in context             |

Clients that want anonymous access **must omit the `Authorization` header**.
Sending a stale or tampered token does not silently fall back to anonymous —
it returns `401` so the client can refresh or drop credentials and so
abuse attempts surface in logs. Token contents are never logged.

OpenAPI specs for endpoints behind `OptionalAuth` should declare
`security: [{}, {bearerAuth: []}]` (both anonymous and bearer accepted) and
document a `401` response for invalid credentials.

### Context Utilities

```go
// Extract user ID from context (returns zero UUID if not found)
userID, ok := auth.GetUserIDFromContext(ctx)

// Get user ID or error - RECOMMENDED approach for protected handlers
// This prevents service crashes on programming errors
userID, err := auth.GetUserIDOrError(ctx)
if err != nil {
    // Handle authentication context error
    return err
}

// Must get user ID - panics if not found (DEPRECATED)
// Use GetUserIDOrError instead to avoid service crashes
userID := auth.MustGetUserID(ctx)

// Check if request is authenticated
if auth.IsAuthenticated(ctx) {
    // User is logged in
}
```

**Best Practice:** Always use `GetUserIDOrError()` in handlers instead of `MustGetUserID()` to prevent service crashes if middleware is misconfigured.

### Vault Integration

```go
// Connect to Vault
vaultClient, err := auth.NewVaultClient(&auth.VaultConfig{
    Address:   "https://vault.example.com",
    Token:     "vault-token",
    // OR use AppRole
    RoleID:    "role-id",
    SecretID:  "secret-id",
    AuthPath:  "approle",
    Namespace: "namespace", // optional
})

// Fetch public key
publicKey, err := vaultClient.GetPublicKey("secret/data/jwt/public-key")

// Health check
err := vaultClient.Health()

// Renew token (if using token auth)
err := vaultClient.RenewToken()
```

## Error Handling

The package provides specific error types for different validation failures:

```go
claims, err := validator.ValidateToken(token)
if err != nil {
    switch {
    case errors.Is(err, auth.ErrTokenExpired):
        // Handle expired token
    case errors.Is(err, auth.ErrInvalidSignature):
        // Handle invalid signature
    case errors.Is(err, auth.ErrInvalidIssuer):
        // Handle wrong issuer
    case errors.Is(err, auth.ErrInvalidAudience):
        // Handle wrong audience
    case errors.Is(err, auth.ErrMissingToken):
        // Handle missing token
    default:
        // Handle other errors
    }
}
```

## Token Format

The package expects JWT tokens in the following format:

**Header:**
```json
{
  "alg": "RS256",
  "typ": "JWT"
}
```

**Claims:**
```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "iss": "cairn-user-service",
  "aud": "cairn-api",
  "sub": "550e8400-e29b-41d4-a716-446655440000",
  "iat": 1640000000,
  "exp": 1640003600,
  "nbf": 1640000000
}
```

**Authorization Header:**
```
Authorization: Bearer <token>
```

## Complete Example: Content Service Integration

Here's how the Content Service (`services/read/content`) wires this package into a chi router. Public-key
refresh and the service-to-service internal API key are included:

```go
// Based on services/read/content/cmd/content/main.go and internal/api/router.go
package main

import (
    "context"
    "net/http"

    "github.com/andrew-craig/cairn-reader/pkg/auth"
    "github.com/go-chi/chi/v5"
)

func newRouter(ctx context.Context, cfg *Config) (http.Handler, error) {
    // Setup Vault client (AppRole in production, token in development)
    vaultClient, err := auth.NewVaultClient(&auth.VaultConfig{
        Address:  cfg.Vault.Address,
        Token:    cfg.Vault.Token,
        RoleID:   cfg.Vault.RoleID,
        SecretID: cfg.Vault.SecretID,
        AuthPath: cfg.Vault.AuthPath,
    })
    if err != nil {
        return nil, err
    }

    // Fetch JWT public key
    publicKey, err := vaultClient.GetPublicKey(cfg.Vault.PublicKeyPath)
    if err != nil {
        return nil, err
    }

    // Create validator and middleware
    validator := auth.NewValidator(publicKey)
    authMiddleware := auth.NewMiddleware(validator)

    // Pick up key rotations performed by the User Service
    keyRefresher := auth.NewKeyRefresher(vaultClient, cfg.Vault.PublicKeyPath, validator)
    keyRefresher.Start(ctx, cfg.Vault.PublicKeyRefreshInterval)

    // Service-to-service endpoints use a shared API key instead of a JWT
    internalAuth := auth.NewInternalAuthMiddleware(cfg.InternalAPIKey)

    r := chi.NewRouter()

    // Public endpoints
    r.Get("/health/live", handleLive)

    // Protected endpoints - require a valid JWT
    r.Route("/api/v1/content/user/{user_id}", func(r chi.Router) {
        r.Use(authMiddleware.RequireAuth)
        r.Get("/", handleListUserContents)
    })

    // Internal endpoints - require the X-Internal-API-Key header
    r.With(internalAuth.RequireInternalAPIKey).Post("/api/v1/internal/content/user/bulk", handleBulkAdd)

    return r, nil
}

func handleListUserContents(w http.ResponseWriter, r *http.Request) {
    // Extract authenticated user ID from context
    // RECOMMENDED: Use GetUserIDOrError to prevent service crashes
    authenticatedUserID, err := auth.GetUserIDOrError(r.Context())
    if err != nil {
        http.Error(w, "Authentication context error", http.StatusInternalServerError)
        return
    }

    // The user ID in the URL must match the token's user, so a user can only read their own data
    if chi.URLParam(r, "user_id") != authenticatedUserID.String() {
        http.Error(w, "Forbidden", http.StatusForbidden)
        return
    }

    // ... rest of handler
}
```

## Best Practices

1. **Store Public Key Securely**: Fetch from Vault, not from environment variables
2. **Use `GetUserIDOrError()` Instead of `MustGetUserID()`**: Return errors instead of panicking to prevent service crashes on middleware misconfiguration
3. **Handle Key Rotation**: Use a `KeyRefresher` (see below) so the validator picks up rotated keys
4. **Validate Issuer/Audience**: Always set these to prevent token misuse
5. **Use HTTPS in Production**: JWT tokens should only be transmitted over TLS
6. **Set Short Token Expiry**: The User Service issues access tokens valid for 60 minutes by default (`JWT_ACCESS_TOKEN_EXPIRY`)
7. **Log Authentication Failures**: For security monitoring and debugging

## Key Rotation Support

The User Service rotates its signing key on a timer (`JWT_KEY_ROTATION_INTERVAL`, default 24h), so a validator built once at startup would reject new tokens. `KeyRefresher` keeps it current in two ways: `Start` polls the key source on an interval, and a signature failure triggers an on-demand re-fetch (debounced to once per 10s) before validation retries once.

```go
// vaultClient satisfies PublicKeyFetcher (GetPublicKey(path))
keyRefresher := auth.NewKeyRefresher(vaultClient, publicKeyPath, validator)
keyRefresher.Start(ctx, 5*time.Minute) // a non-positive interval disables polling only
```

## Testing

The package is designed to be easily testable. You can create test validators with your own RSA key pairs:

```go
// Generate test keys
privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
publicKey := &privateKey.PublicKey

// Create test validator
validator := auth.NewValidator(publicKey)

// Use in tests
claims, err := validator.ValidateToken(testToken)
```

## License

Internal Cairn package - not for external distribution.

## Support

For issues or questions, contact the Cairn Platform Team.

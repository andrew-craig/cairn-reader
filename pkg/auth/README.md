# Cairn Shared Authentication Package

This package provides JWT validation and HTTP middleware for protecting endpoints in Cairn microservices. It validates tokens issued by the Cairn User Service using RS256 public key cryptography.

## Features

- **JWT Validation**: Validate RS256 JWT tokens with issuer and audience verification
- **HTTP Middleware**: Ready-to-use middleware for `net/http` handlers
- **User Context**: Extract authenticated user ID from request context
- **Vault Integration**: Fetch JWT public keys from HashiCorp Vault
- **Token Inspection**: Debug utilities for examining token contents
- **Thread-Safe**: Safe for concurrent use across multiple goroutines

## Installation

```bash
go get github.com/cairn-app/cairn-reader/services/users/pkg/auth
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

    "github.com/cairn-app/cairn-reader/services/users/pkg/auth"
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

    publicKey, err := vaultClient.GetPublicKey("secret/jwt/public-key")
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
        // Check if authenticated (optional auth)
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
mux.Handle("/api/v1/recommendations/", middleware.RequireAuth(recommendationsHandler))
mux.Handle("/api/v1/articles/read", middleware.RequireAuth(markReadHandler))

// Optional auth - useful for endpoints that work with or without auth
mux.Handle("/api/v1/articles/", middleware.OptionalAuth(articlesHandler))

// Public routes - no auth required
mux.HandleFunc("/health", healthHandler)

http.ListenAndServe(":8080", mux)
```

### Custom Configuration

```go
// Custom issuer and audience validation
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

// Update public key (for key rotation)
validator.UpdatePublicKey(newPublicKey)

// Get token info without full validation (debugging only)
info, err := validator.GetTokenInfo(tokenString)
```

### Middleware

```go
middleware := auth.NewMiddleware(validator)

// Require authentication - returns 401 if invalid/missing token
protectedHandler := middleware.RequireAuth(handler)

// Optional authentication - continues even if no token present
optionalHandler := middleware.OptionalAuth(handler)
```

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
publicKey, err := vaultClient.GetPublicKey("secret/jwt/public-key")

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

## Complete Example: Explore Service Integration

Here's how to integrate this package into the Explore/Recommender service:

```go
// File: services/explore/recommender/internal/api/server.go
package api

import (
    "log"
    "net/http"

    "github.com/cairn-app/cairn-reader/services/users/pkg/auth"
)

type Server struct {
    authMiddleware *auth.Middleware
    // ... other dependencies
}

func NewServer(vaultAddr, vaultToken string) (*Server, error) {
    // Setup Vault client
    vaultClient, err := auth.NewVaultClient(&auth.VaultConfig{
        Address: vaultAddr,
        Token:   vaultToken,
    })
    if err != nil {
        return nil, err
    }

    // Fetch JWT public key
    publicKey, err := vaultClient.GetPublicKey("secret/jwt/public-key")
    if err != nil {
        return nil, err
    }

    // Create validator and middleware
    validator := auth.NewValidator(publicKey)
    authMiddleware := auth.NewMiddleware(validator)

    return &Server{
        authMiddleware: authMiddleware,
    }, nil
}

func (s *Server) Routes() http.Handler {
    mux := http.NewServeMux()

    // Protected endpoints - require authentication
    mux.Handle("/api/v1/recommendations/", s.authMiddleware.RequireAuth(
        http.HandlerFunc(s.handleGetRecommendations),
    ))

    mux.Handle("/api/v1/articles/read", s.authMiddleware.RequireAuth(
        http.HandlerFunc(s.handleMarkArticleRead),
    ))

    mux.Handle("/api/v1/articles/vote", s.authMiddleware.RequireAuth(
        http.HandlerFunc(s.handleVote),
    ))

    // Public endpoints
    mux.HandleFunc("/health", s.handleHealth)

    return mux
}

func (s *Server) handleGetRecommendations(w http.ResponseWriter, r *http.Request) {
    // Extract authenticated user ID from context
    // RECOMMENDED: Use GetUserIDOrError to prevent service crashes
    userID, err := auth.GetUserIDOrError(r.Context())
    if err != nil {
        http.Error(w, "Authentication context error", http.StatusInternalServerError)
        return
    }

    // Use userID for recommendations logic
    log.Printf("Getting recommendations for user: %s", userID)

    // ... rest of handler
}
```

## Best Practices

1. **Store Public Key Securely**: Fetch from Vault, not from environment variables
2. **Use `GetUserIDOrError()` Instead of `MustGetUserID()`**: Return errors instead of panicking to prevent service crashes on middleware misconfiguration
3. **Handle Key Rotation**: Update the validator's public key when rotated
4. **Validate Issuer/Audience**: Always set these to prevent token misuse
5. **Use HTTPS in Production**: JWT tokens should only be transmitted over TLS
6. **Set Short Token Expiry**: Default is 15 minutes for access tokens
7. **Log Authentication Failures**: For security monitoring and debugging

## Key Rotation Support

When the user service rotates its JWT signing keys, consuming services need to update their public key:

```go
// Periodic key refresh (every 1 hour)
go func() {
    ticker := time.NewTicker(1 * time.Hour)
    defer ticker.Stop()

    for range ticker.C {
        newPublicKey, err := vaultClient.GetPublicKey("secret/jwt/public-key")
        if err != nil {
            log.Printf("Failed to refresh public key: %v", err)
            continue
        }
        validator.UpdatePublicKey(newPublicKey)
        log.Println("JWT public key refreshed")
    }
}()
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

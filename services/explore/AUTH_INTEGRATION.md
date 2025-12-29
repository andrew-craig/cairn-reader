# Authentication Integration Guide

This document describes how JWT authentication from the Cairn User Service has been integrated into the Explore/Recommender service.

## Overview

The Explore service now uses the shared authentication package (`github.com/andrew-craig/cairn/services/users/pkg/auth`) to validate JWT tokens issued by the User Service. This enables secure, stateless authentication across all protected endpoints.

## Architecture

```
┌──────────────┐         ┌──────────────────┐         ┌─────────────────┐
│  User Service │         │ HashiCorp Vault  │         │ Explore Service │
│              │         │                  │         │                 │
│ - Issues JWT │────────▶│ Stores JWT Keys  │◀────────│ - Validates JWT │
│   tokens     │         │   - Private Key  │         │   tokens        │
│              │         │   - Public Key   │         │                 │
└──────────────┘         └──────────────────┘         └─────────────────┘
```

## Configuration

### Environment Variables

Add these environment variables to configure JWT authentication:

```bash
# Vault Configuration
VAULT_ADDR=http://localhost:8200          # Vault server address
VAULT_TOKEN=your-vault-token              # Vault authentication token

# JWT Public Key Path
JWT_PUBLIC_KEY_PATH=secret/jwt/public-key # Path to JWT public key in Vault
```

### Docker Compose

Update your `docker-compose.yml` to include Vault configuration:

```yaml
services:
  recommender:
    environment:
      - VAULT_ADDR=http://vault:8200
      - VAULT_TOKEN=${VAULT_TOKEN}
      - JWT_PUBLIC_KEY_PATH=secret/jwt/public-key
```

## Protected Endpoints

The following endpoints now require JWT authentication:

### Recommendations
- **GET** `/api/v1/recommendations/`
  - Requires: Valid JWT token in Authorization header
  - User ID extracted from JWT (not from path)

### Article Interactions
- **POST** `/api/v1/articles/read`
  - Requires: Valid JWT token
  - Request body: `{"article_id": "..."}`
  - User ID extracted from JWT (no longer in request body)

### Voting
- **POST** `/api/v1/articles/:id/vote`
  - Requires: Valid JWT token
  - Request body: `{"vote_type": "upvote" | "downvote"}`
  - User ID extracted from JWT

- **DELETE** `/api/v1/articles/:id/vote`
  - Requires: Valid JWT token
  - User ID extracted from JWT

- **GET** `/api/v1/articles/:id/votes`
  - Requires: Valid JWT token
  - Returns vote counts

## Public Endpoints

These endpoints remain publicly accessible:

- **GET** `/health` - Health check
- **POST** `/api/v1/articles` - Receive articles from fetcher (internal use)
- **GET** `/api/v1/articles` - List all articles

## API Changes

### Before (Insecure)
```bash
# User ID was sent in request body or path - easily spoofed!
POST /api/v1/articles/read
{
  "user_id": "user123",  # ❌ User could lie about their ID
  "article_id": "article456"
}
```

### After (Secure)
```bash
# User ID extracted from verified JWT token
POST /api/v1/articles/read
Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...
{
  "article_id": "article456"  # ✅ User ID from JWT, cannot be spoofed
}
```

## Making Authenticated Requests

### 1. Obtain JWT Token

First, authenticate with the User Service:

```bash
# Register or login
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password123"
  }'

# Response contains access_token
{
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "...",
  "expires_in": 3600,
  "user": {...}
}
```

### 2. Use Token in Requests

Include the JWT token in the `Authorization` header:

```bash
# Get recommendations
curl http://localhost:8081/api/v1/recommendations/ \
  -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."

# Mark article as read
curl -X POST http://localhost:8081/api/v1/articles/read \
  -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -H "Content-Type: application/json" \
  -d '{"article_id": "article123"}'

# Vote on article
curl -X POST http://localhost:8081/api/v1/articles/article123/vote \
  -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -H "Content-Type: application/json" \
  -d '{"vote_type": "upvote"}'
```

## Error Responses

### 401 Unauthorized

Returned when JWT token is missing, invalid, or expired:

```json
{
  "error": "unauthorized",
  "message": "token has expired"
}
```

Common causes:
- Missing `Authorization` header
- Invalid token format (not "Bearer <token>")
- Expired token (tokens expire after 60 minutes)
- Invalid signature
- Wrong issuer or audience

### 403 Forbidden

Reserved for authorization failures (not currently implemented but available for future use).

## Implementation Details

### Server Initialization

In [cmd/recommender/main.go](recommender/cmd/recommender/main.go):

```go
// Connect to Vault
vaultClient, err := auth.NewVaultClient(&auth.VaultConfig{
    Address: vaultAddr,
    Token:   vaultToken,
})

// Fetch JWT public key
publicKey, err := vaultClient.GetPublicKey(publicKeyPath)

// Create validator and middleware
validator := auth.NewValidator(publicKey)
authMiddleware := auth.NewMiddleware(validator)

// Pass to server
server := api.NewServer(articleRepo, userRepo, voteRepo, engine, authMiddleware)
```

### Route Protection

In [internal/api/server.go](recommender/internal/api/server.go):

```go
// Protected endpoint
mux.Handle("/api/v1/recommendations/", s.authMiddleware.RequireAuth(
    http.HandlerFunc(s.handleRecommendations),
))

// Public endpoint
mux.HandleFunc("/health", s.handleHealth)
```

### User ID Extraction

In handlers ([internal/api/handlers.go](recommender/internal/api/handlers.go)):

```go
// Extract authenticated user ID from JWT context
authenticatedUserID := auth.MustGetUserID(r.Context())
userID := authenticatedUserID.String()

// Use userID for database operations
recommendations, err := s.engine.GetRecommendations(r.Context(), userID)
```

## Security Benefits

### Before Integration
- ❌ Users could impersonate other users by changing `user_id` in requests
- ❌ No authentication required for personal data
- ❌ Vulnerable to unauthorized access
- ❌ No audit trail of who accessed what

### After Integration
- ✅ User identity verified cryptographically via JWT
- ✅ Cannot spoof user ID (extracted from signed token)
- ✅ Tokens expire automatically (60 minutes)
- ✅ Stateless authentication (no session storage needed)
- ✅ Centralized authentication via User Service
- ✅ Audit trail via token claims (user_id, iat, exp)

## Testing

### Manual Testing

1. Start Vault:
```bash
docker run --rm -p 8200:8200 \
  -e 'VAULT_DEV_ROOT_TOKEN_ID=dev-token' \
  hashicorp/vault
```

2. Store JWT public key in Vault (get from User Service)

3. Start services:
```bash
# User Service
cd services/users
go run cmd/user-service/main.go

# Explore Service
cd services/explore
docker-compose up
```

4. Test authentication flow:
```bash
# 1. Login to get token
TOKEN=$(curl -s -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123"}' \
  | jq -r '.access_token')

# 2. Access protected endpoint
curl http://localhost:8081/api/v1/recommendations/ \
  -H "Authorization: Bearer $TOKEN"

# 3. Try without token (should fail with 401)
curl http://localhost:8081/api/v1/recommendations/
```

## Key Rotation

When the User Service rotates its JWT signing keys:

1. New public key is stored in Vault
2. Explore service automatically fetches updated key on next request
3. Old tokens remain valid until expiration
4. New tokens use new key immediately

To force key refresh without restarting:
```go
// Periodic refresh (future enhancement)
go func() {
    ticker := time.NewTicker(1 * time.Hour)
    for range ticker.C {
        newKey, _ := vaultClient.GetPublicKey(publicKeyPath)
        validator.UpdatePublicKey(newKey)
    }
}()
```

## Troubleshooting

### "Failed to connect to Vault"
- Check `VAULT_ADDR` environment variable
- Verify Vault is running: `curl $VAULT_ADDR/v1/sys/health`
- Check Vault token is valid

### "Failed to get JWT public key"
- Verify path in Vault: `vault kv get secret/jwt/public-key`
- Check Vault token has read permissions
- Ensure User Service has written the public key

### "Unauthorized" on every request
- Verify User Service is issuing valid tokens
- Check issuer and audience match (default: "cairn-user-service", "cairn-api")
- Verify public key in Vault matches User Service private key

### "Token has expired"
- Tokens expire after 60 minutes
- Use refresh token to get new access token
- Check system clock sync between services

## Future Enhancements

- [ ] Automatic periodic key refresh from Vault
- [ ] Rate limiting per user (using user ID from JWT)
- [ ] Scope-based authorization (admin vs regular user)
- [ ] Service-to-service authentication for fetcher
- [ ] Token revocation support
- [ ] Metrics on authentication failures

## References

- [Shared Auth Package README](../users/pkg/auth/README.md)
- [User Service Documentation](../users/README.md)
- [JWT Best Practices](https://tools.ietf.org/html/rfc8725)
- [HashiCorp Vault Documentation](https://www.vaultproject.io/docs)

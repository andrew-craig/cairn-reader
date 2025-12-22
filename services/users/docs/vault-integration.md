# HashiCorp Vault Integration Guide

This document describes how the Cairn User Service integrates with HashiCorp Vault for secrets management.

## Overview

The User Service uses HashiCorp Vault to securely store and retrieve:
- JWT RSA private/public key pairs for token signing and validation
- Database credentials
- Other sensitive configuration

Vault integration is **mandatory** for production deployments per the security requirements.

## Configuration

### Environment Variables

Configure Vault connection using these environment variables (see `.env.example`):

```bash
# Vault server address
VAULT_ADDR=http://localhost:8200

# Development: Use token authentication
VAULT_TOKEN=your_vault_token_here

# Production: Use AppRole authentication
VAULT_ROLE_ID=your_role_id
VAULT_SECRET_ID=your_secret_id
VAULT_AUTH_PATH=approle

# Optional: Vault namespace (for Vault Enterprise)
VAULT_NAMESPACE=

# Paths to secrets in Vault
JWT_PRIVATE_KEY_PATH=secret/data/jwt/private-key
JWT_PUBLIC_KEY_PATH=secret/data/jwt/public-key
VAULT_DB_CREDS_PATH=secret/data/database/credentials

# Key rotation and token renewal intervals
JWT_KEY_ROTATION_INTERVAL=24h
VAULT_TOKEN_RENEWAL_INTERVAL=1h
```

## Authentication Methods

### Development: Token Authentication

For local development, use a Vault token:

```bash
export VAULT_ADDR=http://localhost:8200
export VAULT_TOKEN=your_dev_token
```

### Production: AppRole Authentication

For production, use AppRole authentication:

```bash
export VAULT_ADDR=https://vault.production.com
export VAULT_ROLE_ID=your_role_id
export VAULT_SECRET_ID=your_secret_id
export VAULT_AUTH_PATH=approle
```

AppRole provides secure, automated authentication without embedding long-lived tokens.

## Setting Up Secrets in Vault

### JWT Keys

1. Generate RSA key pair (minimum 2048-bit):

```bash
# Generate private key
openssl genrsa -out private.pem 2048

# Extract public key
openssl rsa -in private.pem -pubout -out public.pem
```

2. Store in Vault (KV v2):

```bash
# Store private key
vault kv put secret/jwt/private-key key=@private.pem

# Store public key
vault kv put secret/jwt/public-key key=@public.pem

# Securely delete local copies
shred -u private.pem public.pem
```

### Database Credentials

Store database credentials in Vault:

```bash
vault kv put secret/database/credentials \
  host=db.example.com \
  port=5432 \
  username=cairn_user \
  password=secure_password \
  database=cairn_users \
  sslmode=require
```

## Code Usage

### Basic Vault Client

```go
import "github.com/andrew-craig/cairn-core/user-service/internal/auth"

// Create Vault client
vaultClient, err := auth.NewVaultClient(&auth.VaultConfig{
    Address:  "http://localhost:8200",
    Token:    "dev-token",
})
if err != nil {
    log.Fatal(err)
}

// Check Vault health
if err := vaultClient.Health(); err != nil {
    log.Fatal("Vault is not healthy:", err)
}
```

### Retrieving JWT Keys

```go
// Get JWT keys from Vault
privateKey, publicKey, err := vaultClient.GetJWTKeys(
    "secret/data/jwt/private-key",
    "secret/data/jwt/public-key",
)
if err != nil {
    log.Fatal(err)
}

// Use keys for JWT signing/validation
```

### Retrieving Database Credentials

```go
// Get database credentials from Vault
dbCreds, err := vaultClient.GetDatabaseCredentials(
    "secret/data/database/credentials",
)
if err != nil {
    log.Fatal(err)
}

// Use credentials to connect to database
connStr := fmt.Sprintf(
    "host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
    dbCreds.Host,
    dbCreds.Port,
    dbCreds.Username,
    dbCreds.Password,
    dbCreds.Database,
    dbCreds.SSLMode,
)
```

### Key Rotation Manager

The `KeyRotationManager` handles automatic key rotation and token renewal:

```go
import (
    "context"
    "time"
    "github.com/andrew-craig/cairn-core/user-service/internal/auth"
)

// Create rotation manager
rotationManager, err := auth.NewKeyRotationManager(vaultClient, auth.KeyRotationConfig{
    PrivateKeyPath:       "secret/data/jwt/private-key",
    PublicKeyPath:        "secret/data/jwt/public-key",
    RotationInterval:     24 * time.Hour,  // Check for new keys every 24 hours
    TokenRenewalInterval: 1 * time.Hour,   // Renew Vault token every hour
    OnRotation: func(keyPair *auth.JWTKeyPair) error {
        // Optional callback when keys rotate
        log.Println("JWT keys rotated successfully")
        return nil
    },
})
if err != nil {
    log.Fatal(err)
}

// Start background rotation
ctx := context.Background()
rotationManager.Start(ctx)
defer rotationManager.Stop()

// Get current keys (thread-safe)
keyPair := rotationManager.GetCurrentKeyPair()

// Force immediate rotation (for testing or manual rotation)
if err := rotationManager.ForceRotation(); err != nil {
    log.Println("Manual rotation failed:", err)
}
```

## Key Rotation

### Automatic Rotation

The `KeyRotationManager` periodically checks Vault for updated keys. To rotate keys:

1. Generate new key pair
2. Store new keys in Vault (same paths)
3. The rotation manager will automatically pick them up on the next interval
4. Optional: Call `ForceRotation()` for immediate rotation

### Rotation Intervals

- **JWT_KEY_ROTATION_INTERVAL**: How often to check for new keys (default: 24h)
- **VAULT_TOKEN_RENEWAL_INTERVAL**: How often to renew Vault token (default: 1h)

Set to `0` to disable either interval.

### Rotation Callback

Provide an `OnRotation` callback to be notified when keys rotate:

```go
OnRotation: func(keyPair *auth.JWTKeyPair) error {
    // Update JWT service with new keys
    // Log rotation event
    // Send metrics
    return nil
}
```

If the callback returns an error, rotation is rolled back to the previous keys.

## Token Renewal

For token-based authentication, the `KeyRotationManager` automatically renews the Vault token to prevent expiration:

- Renewal happens at the `VAULT_TOKEN_RENEWAL_INTERVAL`
- For AppRole authentication, tokens are automatically renewed when they expire
- Renewal failures are logged but don't stop the service

## Health Checks

Check Vault connectivity and status:

```go
if err := vaultClient.Health(); err != nil {
    // Vault is unavailable, sealed, or not initialized
    log.Fatal(err)
}
```

Health checks verify:
- Vault is reachable
- Vault is initialized
- Vault is unsealed

## Security Best Practices

### Production Deployment

1. **Use AppRole authentication** (not tokens)
2. **Enable TLS** for Vault connections
3. **Use Vault namespaces** for multi-tenancy (Enterprise)
4. **Rotate keys regularly** (configure `JWT_KEY_ROTATION_INTERVAL`)
5. **Set appropriate Vault policies** (principle of least privilege)
6. **Monitor token renewal** and key rotation
7. **Never log secrets** (tokens, passwords, private keys)

### Vault Policies

Example Vault policy for the User Service:

```hcl
# Read JWT keys
path "secret/data/jwt/*" {
  capabilities = ["read"]
}

# Read database credentials
path "secret/data/database/credentials" {
  capabilities = ["read"]
}

# Renew own token
path "auth/token/renew-self" {
  capabilities = ["update"]
}

# Read own token info
path "auth/token/lookup-self" {
  capabilities = ["read"]
}
```

### Secrets Organization

Recommended Vault KV v2 structure:

```
secret/
├── jwt/
│   ├── private-key
│   └── public-key
├── database/
│   └── credentials
└── user-service/
    └── config
```

## Troubleshooting

### Connection Issues

```
Error: failed to create vault client: connection refused
```

- Check `VAULT_ADDR` is correct
- Ensure Vault server is running and accessible
- Verify network connectivity

### Authentication Failures

```
Error: failed to authenticate with vault: permission denied
```

- Verify `VAULT_TOKEN` or AppRole credentials are correct
- Check Vault policy allows required operations
- Ensure token hasn't expired

### Sealed Vault

```
Error: vault is sealed
```

- Vault needs to be unsealed using unseal keys
- Contact Vault administrator

### Missing Secrets

```
Error: secret not found at path: secret/data/jwt/private-key
```

- Verify secret exists in Vault
- Check path is correct (KV v2 uses `secret/data/...`)
- Ensure Vault policy allows read access

### Key Rotation Failures

```
Key rotation failed: failed to fetch new keys
```

- Check Vault connectivity
- Verify keys still exist in Vault
- Check Vault policies allow read access
- Review rotation callback errors

## Testing

### Local Development with Vault

1. Start Vault in dev mode:

```bash
vault server -dev -dev-root-token-id=root
```

2. Set environment variables:

```bash
export VAULT_ADDR=http://localhost:8200
export VAULT_TOKEN=root
```

3. Create test secrets:

```bash
# Generate and store test JWT keys
./scripts/generate-test-keys.sh

# Store test database credentials
vault kv put secret/database/credentials \
  host=localhost \
  port=5432 \
  username=test_user \
  password=test_pass \
  database=test_db \
  sslmode=disable
```

### Integration Tests

Run Vault integration tests:

```bash
# Unit tests (no Vault required)
go test ./internal/auth/...

# Integration tests (requires running Vault)
VAULT_ADDR=http://localhost:8200 \
VAULT_TOKEN=root \
go test ./internal/auth/... -tags=integration
```

## Monitoring

Monitor these metrics:

- Vault connection health
- Token renewal success/failure rate
- Key rotation success/failure rate
- Time since last key rotation
- Secret read latency

## References

- [HashiCorp Vault Documentation](https://www.vaultproject.io/docs)
- [Vault AppRole Authentication](https://www.vaultproject.io/docs/auth/approle)
- [Vault KV Secrets Engine v2](https://www.vaultproject.io/docs/secrets/kv/kv-v2)
- [Vault Token Authentication](https://www.vaultproject.io/docs/auth/token)

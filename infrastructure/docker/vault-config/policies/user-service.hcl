# Vault policy for user-service
# This service needs read access to both private and public JWT keys
# to sign and verify JWT tokens

# Read JWT private key (for signing tokens)
path "secret/data/jwt/private-key" {
  capabilities = ["read"]
}

# Read JWT public key (for verifying tokens)
path "secret/data/jwt/public-key" {
  capabilities = ["read"]
}

# Allow token renewal (for long-running services)
path "auth/token/renew-self" {
  capabilities = ["update"]
}

# Allow token lookup (for health checks)
path "auth/token/lookup-self" {
  capabilities = ["read"]
}

# Vault policy for explore-recommender service
# This service only needs read access to the public JWT key
# to verify tokens (it does NOT sign tokens)

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

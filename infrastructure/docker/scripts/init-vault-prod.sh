#!/bin/sh
set -e

# Production Vault initialization script
# This script initializes Vault, creates AppRoles for each service,
# and stores JWT keys securely.
#
# IMPORTANT: This script is designed for production use.
# - Root token is NEVER logged or stored in accessible locations
# - Each service gets its own AppRole with scoped permissions
# - Unseal keys should be distributed to trusted operators

echo "=== Vault Production Initialization ==="
echo "Waiting for Vault to be ready..."

# Wait for Vault to be available
max_attempts=30
attempt=0
while [ $attempt -lt $max_attempts ]; do
    if wget -q --spider http://vault:8200/v1/sys/health?standbyok=true&uninitcode=200&sealedcode=200 2>/dev/null; then
        echo "Vault is available"
        break
    fi
    attempt=$((attempt + 1))
    echo "Waiting for Vault... (attempt $attempt/$max_attempts)"
    sleep 2
done

if [ $attempt -eq $max_attempts ]; then
    echo "ERROR: Vault did not become available"
    exit 1
fi

# Function to check if Vault is initialized
is_vault_initialized() {
    vault status 2>&1 | grep -q "Initialized.*true"
}

# Function to check if Vault is sealed
is_vault_sealed() {
    vault status 2>&1 | grep -q "Sealed.*true"
}

# Initialize Vault if not already initialized
if ! is_vault_initialized; then
    echo "Initializing Vault with 5 key shares, 3 threshold..."

    # Initialize with production-appropriate key shares
    # In production, these keys should be distributed to different operators
    vault operator init -key-shares=5 -key-threshold=3 -format=json > /tmp/init-keys.json

    # Extract unseal keys and root token
    UNSEAL_KEY_1=$(cat /tmp/init-keys.json | grep -o '"unseal_keys_b64":\s*\[[^]]*\]' | grep -o '"[^"]*"' | sed -n '1p' | tr -d '"')
    UNSEAL_KEY_2=$(cat /tmp/init-keys.json | grep -o '"unseal_keys_b64":\s*\[[^]]*\]' | grep -o '"[^"]*"' | sed -n '2p' | tr -d '"')
    UNSEAL_KEY_3=$(cat /tmp/init-keys.json | grep -o '"unseal_keys_b64":\s*\[[^]]*\]' | grep -o '"[^"]*"' | sed -n '3p' | tr -d '"')
    ROOT_TOKEN=$(cat /tmp/init-keys.json | grep -o '"root_token":\s*"[^"]*"' | grep -o '"[^"]*"$' | tr -d '"')

    # Unseal Vault (need 3 of 5 keys)
    echo "Unsealing Vault..."
    vault operator unseal "$UNSEAL_KEY_1" > /dev/null
    vault operator unseal "$UNSEAL_KEY_2" > /dev/null
    vault operator unseal "$UNSEAL_KEY_3" > /dev/null

    export VAULT_TOKEN="$ROOT_TOKEN"

    # Store unseal keys securely (operator should retrieve and delete this file)
    # In a real production environment, use a more secure method like Shamir's Secret Sharing
    # with keys distributed to different operators
    cat > /vault-keys/UNSEAL_KEYS.txt << EOF
===========================================
VAULT UNSEAL KEYS - KEEP SECURE AND DELETE
===========================================
These keys are required to unseal Vault after a restart.
Distribute these to trusted operators and DELETE this file.

Key 1: $UNSEAL_KEY_1
Key 2: $UNSEAL_KEY_2
Key 3: $UNSEAL_KEY_3
Key 4: $(cat /tmp/init-keys.json | grep -o '"unseal_keys_b64":\s*\[[^]]*\]' | grep -o '"[^"]*"' | sed -n '4p' | tr -d '"')
Key 5: $(cat /tmp/init-keys.json | grep -o '"unseal_keys_b64":\s*\[[^]]*\]' | grep -o '"[^"]*"' | sed -n '5p' | tr -d '"')

Threshold: 3 keys required to unseal
===========================================
EOF

    # Clean up temporary file
    rm -f /tmp/init-keys.json

    echo "Vault initialized and unsealed"
    echo ""
    echo "!!! IMPORTANT !!!"
    echo "Unseal keys have been written to /vault-keys/UNSEAL_KEYS.txt"
    echo "Retrieve these keys, distribute to operators, and DELETE the file"
    echo ""

else
    echo "Vault is already initialized"

    # Check if sealed and try to unseal
    if is_vault_sealed; then
        echo "Vault is sealed"

        # Try to unseal using keys from file if available
        if [ -f /vault-keys/UNSEAL_KEYS.txt ]; then
            echo "Attempting to unseal using stored keys..."
            UNSEAL_KEY_1=$(grep "Key 1:" /vault-keys/UNSEAL_KEYS.txt | cut -d: -f2 | tr -d ' ')
            UNSEAL_KEY_2=$(grep "Key 2:" /vault-keys/UNSEAL_KEYS.txt | cut -d: -f2 | tr -d ' ')
            UNSEAL_KEY_3=$(grep "Key 3:" /vault-keys/UNSEAL_KEYS.txt | cut -d: -f2 | tr -d ' ')

            vault operator unseal "$UNSEAL_KEY_1" > /dev/null 2>&1 || true
            vault operator unseal "$UNSEAL_KEY_2" > /dev/null 2>&1 || true
            vault operator unseal "$UNSEAL_KEY_3" > /dev/null 2>&1 || true

            if is_vault_sealed; then
                echo "ERROR: Failed to unseal Vault. Manual intervention required."
                exit 1
            fi
            echo "Vault unsealed successfully"
        else
            echo "ERROR: Vault is sealed and no unseal keys found"
            echo "Manual unsealing required"
            exit 1
        fi
    fi
fi

# At this point we need a valid token to continue
# Check if we have a root token from initialization or need to use provided credentials
if [ -z "$VAULT_TOKEN" ]; then
    # Check if this is a subsequent run where everything is already configured
    if [ -f /vault-keys/approle-credentials.env ]; then
        echo "Vault is already initialized and AppRole credentials exist"
        echo "Assuming Vault is fully configured. Skipping further setup."
        echo ""
        echo "=== Vault Already Configured ==="
        echo "If you need to reconfigure, remove the vault-keys volume and restart"
        exit 0
    fi

    # Try to use AppRole if available (for subsequent runs with manual configuration)
    if [ -n "$VAULT_INIT_ROLE_ID" ] && [ -n "$VAULT_INIT_SECRET_ID" ]; then
        echo "Authenticating with AppRole..."
        VAULT_TOKEN=$(vault write -format=json auth/approle/login \
            role_id="$VAULT_INIT_ROLE_ID" \
            secret_id="$VAULT_INIT_SECRET_ID" | grep -o '"client_token":\s*"[^"]*"' | grep -o '"[^"]*"$' | tr -d '"')
        export VAULT_TOKEN
    else
        echo "ERROR: No authentication method available"
        echo "Either run this during initial setup or provide VAULT_INIT_ROLE_ID and VAULT_INIT_SECRET_ID"
        exit 1
    fi
fi

# Enable KV secrets engine if not already enabled
echo "Enabling KV secrets engine..."
vault secrets enable -version=2 -path=secret kv 2>/dev/null || echo "KV secrets engine already enabled"

# Check if JWT keys already exist
if vault kv get secret/jwt/public-key > /dev/null 2>&1; then
    echo "JWT keys already exist in Vault"
else
    echo "Generating new RSA key pair for JWT signing..."

    # Generate RSA private key (2048-bit)
    openssl genrsa -out /tmp/private.pem 2048 2>/dev/null

    # Extract public key from private key
    openssl rsa -in /tmp/private.pem -pubout -out /tmp/public.pem 2>/dev/null

    # Store keys in Vault
    vault kv put secret/jwt/private-key key="$(cat /tmp/private.pem)"
    vault kv put secret/jwt/public-key key="$(cat /tmp/public.pem)"

    # Clean up temporary key files
    rm -f /tmp/private.pem /tmp/public.pem

    echo "JWT keys stored in Vault"
fi

# Verify keys
echo "Verifying JWT keys..."
vault kv get secret/jwt/private-key > /dev/null && echo "  - Private key: OK"
vault kv get secret/jwt/public-key > /dev/null && echo "  - Public key: OK"

# Enable AppRole authentication if not already enabled
echo "Enabling AppRole authentication..."
vault auth enable approle 2>/dev/null || echo "AppRole already enabled"

# Create policies
echo "Creating service policies..."

# User Service Policy (needs both private and public keys)
vault policy write user-service - <<EOF
# User Service Policy
# Access to JWT keys for signing and verification

path "secret/data/jwt/private-key" {
  capabilities = ["read"]
}

path "secret/data/jwt/public-key" {
  capabilities = ["read"]
}

path "auth/token/renew-self" {
  capabilities = ["update"]
}

path "auth/token/lookup-self" {
  capabilities = ["read"]
}
EOF
echo "  - user-service policy created"

# Explore Recommender Policy (only needs public key)
vault policy write explore-recommender - <<EOF
# Explore Recommender Policy
# Read-only access to JWT public key for verification

path "secret/data/jwt/public-key" {
  capabilities = ["read"]
}

path "auth/token/renew-self" {
  capabilities = ["update"]
}

path "auth/token/lookup-self" {
  capabilities = ["read"]
}
EOF
echo "  - explore-recommender policy created"

# Create AppRoles for each service
echo "Creating AppRoles..."

# User Service AppRole
vault write auth/approle/role/user-service \
    token_policies="user-service" \
    token_ttl=1h \
    token_max_ttl=4h \
    secret_id_ttl=0 \
    secret_id_num_uses=0

USER_SERVICE_ROLE_ID=$(vault read -format=json auth/approle/role/user-service/role-id | grep -o '"role_id":\s*"[^"]*"' | grep -o '"[^"]*"$' | tr -d '"')
USER_SERVICE_SECRET_ID=$(vault write -format=json -f auth/approle/role/user-service/secret-id | grep -o '"secret_id":\s*"[^"]*"' | grep -o '"[^"]*"$' | tr -d '"')

echo "  - user-service AppRole created"

# Explore Recommender AppRole
vault write auth/approle/role/explore-recommender \
    token_policies="explore-recommender" \
    token_ttl=1h \
    token_max_ttl=4h \
    secret_id_ttl=0 \
    secret_id_num_uses=0

EXPLORE_RECOMMENDER_ROLE_ID=$(vault read -format=json auth/approle/role/explore-recommender/role-id | grep -o '"role_id":\s*"[^"]*"' | grep -o '"[^"]*"$' | tr -d '"')
EXPLORE_RECOMMENDER_SECRET_ID=$(vault write -format=json -f auth/approle/role/explore-recommender/secret-id | grep -o '"secret_id":\s*"[^"]*"' | grep -o '"[^"]*"$' | tr -d '"')

echo "  - explore-recommender AppRole created"

# Write AppRole credentials to file for services to use
# In production, consider using Docker secrets or another secure method
cat > /vault-keys/approle-credentials.env << EOF
# AppRole Credentials for Services
# Generated by init-vault-prod.sh
# These credentials allow services to authenticate with Vault

# User Service
USER_SERVICE_ROLE_ID=$USER_SERVICE_ROLE_ID
USER_SERVICE_SECRET_ID=$USER_SERVICE_SECRET_ID

# Explore Recommender
EXPLORE_RECOMMENDER_ROLE_ID=$EXPLORE_RECOMMENDER_ROLE_ID
EXPLORE_RECOMMENDER_SECRET_ID=$EXPLORE_RECOMMENDER_SECRET_ID
EOF

chmod 600 /vault-keys/approle-credentials.env

echo ""
echo "=== Vault Initialization Complete ==="
echo ""
echo "AppRole credentials have been written to /vault-keys/approle-credentials.env"
echo ""
echo "Service Configuration:"
echo "  User Service:"
echo "    VAULT_ROLE_ID=$USER_SERVICE_ROLE_ID"
echo "    VAULT_SECRET_ID=$USER_SERVICE_SECRET_ID"
echo ""
echo "  Explore Recommender:"
echo "    VAULT_ROLE_ID=$EXPLORE_RECOMMENDER_ROLE_ID"
echo "    VAULT_SECRET_ID=$EXPLORE_RECOMMENDER_SECRET_ID"
echo ""
echo "SECURITY REMINDERS:"
echo "  1. Retrieve and securely store the unseal keys from /vault-keys/UNSEAL_KEYS.txt"
echo "  2. Delete the unseal keys file after distributing to operators"
echo "  3. The root token was NOT saved - generate a new one only when needed"
echo "  4. Consider rotating AppRole secret IDs periodically"
echo ""

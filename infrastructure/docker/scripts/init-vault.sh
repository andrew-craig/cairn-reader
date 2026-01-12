#!/bin/sh
set -e

echo "Waiting for Vault to be ready..."
sleep 5

# Check if Vault is initialized
if ! vault status > /dev/null 2>&1; then
    echo "Initializing Vault..."
    vault operator init -key-shares=1 -key-threshold=1 -format=json > /vault-keys/init-keys.json

    UNSEAL_KEY=$(grep unseal_keys_b64 /vault-keys/init-keys.json -A 1 | tail -n 1 | sed 's/[^"]*"\([^"]*\)".*/\1/')
    ROOT_TOKEN=$(grep root_token /vault-keys/init-keys.json | sed 's/.*"root_token"[: ]*"\([^"]*\)".*/\1/')

    echo "Unsealing Vault..."
    echo "DEBUG: UNSEAL_KEY length: ${#UNSEAL_KEY}"
    if [ -z "$UNSEAL_KEY" ]; then
        echo "ERROR: UNSEAL_KEY is empty!"
        cat /vault-keys/init-keys.json
        exit 1
    fi
    vault operator unseal "$UNSEAL_KEY" > /dev/null

    # Update VAULT_TOKEN for this session
    export VAULT_TOKEN="$ROOT_TOKEN"

    echo "Enabling KV secrets engine at secret/ path..."
    vault secrets enable -version=2 -path=secret kv || echo "KV secrets engine may already be enabled"

    echo "Vault initialized and unsealed successfully"
    echo "IMPORTANT: Root token is stored in /vault-keys/init-keys.json"
    echo "Root token: $ROOT_TOKEN"
else
    # Check if Vault is sealed
    if vault status 2>&1 | grep -q "Sealed.*true"; then
        echo "Vault is sealed, attempting to unseal..."
        if [ -f /vault-keys/init-keys.json ]; then
            UNSEAL_KEY=$(grep unseal_keys_b64 /vault-keys/init-keys.json -A 1 | tail -n 1 | sed 's/[^"]*"\([^"]*\)".*/\1/')
            vault operator unseal "$UNSEAL_KEY"
            ROOT_TOKEN=$(grep root_token /vault-keys/init-keys.json | sed 's/.*"root_token"[: ]*"\([^"]*\)".*/\1/')
            export VAULT_TOKEN="$ROOT_TOKEN"
            echo "Vault unsealed successfully"
        else
            echo "ERROR: Vault is sealed but unseal keys not found"
            exit 1
        fi
    else
        echo "Vault is already initialized and unsealed"
    fi
fi

# Ensure we have a valid token
if [ -f /vault-keys/init-keys.json ] && [ -z "$VAULT_TOKEN" ]; then
    ROOT_TOKEN=$(grep root_token /vault-keys/init-keys.json | sed 's/.*"root_token"[: ]*"\([^"]*\)".*/\1/')
    export VAULT_TOKEN="$ROOT_TOKEN"
fi

# Check if keys already exist in Vault
if vault kv get secret/jwt/public-key > /dev/null 2>&1 && vault kv get secret/jwt/private-key > /dev/null 2>&1; then
    echo "JWT keys already exist in Vault, skipping generation..."
    echo "✓ Private key verified"
    echo "✓ Public key verified"
    echo "Vault initialization complete!"
    exit 0
fi

echo "JWT keys not found in Vault, generating new RSA key pair..."

# Generate RSA private key (2048-bit)
openssl genrsa -out /vault-keys/private.pem 2048

# Extract public key from private key
openssl rsa -in /vault-keys/private.pem -pubout -out /vault-keys/public.pem

echo "Storing JWT keys in Vault..."

# Store private key in Vault
vault kv put secret/jwt/private-key key="$(cat /vault-keys/private.pem)"

# Store public key in Vault
vault kv put secret/jwt/public-key key="$(cat /vault-keys/public.pem)"

echo "JWT keys successfully stored in Vault at:"
echo "  - secret/jwt/private-key"
echo "  - secret/jwt/public-key"

# Verify keys were stored
echo "Verifying keys..."
vault kv get secret/jwt/private-key > /dev/null && echo "✓ Private key verified"
vault kv get secret/jwt/public-key > /dev/null && echo "✓ Public key verified"

echo "Vault initialization complete!"

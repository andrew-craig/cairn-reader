#!/bin/sh
set -e

echo "Waiting for Vault to be ready..."
sleep 2

echo "Generating RSA key pair for JWT signing..."

# Generate RSA private key (2048-bit)
openssl genrsa -out /vault-keys/private.pem 2048

# Extract public key from private key
openssl rsa -in /vault-keys/private.pem -pubout -out /vault-keys/public.pem

echo "Storing JWT keys in Vault..."

# Store private key in Vault
vault kv put secret/jwt/private-key value="$(cat /vault-keys/private.pem)"

# Store public key in Vault
vault kv put secret/jwt/public-key value="$(cat /vault-keys/public.pem)"

echo "JWT keys successfully stored in Vault at:"
echo "  - secret/jwt/private-key"
echo "  - secret/jwt/public-key"

# Verify keys were stored
echo "Verifying keys..."
vault kv get secret/jwt/private-key > /dev/null && echo "✓ Private key verified"
vault kv get secret/jwt/public-key > /dev/null && echo "✓ Public key verified"

echo "Vault initialization complete!"

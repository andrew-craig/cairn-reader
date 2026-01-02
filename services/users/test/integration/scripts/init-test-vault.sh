#!/bin/sh
set -e

echo "Waiting for Vault to be ready..."
sleep 2

echo "Generating RSA key pair for JWT signing..."
openssl genrsa -out /tmp/private.pem 2048
openssl rsa -in /tmp/private.pem -pubout -out /tmp/public.pem

echo "Storing JWT private key in Vault..."
PRIVATE_KEY=$(cat /tmp/private.pem)
vault kv put secret/jwt/private-key value="$PRIVATE_KEY"

echo "Storing JWT public key in Vault..."
PUBLIC_KEY=$(cat /tmp/public.pem)
vault kv put secret/jwt/public-key value="$PUBLIC_KEY"

echo "Vault initialization complete!"
echo "JWT keys stored at:"
echo "  - secret/jwt/private-key"
echo "  - secret/jwt/public-key"

# Clean up temporary files
rm -f /tmp/private.pem /tmp/public.pem

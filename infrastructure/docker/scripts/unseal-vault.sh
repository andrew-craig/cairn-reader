#!/bin/sh
# Vault Unseal Watchdog
#
# Unseals Vault after every startup/reboot using keys written by vault-init,
# then monitors continuously and re-unseals if Vault ever becomes sealed again.
#
# This service runs AFTER vault-init (which handles first-time Vault setup
# including initialization and AppRole creation). On subsequent reboots,
# vault-init does not re-run (restart: "no"), so this service handles
# unsealing automatically.

VAULT_ADDR="${VAULT_ADDR:-http://vault:8200}"
UNSEAL_KEYS_FILE="/vault-keys/UNSEAL_KEYS.txt"
CHECK_INTERVAL=30

echo "=== Vault Unseal Watchdog ==="

# Wait for Vault to be HTTP-reachable.
# We accept sealed (sealedcode=200) and uninitialized (uninitcode=200) responses
# here because this service is responsible for unsealing, not for requiring it.
echo "Waiting for Vault to be reachable at ${VAULT_ADDR}..."
max_attempts=60
attempt=0
while [ $attempt -lt $max_attempts ]; do
    if wget -q --spider "${VAULT_ADDR}/v1/sys/health?sealedcode=200&uninitcode=200&standbyok=true" 2>/dev/null; then
        echo "Vault is reachable"
        break
    fi
    attempt=$((attempt + 1))
    echo "  Attempt $attempt/$max_attempts..."
    sleep 2
done

if [ $attempt -ge $max_attempts ]; then
    echo "ERROR: Vault not reachable after $((max_attempts * 2)) seconds"
    exit 1
fi

# Returns 0 (true) if Vault is sealed, 1 (false) if unsealed or unreachable.
is_sealed() {
    vault status 2>/dev/null | grep -q "Sealed.*true"
}

# Unseal Vault using the keys written to the vault-keys volume by vault-init.
unseal_vault() {
    if ! is_sealed; then
        echo "Vault is already unsealed"
        return 0
    fi

    echo "Vault is sealed - reading unseal keys from ${UNSEAL_KEYS_FILE}..."

    if [ ! -f "${UNSEAL_KEYS_FILE}" ]; then
        echo "ERROR: Unseal keys file not found: ${UNSEAL_KEYS_FILE}"
        echo "Manual intervention required - provide 3 of 5 unseal keys."
        return 1
    fi

    UNSEAL_KEY_1=$(grep "Key 1:" "${UNSEAL_KEYS_FILE}" | cut -d: -f2 | tr -d ' ')
    UNSEAL_KEY_2=$(grep "Key 2:" "${UNSEAL_KEYS_FILE}" | cut -d: -f2 | tr -d ' ')
    UNSEAL_KEY_3=$(grep "Key 3:" "${UNSEAL_KEYS_FILE}" | cut -d: -f2 | tr -d ' ')

    if [ -z "${UNSEAL_KEY_1}" ] || [ -z "${UNSEAL_KEY_2}" ] || [ -z "${UNSEAL_KEY_3}" ]; then
        echo "ERROR: Failed to parse unseal keys from ${UNSEAL_KEYS_FILE}"
        return 1
    fi

    echo "Applying unseal keys (3 of 5 required)..."
    vault operator unseal "${UNSEAL_KEY_1}" > /dev/null 2>&1
    vault operator unseal "${UNSEAL_KEY_2}" > /dev/null 2>&1
    vault operator unseal "${UNSEAL_KEY_3}" > /dev/null 2>&1

    if ! is_sealed; then
        echo "Vault unsealed successfully"
        return 0
    fi

    echo "ERROR: Vault remains sealed after applying unseal keys"
    return 1
}

# --- Main ---

if ! unseal_vault; then
    exit 1
fi

echo "Vault is ready. Monitoring seal status every ${CHECK_INTERVAL}s..."

# Watchdog loop: re-unseal if Vault becomes sealed (e.g. after a HA event)
while true; do
    sleep "${CHECK_INTERVAL}"
    if is_sealed; then
        echo "WARNING: Vault has become sealed - attempting re-unseal..."
        unseal_vault || echo "WARNING: Re-unseal failed, will retry in ${CHECK_INTERVAL}s"
    fi
done

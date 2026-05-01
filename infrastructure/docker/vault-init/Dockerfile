# syntax=docker/dockerfile:1
#
# Custom image for the vault-init / vault-unseal containers.
#
# The stock hashicorp/vault image only ships the `vault` CLI. Our init
# scripts in infrastructure/docker/scripts/ also need:
#   - openssl: generate the RSA key pair used for JWT signing
#   - grep:    GNU grep (busybox grep does not support the `\s` patterns
#              used to parse the JSON output of `vault operator init`)
#
# Pre-installing these at build time avoids two production hazards present
# when packages are installed in the entrypoint via `apk add`:
#   1. The container would need to run as root to write to /usr.
#   2. Container startup would depend on Alpine package mirrors being
#      reachable on every restart.
#
# After installation we drop to the unprivileged `vault` user that the
# base image already creates, so the init container runs with the same
# minimal privileges as the Vault server itself.

# Override at build time to keep the CLI in lockstep with the server, e.g.
# `docker build --build-arg VAULT_VERSION=1.19 .`
ARG VAULT_VERSION=1.18
FROM hashicorp/vault:${VAULT_VERSION}

USER root
RUN apk add --no-cache openssl grep \
    && mkdir -p /vault-keys \
    && chown -R vault:vault /vault-keys

USER vault
ENTRYPOINT ["/bin/sh"]

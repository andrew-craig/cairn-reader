---
id: chore_13b3
title: Users service: remove unused Redis config and VAULT_DB_CREDS_PATH
type: chore
status: closed
priority: 3
labels: [users]
blocked_by: []
parent: epic_6e4d
remote_task_url: null
created_at: 2026-10-13T20:28:03Z
updated_at: 2026-10-13T21:05:05Z
---
services/users config parses REDIS_* and VAULT_DB_CREDS_PATH but nothing uses them. Remove the dead config (and their docs entries in docs/CONFIGURATION.md and the .env examples).

## Plan
- [x] Confirm REDIS_* / VAULT_DB_CREDS_PATH / DBCredsPath / RedisConfig are unused (grep)
- [x] Remove RedisConfig, Config.Redis, VaultConfig.DBCredsPath and their env parsing from services/users/internal/config/config.go
- [x] Remove VAULT_DB_CREDS_PATH row and Redis Configuration section from docs/CONFIGURATION.md
- [x] Check .env examples and compose files for these vars
- [x] go build / vet / test in services/users

## Review
- grep showed the only references were in config.go (no config tests existed, nothing read cfg.Redis or Vault.DBCredsPath).
- Removed the struct fields and parsing; docs/CONFIGURATION.md lost one table row and the Redis section. No .env example or compose file set these variables for the users service, so nothing to change there. DB SSL-mode lines in docs/compose were left alone (parallel branch).
- pkg/middleware's Redis rate limiter is a separate, shared package and was not touched.
- go build, go vet, go test ./... pass in services/users.

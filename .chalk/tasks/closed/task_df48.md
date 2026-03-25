---
id: task_df48
title: Implement API key and JWT auth middleware
type: task
status: closed
priority: 1
labels: []
blocked_by: []
parent: epic_0c4d
created_at: 2026-03-23T07:18:10Z
updated_at: 2026-03-25T07:07:08Z
---
Implement both middleware files:
- services/read/email/internal/api/middleware/apikey.go
- services/read/email/internal/api/middleware/auth.go

## API Key Middleware (apikey.go)
- Validate X-API-Key header against hashed keys in api_keys table
- Uses APIKeyRepository.ValidateKey
- Return 401 if missing/invalid
- Used for POST /api/v1/source/email/ingest endpoint (Cloudflare Worker calls this)

## JWT Auth Middleware (auth.go)  
- Follow the same pattern as content service JWT middleware
- Load RSA public key from Vault at startup
- Validate Bearer token from Authorization header
- Extract user_id from JWT claims and set in context
- Return 401 if missing/invalid token
- Used for user-facing endpoints (address management, sender listing)

## Reference
- Content service JWT middleware: services/read/content/internal/api/middleware/auth.go
- API key repo already implemented: services/read/email/internal/repository/apikey.go

## Tests
- Unit tests for both middleware
- Test missing header, invalid key/token, expired token, valid cases

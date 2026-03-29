---
id: epic_0c4d
title: Email Ingest Service Implementation
type: epic
status: closed
priority: 0
labels: []
blocked_by: []
parent: null
created_at: 2026-03-23T07:17:45Z
updated_at: 2026-03-29T17:59:01Z
---
Implement all remaining components of the email ingest service (services/read/email). The scaffolding is complete — database schema, models, DTOs, repositories, config, and entry points are all built. This epic covers the service layer, handlers, middleware, workers, processors, and infrastructure needed to go from scaffold to working service.

## Scope
- Service layer (AddressService, EmailService, SenderService)
- HTTP handlers (IngestHandler, AddressHandler, SenderHandler)  
- Middleware (API key auth, JWT auth)
- Email processing pipeline (EmailCleaner, ContentExtractor — sanitize only, no readability)
- Background workers (EmailProcessorWorker, OutboxWorker)
- Content Service client with retry/circuit breaker
- Cleanup jobs (RawEmailCleanup, OutboxCleanup)
- Router wiring and worker main wiring
- Dockerfiles and docker-compose integration
- Tests for all new code

## Design Decisions
- 1 address per user, no regeneration
- Open by default (accept all senders, no allowlist)
- Sanitize-only for email HTML (skip readability extraction)
- Cloudflare Worker sends JSON to POST /api/v1/source/email/ingest
- API key auth for ingest endpoint, JWT auth for user-facing endpoints

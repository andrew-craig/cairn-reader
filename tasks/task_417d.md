---
id: task_417d
title: Wire worker main and server dependency injection
type: task
status: open
priority: 1
labels: []
blocked_by: [task_838f,task_fad8]
parent: epic_0c4d
created_at: 2026-03-23T07:19:09Z
updated_at: 2026-03-23T07:19:09Z
---
Complete the wiring in both entry points:

## 1. cmd/email_ingest/main.go (API server)
- Initialize all repositories from db connection
- Initialize all services (AddressService, EmailService, SenderService)
- Load JWT public key from Vault (follow content service pattern)
- Create middleware instances (APIKeyAuth, JWTAuth)
- Pass dependencies to NewRouter
- Update NewRouter signature to accept dependencies

## 2. cmd/email_ingest_worker/main.go (worker)
- Replace TODO comments with actual worker initialization
- Initialize repositories, services, processors
- Initialize ContentServiceClient with config
- Start EmailProcessorWorker goroutines
- Start OutboxWorker goroutines
- Register cleanup jobs with cron scheduler
- Implement graceful shutdown (cancel context, wait for workers)

## Reference
- Fetcher worker main: services/read/fetcher/cmd/worker/main.go
- Content server main: services/read/content/cmd/server/main.go

## Tests
- No unit tests needed (integration tested via docker-compose)

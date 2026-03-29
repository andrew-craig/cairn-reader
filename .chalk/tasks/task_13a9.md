---
id: task_13a9
title: Add Dockerfiles and docker-compose integration
type: task
status: in_progress
priority: 2
labels: []
blocked_by: []
parent: epic_0c4d
created_at: 2026-03-23T07:19:17Z
updated_at: 2026-03-29T02:40:54Z
---
Create Docker infrastructure for the email ingest service.

## 1. Dockerfile (API server)
- Multi-stage build (builder + runtime)
- Build cmd/email_ingest/main.go
- Copy migrations directory
- Follow same pattern as services/read/content/Dockerfile

## 2. Dockerfile.worker  
- Multi-stage build
- Build cmd/email_ingest_worker/main.go
- Copy migrations directory
- Follow same pattern as services/read/content/Dockerfile.worker

## 3. docker-compose.yml updates
- Add email-ingest-service (port 8087)
- Add email-ingest-worker
- Add PostgreSQL database (ingest_email) — or add to existing postgres instance
- Add environment variables matching config.go defaults
- Add health check
- Add depends_on for postgres and vault

## 4. Makefile updates (if exists)
- Add build/test/docker targets for email service

## Reference
- services/read/docker-compose.yml for existing service patterns
- infrastructure/docker/ for overall Docker setup

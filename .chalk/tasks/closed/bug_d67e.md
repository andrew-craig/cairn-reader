---
id: bug_d67e
title: Prod compose: content service missing EMAIL_INGEST_SERVICE_URL
type: bug
status: closed
priority: 1
labels: [infra,read]
blocked_by: []
parent: epic_6e4d
remote_task_url: null
created_at: 2026-10-13T20:28:03Z
updated_at: 2026-10-13T21:03:40Z
---
infrastructure/docker/prod/docker-compose.yml does not set EMAIL_INGEST_SERVICE_URL on the content service, so it falls back to localhost:8087. The subscription aggregator (Newsletters list and Feed/Reads toggle for email senders) calls the Email Ingest service with it, so email senders may be missing from subscriptions in prod (reported via failed_sources). Verify against the dev/selfhost compose and the running prod stack, then set it to the email-ingest service's internal URL.

## Plan
- [x] Find correct internal URL (email-ingest listens on PORT=8087 on cairn-network)
- [x] Set EMAIL_INGEST_SERVICE_URL=http://email-ingest:8087 on prod content-service
- [x] Check other content-service URLs

## Review
Added EMAIL_INGEST_SERVICE_URL=http://email-ingest:8087 to the prod content-service. Dev compose had the same omission (would fall back to localhost:8087 inside the container), so it was fixed identically. Selfhost is a single process with in-process adapter, so not applicable. Other URLs the content config reads: INGEST_RSS_SERVICE_URL is already set in prod and dev; no other service URLs exist. Not verified against the running prod stack.

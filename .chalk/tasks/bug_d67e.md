---
id: bug_d67e
title: Prod compose: content service missing EMAIL_INGEST_SERVICE_URL
type: bug
status: open
priority: 1
labels: [infra,read]
blocked_by: []
parent: epic_6e4d
remote_task_url: null
created_at: 2026-10-13T20:28:03Z
updated_at: 2026-10-13T20:28:03Z
---
infrastructure/docker/prod/docker-compose.yml does not set EMAIL_INGEST_SERVICE_URL on the content service, so it falls back to localhost:8087. The subscription aggregator (Newsletters list and Feed/Reads toggle for email senders) calls the Email Ingest service with it, so email senders may be missing from subscriptions in prod (reported via failed_sources). Verify against the dev/selfhost compose and the running prod stack, then set it to the email-ingest service's internal URL.

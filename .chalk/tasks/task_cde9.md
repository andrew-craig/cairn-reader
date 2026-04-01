---
id: task_cde9
title: Add go build verification job to go-checks.yml
type: task
status: open
priority: 1
labels: [ci,backend]
blocked_by: []
parent: epic_d014
created_at: 2026-04-01T03:36:21Z
updated_at: 2026-04-01T03:36:21Z
---
Add a go build job to go-checks.yml that verifies all service binaries compile successfully. Use matrix strategy matching the go vet job. Build commands per service:
- explore: go build ./fetcher/cmd/explore_fetcher && go build ./recommender/cmd/explore_recommender
- read: cd content && go build ./cmd/content && go build ./cmd/worker; cd fetcher && go build ./cmd/ingest_rss && go build ./cmd/ingest_rss_worker
- read/email: go build ./cmd/email_ingest && go build ./cmd/email_ingest_worker
- users: go build -o /dev/null cmd/user-service/main.go

Tooling status: go build is built into Go. No setup needed.
Runs standalone: YES (no Docker/Postgres required).

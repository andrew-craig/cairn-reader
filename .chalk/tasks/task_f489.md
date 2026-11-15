---
id: task_f489
title: Build per-service container images for linux/amd64 (Intel)
type: task
status: open
priority: 1
labels: [infra,ci,hosting]
blocked_by: []
parent: epic_381e
remote_task_url: null
created_at: 2026-11-14T03:20:05Z
updated_at: 2026-11-14T03:20:05Z
---

The hosted beta (epic_381e) will run on **Intel (x86_64)** servers, but every per-service image workflow builds only `platforms: linux/arm64`, so the images would not run there.

## Scope
Switch these workflows from `linux/arm64` to `linux/amd64`:
- `.github/workflows/docker-build-user-service.yml`
- `.github/workflows/docker-build-content-service.yml`
- `.github/workflows/docker-build-content-worker.yml`
- `.github/workflows/docker-build-ingest-rss.yml`
- `.github/workflows/docker-build-ingest-rss-worker.yml`
- `.github/workflows/docker-build-ingest-email.yml`
- `.github/workflows/docker-build-ingest-email-worker.yml`
- `.github/workflows/docker-build-web.yml`

Out of scope: `docker-build-selfhost.yml` stays `linux/arm64`. It serves the existing ARM self-hosted server and is a separate deployment path.

The Dockerfiles themselves have no `GOARCH` / `--platform` pins (checked 2026-11-14), so only the workflow `platforms:` lines should need to change.

`cairn-vault-init` (`infrastructure/docker/vault-init`) has no publishing workflow. Prod compose has a `build:` context for it, so either build it on the host or add an amd64 workflow for it. Decide as part of this task.

## Dependency
These workflows are disabled with `build-and-push: if: false` (task_4f8a, blocker 1). Images are only published once that gate is removed. Removing it can happen here or in task_4f8a, but must not be left half-done.

## Plan
- [ ] Change `platforms:` to `linux/amd64` in the 8 workflows above
- [ ] Decide how the `cairn-vault-init` image is built for amd64
- [ ] Remove the `if: false` gates (coordinate with task_4f8a)
- [ ] Verify: workflows run green on a PR; on merge to main, GHCR shows amd64 manifests (`docker manifest inspect ghcr.io/<org>/cairn-<service>:latest`)
- [ ] Verify: `docker compose pull && docker compose up -d` from `infrastructure/docker/prod` on an x86_64 host brings every service to healthy

## Done when
All prod-stack images are published to GHCR for linux/amd64 and the prod stack starts on an x86_64 host.

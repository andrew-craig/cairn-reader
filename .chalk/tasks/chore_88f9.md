---
id: chore_88f9
title: apps/web/Dockerfile: mobile-only lockfile changes bust the web build cache
type: chore
status: open
priority: 3
labels: [docker,ci,web]
blocked_by: []
parent: null
remote_task_url: null
created_at: 2026-09-14T12:19:52Z
updated_at: 2026-09-14T12:19:52Z
---
**Source:** flagged as out-of-scope by chore_6449 (merged as #384), which fixed the identical bug in `infrastructure/docker/selfhost/Dockerfile`. Filed 2026-09-14 during a backlog sweep and re-verified against `main` at `ba7ac84`.

## Problem
`apps/web/Dockerfile:10-13` is the pre-#384 selfhost pattern, unchanged:
```dockerfile
COPY package.json package-lock.json ./
COPY apps/web/package.json apps/web/package.json
COPY apps/shared/package.json apps/shared/package.json
RUN npm ci -w @cairn/web -w @cairn/shared --include-workspace-root
```
`apps/mobile` is also a workspace member, so the single root `package-lock.json` changes on any
mobile-only dependency bump. That busts the `COPY package-lock.json` layer and everything after
it — `npm ci`, `npm run build -w @cairn/web` — so the whole image rebuilds from scratch even
though no web or shared dependency moved.

## This is live in CI, via docker-test.yml — not via the publish workflow
Worth stating precisely, because the obvious place to look says otherwise:
- `.github/workflows/docker-build-web.yml` builds this Dockerfile (`:72`) but its
  `build-and-push` job is gated **`if: false`** (`:32`) — that workflow is disabled and is
  **not** where the cost lands.
- `.github/workflows/docker-test.yml`'s **`build-web`** job (`:260-279`) is live and builds
  `apps/web/Dockerfile` (`:274`) on every triggering change. Unlike the `selfhost-compose-smoke`
  job that task_7722 fixed, this one is wired correctly — `needs: [changes]` with `changes`
  actually present, gating on `needs.changes.outputs.web == 'true'`.
- That `web` filter (`docker-test.yml:51-55`) lists `package-lock.json`.

So a mobile-only dependency change trips the `web` filter, runs `build-web`, and pays a
full-from-scratch `npm ci` + vite build. Since #384 the selfhost image no longer pays this; the
web image still does.

## What to do
Port the `lockfile` stage from `infrastructure/docker/selfhost/Dockerfile:1-16` — it is already
written, commented, and verified; this is a transplant, not a design task:
1. Add the `lockfile` stage (`FROM node:24-alpine AS lockfile`) that drops `apps/mobile` from
   `package.json`'s `workspaces` and runs `npm install --package-lock-only --offline`.
2. Point the builder stage at it: `COPY --from=lockfile /app/package.json /app/package-lock.json ./`,
   then plain `npm ci` (the pruned lockfile makes the `-w` flags redundant — see below).

**`node:24` for the lockfile stage is load-bearing.** The selfhost comment records why: npm 11+
is needed to prune the orphaned workspace entry fully. Do not "tidy" it to match the builder
stage's `node:22`.

**The `-w @cairn/web -w @cairn/shared --include-workspace-root` flags go away**, as they did in
selfhost (`:28` is a bare `npm ci`) — once mobile is out of the lockfile there is nothing to
scope against. Dropping them is part of the change, not an unrelated cleanup.

## Done when
- [ ] Pruned lockfile is byte-identical across a mobile-only change (reproduce #384's check: regenerate from two commits that differ only in mobile deps, under `node:24-alpine --network none`, and diff)
- [ ] After a clean build, a synthetic mobile-only dependency change leaves `COPY --from=lockfile`, `npm ci` and `npm run build -w @cairn/web` all `CACHED`
- [ ] A real web or shared dependency change still busts the cache and rebuilds — the negative control, without which "always cached" would pass silently
- [ ] `cairn-web:test` still serves the SPA: `docker-test.yml:280-290` curls `/healthz`, so that job passing is the end-to-end check

## Related, deliberately not folded in
- **chore_3ea1** — the `paths:` filters that make this job trigger at all. Fixing this task makes
  that one much less costly, which is why it is sequenced after.
- `docker-build-web.yml` being disabled at `if: false` is its own question — is the web image
  still meant to be published? Not filed as a task: it needs a product answer, not a fix, and
  guessing at one would be inventing scope. Noted here so it is on the record.

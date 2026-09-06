---
id: chore_6449
title: Selfhost image: mobile-only root-lockfile changes bust the web build cache
type: chore
status: in_progress
priority: 2
labels: [selfhost,docker,ci]
blocked_by: []
parent: null
remote_task_url: null
created_at: 2026-09-06T13:16:08Z
updated_at: 2026-09-06T13:16:11Z
---

## Problem

`infrastructure/docker/selfhost/Dockerfile` `web-builder` stage does:

```
COPY package.json package-lock.json ./
COPY apps/web/package.json apps/web/package.json
COPY apps/shared/package.json apps/shared/package.json
RUN npm ci -w @cairn/web -w @cairn/shared --include-workspace-root
```

`apps/mobile` is also a workspace member, so the single root `package-lock.json`
changes whenever a mobile-only dependency changes (e.g. #382 added `expo-sqlite`
+ `await-lock`). That busts the `COPY package-lock.json` layer and everything
after it — `npm ci`, `npm run build -w @cairn/web` — so the whole web half of the
image rebuilds from scratch even though not a single web/shared dependency moved.
The `.github/workflows/docker-build-selfhost.yml` + `selfhost-compose-smoke.yml`
`paths:` filters also list `package-lock.json`, so a pure-mobile PR triggers the
full build + compose smoke.

## Fix

Add a dedicated `lockfile` build stage that derives a web+shared-only view of the
workspace lockfile (drop `apps/mobile` from `workspaces`, `npm install
--package-lock-only --offline`). The web build stage consumes that pruned
lockfile via `COPY --from=lockfile`, which BuildKit caches by content digest — so
a mobile-only change produces a byte-identical pruned lockfile and the `npm ci` /
vite build layers stay cached.

Verified locally (node:24-alpine, `--network none`): pruned lockfile is
byte-identical with vs without the #382 mobile change, and `npm ci` against the
pruned lockfile resolves web+shared cleanly.

Scope: selfhost Dockerfile only. `apps/web/Dockerfile` has the identical pattern
and the same bug — flagged, not touched (out of scope for this task).

## Plan

- [x] Add `lockfile` stage to `infrastructure/docker/selfhost/Dockerfile`
- [x] Point `web-builder` at the pruned lockfile
- [x] Build the image locally and confirm it succeeds
- [x] Confirm the mobile-only-change cache-hit behaviour with a local buildx run

## Review

**Change:** `infrastructure/docker/selfhost/Dockerfile` only.

- New `lockfile` stage (`node:24-alpine`): copies the root manifests + web/shared
  `package.json`, drops `apps/mobile` from `workspaces`, runs
  `npm install --package-lock-only --offline` to regenerate a web+shared-only
  `package-lock.json`. npm 11 (node 24) fully prunes the orphaned workspace entry
  and its exclusive deps; npm 10 (node 22) leaves the workspace's own dependency
  list behind, so the pruner stage pins node 24.
- `web-builder` now does `COPY --from=lockfile /app/package.json /app/package-lock.json ./`
  and plain `npm ci` (was `COPY package.json package-lock.json ./` from context +
  `npm ci -w @cairn/web -w @cairn/shared --include-workspace-root`; the `-w` flags
  are redundant once mobile is not a workspace). BuildKit keys `COPY --from` on the
  content digest of the copied files, so an unchanged pruned lockfile = cache hit.

**Verification (local, aarch64):**
- `docker build` of the full Dockerfile succeeds; image has `/app/cairn-selfhost`
  (version injected) + `/app/web` static assets; `tsc --noEmit && vite build` pass
  inside the build.
- Pruned-lockfile determinism: regenerated the pruned lockfile from `HEAD` and
  from `d2f92e3` (pre-#382, before `expo-sqlite`/`await-lock` were added to
  mobile) under `node:24-alpine --network none` — byte-identical.
- Cache behaviour: after a clean build, injected a synthetic mobile-only entry
  into root `package-lock.json` + `apps/mobile/package.json` and rebuilt. The
  `lockfile` stage re-ran (~2s) but `COPY --from=lockfile`, `npm ci`, and
  `npm run build -w @cairn/web` were all `CACHED`. The Go build stage is
  independent of the lockfile and was unaffected before and after.
- `npm ci` against the pruned lockfile + real web/shared `package.json` resolves
  cleanly (dry-run: 343 packages, no lock/manifest sync error).

**Out of scope (flagged, not touched):**
- `apps/web/Dockerfile` has the identical `COPY package-lock.json` + `npm ci`
  pattern and the same cache-bust bug.
- `.github/workflows/docker-build-selfhost.yml` / `selfhost-compose-smoke.yml`
  still list `package-lock.json` in their `paths:` filters, so a mobile-only PR
  still *triggers* these workflows — but the build is now a near-full cache hit
  instead of a from-scratch rebuild.



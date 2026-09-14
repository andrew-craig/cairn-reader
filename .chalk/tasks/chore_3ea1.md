---
id: chore_3ea1
title: CI paths: filters over-trigger web/selfhost builds on mobile-only lockfile changes
type: chore
status: open
priority: 4
labels: [docker,ci]
blocked_by: [chore_88f9]
parent: null
remote_task_url: null
created_at: 2026-09-14T12:20:24Z
updated_at: 2026-09-14T12:20:24Z
---
**Source:** flagged as out-of-scope by chore_6449 (merged as #384). Filed 2026-09-14 during a backlog sweep and re-verified against `main` at `ba7ac84`.

## Problem
A mobile-only dependency change touches the shared root `package-lock.json`, which appears in
the `paths:` filter of every web and selfhost workflow — so a pure-mobile PR triggers all of
them:

| Filter | Location |
|---|---|
| `docker-build-selfhost.yml` | `:9-18` (push), `:21-30` (pull_request) |
| `selfhost-compose-smoke.yml` | `:7-16` (pull_request), `:19-28` (push) |
| `docker-test.yml` `changes` job — `web` | `:51-55` |
| `docker-test.yml` `changes` job — `selfhost` | `:56-64` |
| `docker-build-web.yml` | `:9-14`, `:16-23` — moot, its job is `if: false` (`:32`) |

## Severity is lower than it looks, and drops further once chore_88f9 lands
This is **triggering** waste, not rebuild waste, and #384 already removed most of the cost:
- Selfhost builds now cache-hit through the pruned lockfile, so those runs are near-no-ops.
- `docker-test.yml`'s `build-web` still rebuilds from scratch — but that is **chore_88f9**, not
  this task, and fixing it there is the higher-value change.

Once chore_88f9 lands, every workflow this filter triggers is a cache hit, and what remains is
runner startup, checkout and image pull on a PR that could not have affected the output. Hence
P4 and the `blocked_by`.

## The obvious fix is wrong — do not just delete the line
`package-lock.json` is a **genuine** input to the web and selfhost builds: a real web or shared
dependency bump shows up *only* in the root lockfile, never in `apps/web/package.json` alone.
Removing it from the filters would stop triggering builds that genuinely need to run, trading a
cheap false positive for an expensive false negative — a broken web image merging green.

GitHub `paths:` filters match filenames, not content, so they cannot express "the lockfile
changed in a way that affects web". Any real fix needs a content check *inside* the job — e.g. a
first step that regenerates the pruned lockfile (chore_88f9's `lockfile` stage does exactly this)
and exits early if it is unchanged from the base commit's.

## What to do
**First decide whether this is worth fixing at all**, then either:
- **Close as won't-fix** if, after chore_88f9, the residual cost is just runner startup on
  mobile-only PRs. Record the measurement in the closure note so the next person does not
  re-litigate it. This is the likely outcome and an acceptable one.
- **Or** add the in-job pruned-lockfile comparison with an early exit, and keep the `paths:`
  filters as the cheap first pass.

Reuse chore_88f9's lockfile-pruning step rather than writing a second implementation — three
copies of that `node -e` workspace filter would be a worse outcome than the over-triggering.

## Done when
- [ ] A decision is recorded (fix or won't-fix), with the post-chore_88f9 cost of a mobile-only PR measured, not estimated
- [ ] If fixed: a mobile-only change skips the web and selfhost jobs, **and** a web/shared dependency bump still runs them — the second half is the one that matters

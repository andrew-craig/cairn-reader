---
id: task_5bda
title: Identify true root cause of 2026-09-20 13:17 self-host Postgres auth incident
type: task
status: open
priority: 2
labels: [infrastructure,investigation]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-09-23T07:36:23Z
updated_at: 2026-09-23T07:36:42Z
---

On 2026-09-20 ~13:17 AEST, the real self-hosted Cairn deployment on this server hit `password authentication failed for user cairn` against its own Postgres, with the app container crash-looping. `task_9def`/PR #406 originally claimed this was caused by GitHub Actions' `selfhost-compose-smoke` CI job colliding with the real deployment's Docker volume via a shared Compose project name. That claim is disproven: the job runs on `runs-on: ubuntu-latest`, a GitHub-hosted ephemeral VM that cannot share a Docker daemon/volume namespace with this physical server.

Also ruled out: SSH-driven manual commands (no session active at 13:17 AEST per `last -F`), cron/systemd auto-start of the `cairn-selfhost` stack (no such unit exists), and local Claude Code session transcripts from that day (no `docker compose up`/`down` at 13:17 AEST / 03:17 UTC).

**The true root cause is still unknown.** Not urgent to chase down proactively, but worth deeper forensics (e.g. Postgres/container logs around that timestamp, checking for other processes/automation with Docker access on this host) or at minimum monitoring for recurrence, since an unexplained auth failure against a production self-host deployment could recur. See `task_9def` for the full correction record.

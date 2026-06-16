---
id: task_ed30
title: Audit 5: Observability & operational readiness
type: task
status: in_progress
priority: 2
labels: [audit,observability,ops]
blocked_by: []
parent: epic_9c21
remote_task_url: null
created_at: 2026-06-06T05:06:48Z
updated_at: 2026-06-16T10:23:34Z
---
Audit ability to OPERATE the system in public beta. pkg/logging, pkg/config, pkg/env, health handlers, infrastructure/docker/{prod,selfhost}, .github/workflows.

Findings to confirm/resolve:
1. CRITICAL gap: no metrics. No /metrics endpoint, no Prometheus, no Grafana. Cannot observe latency/error rates/saturation in prod. Define minimal metrics (HTTP latency+count by route/status, DB query time, worker job duration, pool in-use) and a dashboard+alert plan. Cross-ref existing tasks task_644a/task_652a/task_8392.
2. No distributed tracing / request-trace propagation across services (X-Request-ID exists for logs — assess if enough for beta).
3. Container resource limits absent in prod compose (no cpu/mem) — cross-ref task_48ea. Risk of noisy-neighbour OOM.
4. CI runs unit tests/lint/build per-service but NO integration tests against a real Postgres/full stack — cross-ref task_6f3a. Breaking inter-service changes can merge undetected (ties to Audit 1 contract drift).
5. Confirm graceful shutdown (present, 30s) and HTTP read/write timeouts are actually set in prod.
6. Log aggregation: structured slog JSON exists but no shipping/aggregation stack — cross-ref feature_1a69.

Deliverable: findings section listing observability/ops gaps, minimal beta bar vs nice-to-have, and explicit links to the existing chalk tasks that already cover some of this so we don't duplicate.

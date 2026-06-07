---
id: epic_9c21
title: Pre-beta architecture audit
type: epic
status: open
priority: 1
labels: [audit,pre-beta,architecture]
blocked_by: []
parent: null
remote_task_url: null
created_at: 2026-06-06T05:06:01Z
updated_at: 2026-06-06T05:06:01Z
---
Complete architecture audit before opening Cairn to public beta. Goal: assess the system for efficiency, optimisation, consistency, scalability, and operational readiness while API/contract changes are still cheap to make. The audit is split into workstreams (API contract, data layer, mobile client, security, observability/ops, infra reliability). Each workstream produces findings; a final task consolidates everything into a single proposed-next-steps document (docs/architecture/PRE_BETA_AUDIT.md) that ranks remediation work as beta-blocking vs fast-follow vs roadmap.

Scope grounding (LOC): read ~34.5k, users ~14.7k, explore ~8.3k, mobile ~7.8k, pkg ~5.9k. 6 backend services across users/read/explore + shared pkg, single Postgres with per-service logical DBs, JWT/RS256 via Vault, React Native/Expo mobile client.

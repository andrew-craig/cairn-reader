---
id: task_f839
title: Explore Recommender Phase 7: Article cleanup with 90-day retention
type: task
status: closed
priority: 2
labels: []
blocked_by: []
parent: epic_a3df
remote_task_url: null
created_at: 2026-04-07T11:52:28Z
updated_at: 2026-04-07T11:53:12Z
---

Implemented automatic periodic cleanup of articles older than 90 days. Two-phase deletion: soft delete (sets deleted=true) then hard delete after 30-day grace period. Background job runs every 24 hours with graceful shutdown support. Configurable via ARTICLE_RETENTION_DAYS env var. Originally documented in `services/explore/recommender/migrations/PHASE_7_SUMMARY.md`.

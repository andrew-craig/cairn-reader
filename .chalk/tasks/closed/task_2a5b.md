---
id: task_2a5b
title: Explore Recommender Phase 2: Article deduplication with ON CONFLICT
type: task
status: closed
priority: 2
labels: []
blocked_by: []
parent: epic_a3df
remote_task_url: null
created_at: 2026-04-07T11:52:27Z
updated_at: 2026-04-07T11:53:12Z
---

Implemented article deduplication using PostgreSQL ON CONFLICT clause. Updated Article model with voting/recommendation fields. Duplicate detection based on article link (UNIQUE constraint). Updates metadata on re-publish while preserving vote counts and deleted status. Completed December 2025. Originally documented in `services/explore/recommender/migrations/PHASE_2_SUMMARY.md`.

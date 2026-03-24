---
id: feature_d8e7
title: Add Vault connection retry with exponential backoff
type: feature
status: open
priority: 3
labels: []
blocked_by: []
parent: null
created_at: 2026-03-21T04:20:37Z
updated_at: 2026-03-21T04:20:37Z
---
Services fail immediately if Vault is unavailable at startup with no retry logic. Implement exponential backoff retry.

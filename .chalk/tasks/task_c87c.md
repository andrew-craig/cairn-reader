---
id: task_c87c
title: Mobile: connectivity awareness and offline banner
type: task
status: open
priority: 2
labels: [mobile,offline]
blocked_by: []
parent: feature_90a5
remote_task_url: null
created_at: 2026-09-05T23:36:10Z
updated_at: 2026-09-05T23:36:10Z
---
Phase 2 of feature_90a5. Add expo-network, a useNetworkStatus hook and an isOffline() helper. Global offline banner in RootNavigator. withRetry and fetchWithAuth surface a distinguishable NetworkError so callers can fall back to local data instead of showing generic errors.

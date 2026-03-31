---
id: task_f7ae
title: Add Metro bundle verification CI step
type: task
status: open
priority: 2
labels: [ci,mobile]
blocked_by: []
parent: epic_b701
created_at: 2026-03-31T07:57:06Z
updated_at: 2026-03-31T07:57:06Z
---
Add a GitHub Actions job that verifies the JS bundle compiles successfully on PRs touching apps/mobile/**. Run 'npx expo export' or 'npx react-native bundle' to catch broken imports, circular dependencies, and missing assets that TypeScript alone won't detect. This validates the app can actually build beyond just type-checking.

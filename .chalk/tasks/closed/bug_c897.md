---
id: bug_c897
title: Mobile: closing header search takes two taps
type: bug
status: closed
priority: 2
labels: []
blocked_by: []
parent: null
remote_task_url: null
created_at: 2026-11-03T20:34:46Z
updated_at: 2026-11-03T20:37:01Z
---


## Cause
The mobile header search sits in `ArticleListScreen`'s FlatList (`ListHeaderComponent`). The FlatList used the default `keyboardShouldPersistTaps="never"`, so while the keyboard was up the first tap on "Close search" only dismissed the keyboard.

## Plan
- [x] Set `keyboardShouldPersistTaps="handled"` on the FlatList
- [x] Test asserting the prop (fails without the fix)
- [x] Mobile jest suite, tsc, eslint pass

## Review
One-line fix. Side effect: tapping an article row while the search keyboard is up opens it straight away; tapping empty space still dismisses the keyboard.

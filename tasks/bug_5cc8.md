---
id: bug_5cc8
title: tsk ready shows tasks with non-empty blocked_by
type: bug
status: open
priority: 1
labels: []
blocked_by: []
parent: null
created_at: 2026-03-21T04:24:44Z
updated_at: 2026-03-21T04:24:44Z
---
cmd_ready only filters by status==open but doesn't check blocked_by. Tasks with non-empty blocked_by arrays still appear in ready output. Fix: skip tasks where blocked_by is not empty/[].

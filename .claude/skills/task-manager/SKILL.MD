---
name: task-manager
description: File-based task management. Use for ALL task operations — listing, querying, creating, updating, closing, or picking up tasks. Always use this tool (never raw file reads or shell commands) when the user asks about tasks. CLI tool at scripts/tsk.
---

All commands use: `${CLAUDE_SKILL_DIR}/scripts/tsk`

### Quick Reference

```bash
${CLAUDE_SKILL_DIR}/scripts/tsk ready                                  # Find available work (open tasks by priority)
${CLAUDE_SKILL_DIR}/scripts/tsk show <id>                              # View task details
${CLAUDE_SKILL_DIR}/scripts/tsk update <id> --status=in_progress       # Claim work
${CLAUDE_SKILL_DIR}/scripts/tsk close <id>                             # Complete work
${CLAUDE_SKILL_DIR}/scripts/tsk list                                   # List all active tasks
${CLAUDE_SKILL_DIR}/scripts/tsk list --status=open --type=bug          # Filter tasks
```

### Creating Tasks

```bash
${CLAUDE_SKILL_DIR}/scripts/tsk create "Task title"                    # Create with defaults (type=task, priority=2)
${CLAUDE_SKILL_DIR}/scripts/tsk create "Fix login bug" --type=bug --priority=1 --description="Steps to reproduce..."
${CLAUDE_SKILL_DIR}/scripts/tsk create "Sub-task" --parent=task_ed8f   # Link to parent
${CLAUDE_SKILL_DIR}/scripts/tsk create "Blocked task" --blocked_by=bug_a3f2,task_9c1d
```

### Task Fields

| Field | Values |
|---|---|
| `type` | `task` `bug` `feature` `chore` `epic` `decision` |
| `status` | `open` `in_progress` `blocked` `deferred` (closed via `tsk close`) |
| `priority` | `0`=critical `1`=high `2`=medium `3`=low `4`=backlog |
| `labels` | Comma-separated tags |
| `blocked_by` | Comma-separated task IDs |
| `parent` | Parent task ID |

### File Format

Tasks are stored as `tasks/<type>_<id>.md`, for example `tasks/bug_a3f2.md`:

```markdown
---
id: bug_a3f2
title: Fix login error on mobile
type: bug
status: in_progress
priority: 1
labels: [auth, mobile]
blocked_by: []
parent: null
created_at: 2026-03-21T10:00:00Z
updated_at: 2026-03-21T11:30:00Z
---
Optional description or notes go here.
```

### Agent Workflow

```bash
${CLAUDE_SKILL_DIR}/scripts/tsk ready                                  # Find available work
${CLAUDE_SKILL_DIR}/scripts/tsk update <id> --status=in_progress       # Claim a task before starting
# ... do work ...
${CLAUDE_SKILL_DIR}/scripts/tsk close <id>                             # Mark done when complete
```

When discovering new work during a task, create linked sub-tasks:
```bash
${CLAUDE_SKILL_DIR}/scripts/tsk create "Discovered issue" --parent=<current-task-id>
```
---
name: chalk-task-manager
description: "File-based task management. Use for ALL task operations — listing, querying, creating, updating, closing, or picking up tasks. ALWAYS use this skill (never raw file reads via Glob/Read/Grep) to browse or query tasks. When asked to 'pick up a task' or 'find work', start with `chalk ready` (or skill-bundled `chalk ready`). Primary CLI: `.chalk/scripts/chalk`; skill-bundled: scripts/chalk."
---

**IMPORTANT**: Always use `chalk` — never manually read task files with Glob/Read/Grep to browse tasks. Use `chalk ready` as the **first command** when looking for available work.

If chalk is not available, refer to the [installation instructions](task_installation.md) to set it up.

### Quick Reference

```bash
chalk ready                                  # FIRST COMMAND — find available work (unblocked, sorted by priority)
chalk ready --parent=<epic_id>               # Find available work under a specific epic
chalk show <id>                              # View task details
chalk update <id> --status=in_progress       # Claim work
chalk close <id>                             # Complete work
chalk list                                   # List all active tasks
chalk list --status=open --type=bug          # Filter tasks
```

### Creating Tasks

```bash
chalk create "Task title"                    # Create with defaults (type=task, priority=2)
chalk create "Fix login bug" --type=bug --priority=1 --description="Steps to reproduce..."
chalk create "Sub-task" --parent=task_ed8f   # Link to parent
chalk create "Blocked task" --blocked_by=bug_a3f2,task_9c1d
```

### Task Fields

| Field | Values |
|---|---|
| `type` | `task` `bug` `feature` `chore` `epic` `decision` |
| `status` | `open` `in_progress` `blocked` `deferred` (closed via `chalk close`) |
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
chalk ready                                  # Find available work
chalk update <id> --status=in_progress       # Claim a task before starting
# ... do work ...
chalk close <id>                             # Mark done when complete
```

When discovering new work during a task, create linked sub-tasks:
```bash
chalk create "Discovered issue" --parent=<current-task-id>
```
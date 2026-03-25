# Learnings

## 2026-03-25: Always use `tsk` CLI for task operations

**Mistake**: Manually read task files with Glob/Read instead of running `tsk ready` to explore open tasks.

**Rule**: When picking up work or browsing tasks, ALWAYS run `tsk ready` (or `tsk list`, `tsk show`) first. Never use Glob/Read/Grep to browse `.chalk/tasks/` directly. The `tsk` tool gives structured, filtered, priority-sorted output that is faster and more useful than raw file reads.

**Why it matters**: Manual reads waste tokens scanning YAML frontmatter, miss the priority sorting and dependency filtering that `tsk ready` provides, and bypass the intended workflow.

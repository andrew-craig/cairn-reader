---
id: task_4fd9
title: [Audit/Tier 2] Delete the phantom transaction surface DB.WithTransaction (zero callers)
type: task
status: open
priority: 3
labels: [quality,deletion,audit]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-17T10:14:04Z
updated_at: 2026-08-17T10:14:04Z
---
**Source:** Cairn Simplification Audit (read-only pass at HEAD `a6c56a1`, 2026-08-16) — https://claude.ai/code/artifact/286883fb-3f93-49c4-942f-4880251a409f
Read docs/QUALITY_REMEDIATION_STRATEGY.md §0 (rules of engagement) and §2.6 (definition of done) before starting. One finding, one branch, one PR. Re-verify before fixing — all file:line references below were confirmed at `a6c56a1`.

**Audit tier:** 2 (deletions — near-zero risk, compiler-verified).

## Problem
`services/read/content/internal/database/connection.go:114` defines:
```go
func (db *DB) WithTransaction(ctx context.Context, fn func(*sql.Tx) error) error
```
**Zero callers** — `grep -rn 'WithTransaction' --include=*.go .` matches only the definition. The one place in the service that actually uses a transaction bypasses it and drives `BeginTx` directly: `services/read/content/internal/repository/user_content.go:667` (`tx, err := r.db.BeginTx(ctx, nil)`).

So the codebase presents a transaction-management helper that manages no transactions, while the real transaction is hand-rolled elsewhere.

## What to do
Pick one and say which in the PR — **do not leave both**:
- **(a) Delete `WithTransaction`.** Smallest diff, matches 'remove obsolete paths instead of keeping unused abstractions'.
- **(b) Keep it and repoint `user_content.go:667` onto it**, if the hand-rolled block's commit/rollback handling is in fact what the helper does. Only choose this if it strictly simplifies the call site.

The audit's tier places this in deletions, i.e. (a) is the default; (b) needs a reason.

## Done when
- Exactly one transaction idiom remains in the content service, and `services/read` builds green.

## Scope note — the audit pairs this with 'dead pagination'
The audit lists this item as **'dead pagination and a phantom transaction surface'**. The transaction half is confirmed above. The **pagination half is NOT yet identified**: `pkg/api.WritePaginated` is live (2 callers — `services/read/content/internal/api/handlers/user_content_handler.go:196` and `:748`), and no other pagination helper with zero callers turned up. **Do not guess** — get the specific symbol from the audit author before touching anything pagination-related, or leave it out of this PR entirely.

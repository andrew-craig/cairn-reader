---
id: feature_c5f6
title: Extraction feedback triage CLI (list/export/resolve)
type: feature
status: open
priority: 3
labels: []
blocked_by: []
parent: null
remote_task_url: null
created_at: 2026-11-07T20:31:54Z
updated_at: 2026-11-07T20:45:28Z
---

Spec: `.chalk/tasks/closed/feature_27cd.md` §6 (Triage) and §3 (snapshot retention).

`cmd/feedback` CLI in the Content Service:
- `list`: open reports, oldest first.
- `export <id> <dir>`: writes `raw.html`, `output.html`, `meta.json`. These are temporary; delete them once the fixture is cut down (email content must be stripped of personal content before committing a fixture).
- `resolve <id>`: sets `status = 'resolved'` and NULLs `raw_html` and `cleaned_html` in one statement.

Until the CLI exists, resolve with exactly:

```sql
UPDATE content_feedback
SET status = 'resolved', raw_html = NULL, cleaned_html = NULL, updated_at = NOW()
WHERE id = '<id>';
```

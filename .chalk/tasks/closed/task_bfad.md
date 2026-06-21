---
id: task_bfad
title: Split content list vs detail payload (drop cleaned_html from list/search responses)
type: task
status: closed
priority: 1
labels: []
blocked_by: []
parent: epic_5378
remote_task_url: null
created_at: 2026-06-16T10:31:00Z
updated_at: 2026-06-19T18:33:36Z
---

## Review note (2026-06-19)

A backend-only implementation was prepared during the epic_5378 consolidation but
**pulled from the consolidation PR** because it breaks the mobile reader.

Finding: the mobile app has no content-detail fetch path. `ReadScreen` /
`BookmarksScreen` build the in-memory `Article` from list items via
`ReadService.transformToArticle`, which reads `content.cleaned_html`, and
`ReadArticleDetailScreen` renders that body directly. Dropping `cleaned_html`
from the list/search response therefore leaves every opened article with an
empty body (`apps/mobile/src/services/read.ts:523`). The same applies to the
add-URL flow (`AddArticleScreen` uses the create response's content).

To land safely this needs a **coordinated server + mobile change**:
- add `GET /api/v1/content/{id}` detail fetch in the mobile read service,
- have `ReadArticleDetailScreen` (and the add flow) lazy-load the full body on open,
- consider the offline-cache implication (overlaps audit M-5 / task_5229 — list
  items would no longer carry bodies for offline reading).

Keep the list/search summary projection (the real win: stop pulling up to ~5 MB
`cleaned_html` per item) but only after the mobile detail-fetch ships.


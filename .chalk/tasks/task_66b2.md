---
id: task_66b2
title: Content service: moving an item to Reads bumps added_at (lands at top of Reads)
type: task
status: in_progress
priority: 1
labels: [read,backend]
blocked_by: []
parent: epic_6e4d
remote_task_url: null
created_at: 2026-10-10T08:28:32Z
updated_at: 2026-10-10T09:03:16Z
---
Owner decision 2026-10-10: newly saved content must land at the top of Reads. Reads is ordered by added_at DESC (user_content.go ListByUserWithCursor/SearchWithCursor), and PATCH list currently leaves added_at untouched (task_7df9 review).
- [ ] UserContentRepository.UpdateMetadata: when list is set to 'reads' and the item was in 'feed', also set added_at = now (no-op if already in reads) → repo test + PATCH handler test: moved item is first in the Reads list, and cursor paging stays stable
- [ ] Confirm retention (task_d0f5) is unaffected: it only deletes list='feed' rows
- [ ] openapi.yaml: document that moving to reads re-stamps added_at
- Done when: go test passes in services/read/content, incl. integration (-tags=integration)
Blocks the 'Save to Reads' action in task_f9ad.

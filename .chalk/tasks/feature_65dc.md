---
id: feature_65dc
title: [iOS native 13] Report extraction issue sheet
type: feature
status: open
priority: 3
labels: [ios,ui]
blocked_by: [feature_4e37]
parent: epic_bc48
remote_task_url: null
created_at: 2026-11-14T02:35:53Z
updated_at: 2026-11-14T02:35:53Z
---
## Goal
Let the reader report a badly extracted article.

## Source
`apps/mobile/src/components/ReportIssueModal.tsx` and test; `components/common/HeaderPopover.tsx`.

## Plan
- [ ] `.sheet` containing a `NavigationStack` + `Form`: a multi-select list of the ten reasons (same labels and category values), a comment `TextEditor` limited to 2000 characters with a counter, Cancel and Send in the toolbar.
- [ ] Prefill from `getExtractionFeedback` each time it opens; submit with `submitExtractionFeedback`; inline error on failure; Send disabled with nothing selected and while sending.

## Verify
- View-model tests ported from `ReportIssueModal.test.tsx`: prefill, validation, submit payload, error path.

---
id: bug_20cf
title: [iOS native] Reader: tables have borders and cell padding the Expo app doesn't
type: bug
status: open
priority: 2
labels: [ios,ui]
blocked_by: []
parent: epic_bc48
remote_task_url: null
created_at: 2026-11-15T22:06:46Z
updated_at: 2026-11-15T22:06:46Z
---
## Feedback (owner, TestFlight testing of task 12)
Tables now have borders and padding that weren't visible before.
## Cause
`apps/ios/CairnReader/Resources/Reader/reader.css` L95 adds `th, td { border: 1px solid var(--border); padding: 0.25rem 0.5rem; }`. The RN `tagsStyles` (`apps/mobile/src/components/common/ArticleContent.tsx`) has no table styles, so the Expo app shows tables without borders or padding.
## Change
Remove the cell border and padding to match the Expo app. Keep the `table` rule that lets wide tables scroll sideways (`display: block; max-width: 100%; overflow-x: auto`), since it only stops the page overflowing. Keep `text-align: left` only if the Expo app aligns cells that way.
## Verify
- Owner on device: an article with a table looks like it does in the Expo app, in light and dark.

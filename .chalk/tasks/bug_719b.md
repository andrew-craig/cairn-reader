---
id: bug_719b
title: [iOS native] Reader: link options dialog always appears at the top of the page
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
Tapping a link correctly shows the options (Save to Reads, Open in Browser, Copy Link), but they always appear at the top of the page. If the position is fixed, it should be the middle of the page.
## Cause
`ReaderView.swift` L68: `.confirmationDialog` is attached to the whole web-view content. On iOS 26 the dialog is a popover anchored to the view it's attached to, so it points at the top of the full-screen content.
## Change
- Preferred: anchor the dialog at the tapped link. Attach it to a zero-size anchor view placed at the tap location. Get the location from the web view's last touch (a simultaneous tap gesture recogniser on the `WKWebView`), not from content JavaScript, which stays off.
- Otherwise: attach it to a zero-size view centred in the reader, so it opens in the middle of the page.
Keep `confirmationDialog` (system component); don't build a custom menu.
## Verify
- Owner on device: tap links near the top, middle and bottom of an article, and the dialog opens at the link (or in the centre).

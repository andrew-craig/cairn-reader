---
id: bug_719b
title: [iOS native] Reader: link options dialog always appears at the top of the page
type: bug
status: closed
priority: 2
labels: [ios,ui]
blocked_by: []
parent: epic_bc48
remote_task_url: null
created_at: 2026-11-15T22:06:46Z
updated_at: 2026-11-16T10:52:06Z
---
## Feedback (owner, TestFlight testing of task 12)
Tapping a link correctly shows the options (Save to Reads, Open in Browser, Copy Link), but they always appear at the top of the page. If the position is fixed, it should be the middle of the page.
## Cause
`ReaderView.swift` L68: `.confirmationDialog` is attached to the whole web-view content. On iOS 26 the dialog is a popover anchored to the view it's attached to, so it points at the top of the full-screen content.
## Change
- Preferred: anchor the dialog at the tapped link. Attach it to a zero-size anchor view placed at the tap location. Get the location from the web view's last touch (a simultaneous tap gesture recogniser on the `WKWebView`), not from content JavaScript, which stays off.
- Otherwise: attach it to a zero-size view centred in the reader, so it opens in the middle of the page.
Keep `confirmationDialog` (system component); don't build a custom menu.
- [x] Preferred approach: the dialog is anchored at the tapped link.
## Verify
- Owner on device: tap links near the top, middle and bottom of an article, and the dialog opens at the link (or in the centre).
## Review
- `ArticleWebView.swift`: `TouchDownRecognizer`, a `UIGestureRecognizer` subclass on the `WKWebView` that records `location(in: view)` in `touchesBegan` and fails at once, so it never takes part in the web view's own gestures (no simultaneous-recognition delegate needed). `onLink` now passes that point with the URL from `decidePolicyFor`.
- Why touch-down rather than a tap recogniser's `.ended`: the link's navigation action reaches the delegate only after the finger lifts and a round trip to the web content process, so the touch-down of that tap always comes first; a `UITapGestureRecognizer`'s action fires at touch-up too, and its order against WebKit's own tap handling is not guaranteed. No JavaScript.
- `ReaderView.swift`: the `confirmationDialog` moved from the whole screen to `linkAnchor`, a 1-point clear view in the web view's `.overlay`, placed with `.position(linkPoint)`. The overlay sits inside the same `.ignoresSafeArea()` as the web view, so the web view's coordinates are the overlay's. The dialog is attached before `.position` (which fills the overlay), so it points at the 1-point frame. The title link still opens in the browser.
- `apps/ios/CLAUDE.md` Reader line updated.
- Verified: `xcodebuild test` on the iPhone 17 Pro simulator (iOS 26.5) builds with warnings as errors and the suite passes (apart from the intermittent, unrelated `HTTPClientTests.plusInQueryIsPercentEncoded` issue described in bug_6ed5). No unit test: no pure logic was added. Where the popover lands is the owner's on-device check.

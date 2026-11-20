---
id: task_fe45
title: [iOS native 18] Parity and platform QA pass on device
type: task
status: closed
priority: 1
labels: [ios,qa]
blocked_by: [feature_983c]
parent: epic_bc48
remote_task_url: https://linear.app/andrew-cairn/issue/AND-5/ios-native-18-parity-and-platform-qa-pass-on-device
created_at: 2026-11-14T02:35:53Z
updated_at: 2026-11-20T10:59:22Z
---
## Goal
Confirm the native app can replace the RN app. Agents cannot run a simulator from Linux sessions, so the device checks are the owner's; findings become bug tasks under epic_bc48.

## Checklist
- [ ] Device login restores the existing device account after installing over the RN build (decision 4); email login; create account; upgrade device account; change password; log out clears local data.
- [ ] Custom server URL against a self-host stack.
- [ ] Reads: cold start shows stored rows before the network; paging; search; pull to refresh; swipe archive.
- [ ] Feed: paging; search; swipe Save to Reads; item appears in Reads.
- [ ] Edit ordering (decision 9): favourite offline, go online and unfavourite at once; the article ends unfavourited on the server. Kill the app straight after an offline archive; the archive is still queued on relaunch.
- [ ] Store as source of truth (decision 8): archive an article on the web app, pull to refresh on the phone, the row goes; favourite in the reader, go back, the list row reflects it without a refresh.
- [ ] Session scope (decision 10): log out with edits queued offline, sign in as another account; nothing from the first account appears or is sent.
- [ ] Offline: airplane mode banner; Reads list and prefetched bodies open; favourite, archive, scroll position and Save to Reads queue and replay on reconnect; unfetched article shows "Not available offline"; Feed shows the stale banner.
- [ ] Reader: position restore after relaunch; completion at the end; next article; link dialog; share; report issue.
- [ ] Add link (article, feed, Find feed) and feed-only mode from RSS.
- [ ] Subscriptions: route change affects only new items; unsubscribe; newsletter address copy.
- [ ] Brand fonts render everywhere (list rows, forms, navigation titles, tab labels, reader) with no system-font fallback, including the system bars and the search field at the largest Dynamic Type size (decisions 5 and 7).
- [ ] Dark mode, largest Dynamic Type size, VoiceOver labels on toolbar and swipe actions, iPad layout, landscape on iPad.
- [ ] Every RN test file has a ported counterpart or a written reason it does not apply (component tests for controls that are now system-provided; cases removed by decision_adb1).

## Done when
- [ ] No open P0/P1 bug under the epic and the owner signs off on cutover.

## Review
Migrated to Linear as AND-5 (2026-11-20); the QA pass is tracked there. Closed in chalk so it no longer blocks the cutover (chore_c364), which the owner chose to do ahead of this pass.

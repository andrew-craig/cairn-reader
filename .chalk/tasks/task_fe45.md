---
id: task_fe45
title: [iOS native 18] Parity and platform QA pass on device
type: task
status: open
priority: 1
labels: [ios,qa]
blocked_by: [feature_75bc,feature_4e37,feature_65dc,feature_704f,feature_983c,task_4713]
parent: epic_bc48
remote_task_url: null
created_at: 2026-11-14T02:35:53Z
updated_at: 2026-11-14T02:35:53Z
---
## Goal
Confirm the native app can replace the RN app. Agents cannot run a simulator from Linux sessions, so the device checks are the owner's; findings become bug tasks under epic_bc48.

## Checklist
- [ ] Device login restores the existing device account after installing over the RN build (decision 4); email login; create account; upgrade device account; change password; log out clears local data.
- [ ] Custom server URL against a self-host stack.
- [ ] Reads: cold start shows stored rows before the network; paging; search; pull to refresh; swipe archive.
- [ ] Feed: paging; search; swipe Save to Reads; item appears in Reads.
- [ ] Offline: airplane mode banner; Reads list and prefetched bodies open; favourite, archive, scroll position and Save to Reads queue and replay on reconnect; unfetched article shows "Not available offline"; Feed shows the stale banner.
- [ ] Reader: position restore after relaunch; completion at the end; next article; link dialog; share; report issue.
- [ ] Add link (article, feed, Find feed) and feed-only mode from RSS.
- [ ] Subscriptions: route change affects only new items; unsubscribe; newsletter address copy.
- [ ] Brand fonts render everywhere (list rows, forms, navigation titles, tab labels, reader) with no system-font fallback.
- [ ] Dark mode, largest Dynamic Type size, VoiceOver labels on toolbar and swipe actions, iPad layout, landscape on iPad.
- [ ] Every RN test file has a ported counterpart or a written reason it does not apply (component tests for controls that are now system-provided).

## Done when
- [ ] No open P0/P1 bug under the epic and the owner signs off on cutover.

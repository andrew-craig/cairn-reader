---
id: feature_704f
title: [iOS native 14] Add link / subscribe sheet
type: feature
status: open
priority: 2
labels: [ios,ui]
blocked_by: [feature_33bc]
parent: epic_bc48
remote_task_url: null
created_at: 2026-11-14T02:35:53Z
updated_at: 2026-11-14T02:35:53Z
---
## Goal
Save a URL to Reads or subscribe to a feed, from the Reads toolbar and from the RSS screen.

## Source
`apps/mobile/src/components/AddLinkModal.tsx` (+ test), `screens/AddArticleScreen.tsx`, `components/common/{ListChoice,HeaderPopover}.tsx`.

## Plan
- [ ] One `.sheet` with a `Form`: URL field (`.keyboardType(.URL)`, no autocapitalisation, paste button), debounced `detectURL` with cancellation of superseded runs, result summary (article vs feed), "Find feed" via `discoverFeed` for non-feed pages.
- [ ] When the target is a feed: a Feed/Reads `Picker` (inline style) whose section footer carries the explanation text from `ListChoice`. Default Reads.
- [ ] `mode: .link | .feed`. In `.feed` mode a non-feed URL can only go through Find feed and is never saved as an article.
- [ ] Submit through `addURL` / `addContentToUser`; on success dismiss and refresh the presenting list. Online-only: disabled with a note when offline.
- [ ] `AddArticleScreen` is registered in the RN navigator but check whether anything still navigates to it; if not, do not port it and list it as dead code in the PR.

## Verify
- View-model tests ported from `AddLinkModal.test.tsx`: detection debounce and cancellation, feed mode restriction, list choice sent on subscribe, error states.

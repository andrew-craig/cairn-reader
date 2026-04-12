---
id: task_d23d
title: Mobile: Update AddLinkModal Find Feed and button merge behavior
type: task
status: open
priority: 2
labels: []
blocked_by: []
parent: feature_c95a
remote_task_url: null
created_at: 2026-04-10T08:08:41Z
updated_at: 2026-04-12T22:58:43Z
---
Update AddLinkModal to implement the correct Find Feed and button-merge behavior.

## File: `apps/mobile/src/components/AddLinkModal.tsx`

### Change 1: Button merge when feed detected

When `detectionResult?.type === 'feed'`, hide the "Find feed" button entirely and show only the "Add Feed" primary button (which calls `handleAddPress`). Currently both buttons are always visible.

```
if detectionResult.type === 'feed':
  Show: [ Add Feed ]           ← single primary button, calls handleAddPress
else:
  Show: [ Add ] [ Find feed ]  ← two buttons, current layout
```

### Change 2: Rewrite handleFindFeedPress

Current behavior: forces `type: 'feed'` and immediately subscribes — broken.

New behavior:
1. Call `ReadService.discoverFeed(normalizedUrl)`
2. If feeds found:
   - If exactly one feed: update the URL input with the feed URL (calls `setUrl(feed.url)`) — this triggers re-detection via the existing `useEffect`, which will detect it as a feed and merge the buttons into "Add Feed"
   - If multiple feeds: show an Alert with the list and let user pick one, then update URL input with selection
3. If no feeds found: show error "No RSS feed found for this site"

### Change 3: Loading state for discovery

Add a `discovering` state (separate from `loading`) so the "Find feed" button shows a spinner while probing, without blocking the "Add" button.

### Change 4: Remove dead code

Remove the current `handleFindFeedPress` logic that forces `type: 'feed'` — that path is wrong and should not exist.

## UX Flow Summary

```
User pastes "https://blog.example.com"
  → auto-detect fires → returns type: "page", title: "Example Blog"
  → UI shows: [ Add ] [ Find feed ]

User taps "Find feed"
  → spinner on "Find feed" button
  → backend discovers "https://blog.example.com/feed.xml"
  → URL box updates to "https://blog.example.com/feed.xml"
  → auto-detect fires → returns type: "feed", title: "Example Blog RSS"
  → UI shows: [ Add Feed ]  (merged button)

User taps "Add Feed"
  → subscribes to the feed via existing handleAddPress flow
```

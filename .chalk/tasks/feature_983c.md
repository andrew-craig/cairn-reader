---
id: feature_983c
title: [iOS native 16] Subscriptions: RSS and Newsletters with per-source Feed/Reads routing
type: feature
status: open
priority: 2
labels: [ios,ui]
blocked_by: [feature_b604,feature_704f]
parent: epic_bc48
remote_task_url: null
created_at: 2026-11-14T02:35:53Z
updated_at: 2026-11-14T02:35:53Z
---
## Goal
Manage sources and choose which list each one delivers to.

## Source
`apps/mobile/src/components/SubscriptionListScreen.tsx` (+ test), `screens/{RssScreen,NewslettersScreen}.tsx` (+ `RssScreen.test.tsx`), `components/common/ListToggle.tsx`.

## Plan
- [ ] One `SubscriptionListView` used by both screens: `List` of sources with title and subtitle, and a Feed/Reads `Picker` (`.menu` style) per row calling `setSourceList`; optimistic with rollback and an alert on failure. Section footer states that the change applies to new items only.
- [ ] RSS: swipe-to-delete with a confirmation dialog calling `unsubscribeFromRSSFeed`; toolbar "Add feed" opens the add-link sheet in `.feed` mode and reloads on success.
- [ ] Newsletters: show the user's ingest address (`getOrCreateEmailAddress`) with a copy button and the explanatory text; unsubscribe is not supported for email sources, so offer no delete action there.
- [ ] Empty states with `ContentUnavailableView`; pull to refresh.

## Verify
- View-model tests ported from `SubscriptionListScreen.test.tsx` and `RssScreen.test.tsx`: filter by type, route change payload and rollback, unsubscribe flow, feed-mode sheet.

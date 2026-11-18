---
id: feature_983c
title: [iOS native 16] Subscriptions: RSS and Newsletters with per-source Feed/Reads routing
type: feature
status: in_progress
priority: 2
labels: [ios,ui]
blocked_by: [feature_b604]
parent: epic_bc48
remote_task_url: null
created_at: 2026-11-14T02:35:53Z
updated_at: 2026-11-18T11:58:36Z
---
## Goal
Manage sources and choose which list each one delivers to.

## Source
`apps/mobile/src/components/SubscriptionListScreen.tsx` (+ test), `screens/{RssScreen,NewslettersScreen}.tsx` (+ `RssScreen.test.tsx`), `components/common/ListToggle.tsx`.

## Plan
- [x] One `SubscriptionListView` used by both screens: `List` of sources with title and subtitle, and a Feed/Reads `Picker` (`.menu` style) per row calling `setSourceList`; optimistic with rollback and an alert on failure. Section footer states that the change applies to new items only.
- [x] RSS: swipe-to-delete with a confirmation dialog calling `unsubscribeFromRSSFeed`; toolbar "Add feed" opens the add-link sheet in `.feed` mode and reloads on success.
- [x] Newsletters: show the user's ingest address (`getOrCreateEmailAddress`) with a copy button and the explanatory text; unsubscribe is not supported for email sources, so offer no delete action there.
- [x] Empty states with `ContentUnavailableView`; pull to refresh.

## Verify
- View-model tests ported from `SubscriptionListScreen.test.tsx` and `RssScreen.test.tsx`: filter by type, route change payload and rollback, unsubscribe flow, feed-mode sheet.

## Review
Implemented on `claude/epic-bc48-tasks-95c846` (after tasks 13 and 14). Built and tested locally on the iOS 26 simulator (Xcode 26.6): `xcodebuild test`, 433 test cases passed, 0 failed (14 of them new in `SubscriptionListModelTests`), no warnings.

What changed:
- `Features/Subscriptions/SubscriptionListModel.swift` (new): `@Observable`, `init(read:kind:)` with `kind` `.rss` (every non-email source) or `.newsletters` (email). `load()` lists `listAllSubscriptions` filtered by kind; a failure is the RN "Failed to load subscriptions. Please try again." alert and the empty list. `route(for:)` is RN's `getRoute` (RSS by `feed_id`, email by subscription id, nil for an RSS row without its feed). `setList(_:for:)` ignores the current list, flips the row at once, calls `setSourceList`, and on failure restores the previous list with "Failed to update. Please try again.". `unsubscribe(_:)` (RSS with a feed only) removes the row, calls `unsubscribeFromRSSFeed`, and on failure reinserts it at its index with "Failed to unsubscribe. Please try again.". `subtitle(for:)` is the feed URL or sender address, else the description. `loadEmailAddress()` fetches `getOrCreateEmailAddress` once ("Unable to load address" on failure). `ReadService` already had every endpoint, so it is unchanged.
- `Features/Subscriptions/SubscriptionListView.swift` (new): one screen for both routes. Plain `List`, one section whose rows are a `.menu` `Picker` (label: bold title + secondary subtitle; value Feed/Reads) for routable sources, the label alone otherwise; the footer carries RN's "applies to new items only" caption when any row is routable. RSS rows have a trailing swipe "Unsubscribe" (trash) behind a `.confirmationDialog` with RN's title and message. Toolbar `plus`: "Add feed" presents `AddLinkSheet(read:mode: .feed)`, whose `onAdded` reloads; "Add newsletter" opens a popover (`.presentationCompactAdaptation(.popover)`, RN's `HeaderPopover`) with "Send or forward emails to", the address (selectable) with a copy button (`UIPasteboard`, a checkmark "Copied" for 2 s) and RN's explanatory line. Overlay: `ProgressView` until the first load, then `ContentUnavailableView` "No RSS feeds yet" / "No newsletters yet". Pull to refresh; reloads on every appearance (RN `useFocusEffect`) via `.onAppear` + `Task`, as the other pushed screens.
- `Features/Main/Route.swift`: `.rss` and `.newsletters` show `SubscriptionListView`; the placeholder note is gone.
- `Models/ReadModels.swift`: `UnifiedSubscription.list` is `var`, so the model can flip a row.
- `apps/ios/CLAUDE.md`: Subscriptions bullet; placeholder sentence removed.

Tests (`SubscriptionListModelTests`, stub HTTP server):
- SubscriptionListScreen.test.tsx: each source's list and the routing note; RSS routed by feed id (PUT `.../subscriptions/rss/feed-1/list`, `{list: feed}`) with the row flipped while the request is held; email routed by subscription id; the current list sends nothing; failure rolls back with the alert; an unroutable source offers no choice and sends nothing.
- RssScreen.test.tsx: empty list, then `AddLinkModel(read:mode: .feed)` on the same `ReadService` detects a feed, picks Feed, adds (`url`, `type: feed`, `list: feed`), and the reload (`onAdded`) shows Go Blog after two list requests.
- Beyond jest: filter and subtitles by kind; unsubscribe success (DELETE `.../subscriptions/rss/feed-1`) and failure (reinserted at its index, alert); email sources cannot be unsubscribed; a failed load; the newsletter address loaded once, and its failure text.

Dropped or re-expressed jest cases:
- "offers no toggle for source types the backend cannot route": the app's `SubscriptionType` has no `social` (the backend's is marked future and never returned), so the unroutable case is an RSS row without `rss_data`.
- The accessibility-label/`selected` assertions become assertions on `subscriptions[].list` and `route(for:)`; the Alert spy becomes `errorMessage`.
- RssScreen's taps through the modal become `AddLinkModel` calls plus the `load()` the sheet's `onAdded` runs (the sheet wiring itself is view code).

Deliberate differences from RN:
- Email rows have no delete action at all (RN showed the trash and answered "Unsupported").
- The Newsletters header's no-op search button is not ported.
- The Feed/Reads choice is a `.menu` `Picker` rather than a two-segment pill; the routing caption is the section footer rather than a header line.
- You's counts are not refreshed after subscribe/unsubscribe: RN only refetched on focus after its 30 s TTL, and `YouModel` already does the same.

Risks:
- On device (owner): the menu picker's layout with long titles/URLs; the swipe and confirmation dialog; the newsletter popover on iPhone (compact adaptation) and iPad; copy to pasteboard (iOS may show the paste banner only when reading, not writing).
- A load that lands while a routing change is in flight replaces the list with the server's (RN had the same race).
- The routing test holds the PUT on a `DispatchSemaphore` inside the stub handler (which runs on a global queue) to observe the optimistic state.

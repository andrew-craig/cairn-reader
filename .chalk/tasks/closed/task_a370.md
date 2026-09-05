---
id: task_a370
title: [Audit/Tier 4] Triplicated cursor-list machine — and both mobile copies blank the list on pull-to-refresh
type: task
status: closed
priority: 2
labels: [quality,consolidation,audit,mobile,web]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-17T12:54:15Z
updated_at: 2026-09-05T09:20:13Z
---
**Source:** Cairn Simplification Audit — https://claude.ai/code/artifact/286883fb-3f93-49c4-942f-4880251a409f · located and verified against HEAD `a6c56a1` (the report names these two items by description; file:line below is my own derivation, so re-verify before relying on it).
Read docs/QUALITY_REMEDIATION_STRATEGY.md §0 (rules of engagement) and §2.6 (definition of done) before starting. One finding, one branch, one PR.

**Audit tier:** 4 (frontend) | Two problems, one root cause.

## Problem (a) — the same machine three times
The cursor-paginated list state machine (items + cursor + hasMore + loading + loadingMore + refreshing, reset-vs-append, search overlay) is implemented three times:
- `apps/mobile/src/screens/ReadScreen.tsx` — state :22-31, `loadReadArticles` :36-83, `clearSearch` :142-147, `handleRefresh` :149-156
- `apps/mobile/src/screens/BookmarksScreen.tsx` — state :17-22, `loadBookmarks` :26-61, `clearSearch` :89-95, `handleRefresh` :104+
- `apps/web/src/routes/Read.tsx` — state :16-47, `fetchPage` :49-62, `handleRefresh` :78-83

The web copy keeps its cursor and guards in **refs** (:42-47) because an `IntersectionObserver` closure reads them; the mobile copies use plain state with `[cursor]` dependencies. Same machine, materially different concurrency stories.

## Problem (b) — the mobile copies blank the list on every pull-to-refresh
**The mechanism is not `setArticles([])`** — do not go looking for it. It is:
1. `loadReadArticles(reset = true)` calls `setLoading(true)` (ReadScreen:41; BookmarksScreen equivalent)
2. `handleRefresh` calls `loadReadArticles(true)` (:149-156)
3. `ArticleListScreen.tsx:140` early-returns a full-screen `ActivityIndicator` whenever `loading` is true — **unconditionally, regardless of whether items are already on screen**

So pull-to-refresh replaces the populated list with a spinner until the request returns. `refreshing` is already threaded separately into the `RefreshControl` (:205), so the spinner is redundant as well as destructive.

**Web already solved this deliberately** — `Read.tsx:78-83`:
> `// Keep the current list visible; fetchPage(true) replaces it with the first page once it loads, avoiding an empty-state flash mid-refresh.`

## What to do
1. **Fix (b) first, on its own** — it is a user-visible defect and a small diff: either don't set `loading` when the reset was refresh-triggered, or gate `ArticleListScreen`'s spinner on `articles.length === 0`. Prefer the latter; it fixes every current and future caller.
2. Then consider (a). Extract one hook, and **use web's ref-based shape as the reference** — it is the copy that survived contact with an observer-driven load-more.
3. Land (a) and (b) as separate PRs. They have different risk profiles.

## Done when
- (b): pull-to-refresh on both mobile screens keeps the existing list visible, with the RefreshControl as the only progress affordance.
- (a): one cursor-list implementation, with each screen's genuine differences (search source, cache/stale handling, empty message) as parameters — not a config object with a flag per screen.

## Restraint
If collapsing all three requires more than a couple of parameterisation points, do two and say why the third stayed out. The audit's watchlist explicitly flags proposed hooks carrying four-plus parameterisation points as over-abstraction risk.

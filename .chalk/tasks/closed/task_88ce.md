---
id: task_88ce
title: Visual polish for web app implementation
type: task
status: closed
priority: 1
labels: []
blocked_by: []
parent: null
remote_task_url: null
created_at: 2026-06-14T11:09:34Z
updated_at: 2026-06-14T22:53:57Z
---

1. Re-use the floating quick action bar at the bottom of an article from mobile. Do not add the top bar across the top of the app
2. Remove the up and down vote buttons from the Explore list
3. Restructure the left nav / main columns. 
    - The main content column should have a max width of 700px
    - The left nav should have a max width of 160px
    - The left nav should stay docked next to the main content (i.e. put the padding around both, rather than between them)
4. Remove the indent for the sub-items under You, the visual difference between L1 and L2 is enough
5. Switch all of the app content to Inter. Crimson Pro is only used for 'Cairn'
6. Remove all Refresh buttons
7. Add the icons from the mobile app for Read, Explore and You to the left nav. They should be to the left of the text and sized to be visually similar to the text
8. Remove the Explore / Read / Feeds title from the top of the main content column

---

## Implementation Plan

All changes are in `apps/web/`. No mobile app changes.

### 1. Floating action bar on article readers
**Files**: `ReadArticle.tsx`, `ExploreArticle.tsx`, `ReadArticle.css`, new `FloatingActionBar.tsx` + `FloatingActionBar.css`
- Create a `FloatingActionBar` component: a fixed-position pill at the bottom-center with icon-only buttons, matching mobile's `BottomActionMenu` visual style (blur background, rounded pill, shadow)
- Port SVG icons from mobile icon components (bookmark, archive, return, arrow-down, thumbs-up, thumbs-down) plus open-original and delete/save icons
- **ReadArticle bar**: return (back→/read), bookmark (favorite toggle), archive, next-article, delete, open-original
- **ExploreArticle bar**: return (back→/explore), thumbs-up, thumbs-down, next-article, save-to-reading-list, open-original
- Remove the existing `reader__toolbar` from both readers (the text-button bar at the top)
- Add bottom padding to reader content to account for the floating bar
- [ ] FloatingActionBar component created
- [ ] ReadArticle toolbar replaced
- [ ] ExploreArticle toolbar replaced

### 2. Remove vote buttons from Explore list
**Files**: `Explore.tsx`, `Explore.css`
- Delete `explore-card__votes` div and its two child buttons from `ExploreCard`
- Remove `VoteState` interface, `handleVote`/`handleRemoveVote` callbacks, `votes` state, and related props
- Delete `.explore-card__votes` and `.explore-card__vote-btn*` CSS rules
- [ ] Done

### 3. Restructure sidebar + content layout
**Files**: `AppLayout.css`, `AppLayout.tsx`
- Change `.sidebar` width from 260px to 160px
- Set `.app-content` to `max-width: 700px`
- Wrap sidebar+main in a centered container with `max-width: 860px` (160+700), `margin: 0 auto`, `display: flex` so children stay side-by-side
- Padding around the outer container, not between sidebar and content
- [ ] Done

### 4. Remove sub-item indent under You
**Files**: `AppLayout.css`
- Zero out only `margin-left` (preserve `var(--spacing-xs)` top/bottom vertical spacing) and remove `padding-left` from `.sidebar__subnav`
- Remove `border-left` from `.sidebar__subnav`
- [ ] Done

### 5. Switch headings to Inter
**Files**: `index.css`
- Change `--font-heading` to `'Inter'`
- Heading font-weight → 600 (matching mobile `Inter_600SemiBold`)
- Keep `.sidebar__brand` explicitly on Crimson Pro
- [ ] Done

### 6. Remove Refresh buttons
**Files**: `Explore.tsx`, `Read.tsx` and their CSS
- Delete Refresh `<button>` elements and CSS rules
- [ ] Done

### 7. Add nav icons to sidebar
**Files**: `Sidebar.tsx`, `AppLayout.css`
- Inline SVG icons (book/compass/person) left of text labels, ~18px
- At tablet width: show icons instead of single-letter abbreviations
- [ ] Done

### 8. Remove section titles
**Files**: `Explore.tsx`, `You.tsx` and their CSS, `index.css`
- Visually hide `<h1>` titles using a `.sr-only` utility class (keep in DOM for screen reader accessibility)
- Add `.sr-only` class to `index.css`
- Remove the now-unnecessary visual CSS for the titles
- [ ] Done

## Review (completed 2026-06-14)
All 8 items implemented in `apps/web/` (no mobile changes) and verified against
cairn.seatrain.net with the seeded web test account via Playwright:

1. ✅ Floating action bar — new `FloatingActionBar` component; replaces the top
   toolbar in `ReadArticle` (back/favorite/archive/next/delete/open-original) and
   `ExploreArticle` (back/upvote/downvote/next/save/open-original). Sits above the
   mobile BottomNav.
2. ✅ Vote buttons removed from the Explore list (card + dead state/handlers).
3. ✅ Layout: sidebar 160px, content max 700px, both docked inside a centered
   860px container with padding on the outer edges only.
4. ✅ Sub-item indent removed under You (no margin-left/padding-left/border-left).
5. ✅ Headings switched to Inter; `Cairn` wordmark pinned to Crimson Pro via
   `--font-brand`.
6. ✅ Refresh buttons removed (only Read + Explore had them; others were
   error-state "Try again", left intact).
7. ✅ Mobile Read/Explore/You icons added left of the labels; tablet rail shows
   the icons instead of letter abbreviations.
8. ✅ Redundant section titles hidden with a new `.sr-only` utility (kept in DOM).

Verification: `tsc --noEmit` clean, `vite build` clean, eslint clean (one
pre-existing unrelated warning). Visual smoke test passed at desktop/tablet/mobile.

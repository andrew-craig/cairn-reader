---
id: task_3c49
title: [FE resilience] Error boundaries, mobile list error states, mobile a11y labels, web destructive-action confirmations
type: task
status: open
priority: 2
labels: [quality,wave4,frontend]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-09T06:53:56Z
updated_at: 2026-08-09T06:53:56Z
---
Read docs/QUALITY_REMEDIATION_STRATEGY.md §0 (rules of engagement) and §2.6 (definition of done) before starting. Read the full finding text in docs/CODE_QUALITY_REVIEW.md. One finding, one branch, one PR. Re-verify on main first — cited line numbers are from 2026-07-05 and drift.

**Finding:** Front-end resilience & accessibility (Part 2) | **Wave 4** | **Test level:** component tests per apps/web and apps/mobile conventions
**Touches:** apps/web and apps/mobile — error boundaries, mobile list screens, ReadArticleDetailScreen action bar, web reader destructive actions

## Problem
Web is meaningfully more resilient than mobile (real loading/error/empty triad plus explicit retry on every route). Mobile is weak exactly where it matters:

- **No React error boundary in either app** — one malformed article throws during render and the user gets a blank white screen (web) or a hard crash (mobile).
- **Mobile Read / Explore / Bookmarks / Votes have no error state** — a network failure is indistinguishable from an empty account, and pull-to-refresh is the only retry affordance (itself an accessibility gap).
- **Every icon-only control in the mobile app lacks an `accessibilityLabel`** — the entire reader action bar (Back, Next, Favorite, Archive, Upvote, Downvote, Save) is unlabeled to screen readers. Web sets `aria-label`/`aria-pressed` correctly; mobile has regressed relative to it.
- **Web reader destructive actions** (archive/delete) fire immediately with no confirmation and only `console.error` on failure.

## What to do
Four independent pieces — split into separate PRs if any one grows large:
1. React error boundaries in both apps.
2. Mobile list-screen error states with an explicit retry affordance.
3. `accessibilityLabel` on every icon-only mobile control, starting with the reader action bar.
4. Confirmation on web destructive actions, plus real user-visible error handling instead of `console.error`.

Note: the mobile "Archive = hard delete" bug (P2-C5) was already fixed in PR #298 — do not redo it.

## Done when
- Each piece has a test at the level its app's suite uses; a thrown render error no longer blanks or crashes either app.


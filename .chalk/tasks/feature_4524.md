---
id: feature_4524
title: Save read progress for Feed items on web
type: feature
status: open
priority: 3
labels: [web,feed]
blocked_by: []
parent: null
remote_task_url: null
created_at: 2026-11-11T21:41:11Z
updated_at: 2026-11-11T21:41:11Z
---
## Context
The partially read title indicator (PR #452, shared `isPartiallyRead`) relies on saved scroll progress. Neither client saves progress for Feed items today, so Feed rows never show it:
- Web: `apps/web/src/routes/ReadArticle.tsx` skips mark-reading, completion and scroll saves when `isFeed`.
- Mobile: `ReadArticleDetailScreen.tsx` sets `tracksProgress = article.list === 'reads'`. This is deliberate: Feed items are skimmed and triaged by Save to Reads, and on mobile progress writes would also queue outbox rows.

## Decision needed first
Changing this reverses an intentional product rule. Decide whether Feed should track progress at all, and whether the change is web-only or covers both clients (web-only would make the two clients behave differently).

## Plan (if approved)
- [ ] Confirm the decision above (web-only vs both clients)
- [ ] Web: remove the `isFeed` guards in ReadArticle.tsx for scroll save and the unmount flush (decide separately whether status reading/completed applies)
- [ ] Check Save to Reads carries the saved scroll_position over
- [ ] Tests: ReadArticle saves scroll_position for Feed items
- [ ] Update apps/web/CLAUDE.md (and mobile docs, if mobile changes too)

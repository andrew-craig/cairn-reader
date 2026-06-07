---
id: bug_06d6
title: Mobile: store reading progress as a fraction, not absolute pixels
type: bug
status: open
priority: 3
labels: [mobile,reader]
blocked_by: []
parent: null
remote_task_url: null
created_at: 2026-06-07T11:45:46Z
updated_at: 2026-06-07T11:45:46Z
---
Reading progress in the article reader (apps/mobile/src/screens/ReadArticleDetailScreen.tsx + ArticleContent.tsx) is persisted as an absolute pixel scroll offset (scroll_position). On reopen the saved offset is re-applied verbatim.

Problem: the rendered height of an article can differ between sessions — emails/HTML reflow as images load, font-size settings change, device rotation, or RenderHTML layout differences — so an absolute pixel offset no longer points at the same place in the text. The restore can land slightly off even when working correctly.

Fix direction: persist a fractional progress value (offsetY / contentHeight) instead of (or alongside) the raw pixel offset, and on restore compute target = fraction * currentContentHeight. This is resilient to height changes between sessions.

Context: the restore-timing bug (latching before progressively-rendered content finished growing) was already fixed on branch claude/email-read-progress-fc54s. This task is the separate, follow-up robustness improvement noted there. Touches the mobile client; the backend scroll_position column stays as-is unless we decide to store the fraction server-side too.

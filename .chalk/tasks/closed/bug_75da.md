---
id: bug_75da
title: Web: Decode HTML entities in article titles
type: bug
status: closed
priority: 1
labels: [web,design-review]
blocked_by: []
parent: epic_f54e
remote_task_url: null
created_at: 2026-06-24T22:59:32Z
updated_at: 2026-06-28T22:49:35Z
---
Titles render literal HTML entities (e.g. 'Charlie Kirk&#8217;s legacy' instead of an apostrophe) in the reading list, article reader, search results, and mobile. ArticleRow.tsx and ReadArticle.tsx render {article.title} as plain text, so entities in the data leak through. The article *body* decodes correctly (sanitize + dangerouslySetInnerHTML), which makes broken titles stand out more.

Fix: decode entities once at the data boundary (a decodeEntities() helper, or textarea/DOMParser) before rendering titles. Covers &#8217; &amp; &quot; etc.

Also file upstream: Explore excerpts show missing spaces ('SummaryWe've', 'hood.We show') — likely backend extraction concatenating heading+body without whitespace.

From design review of apps/web. Priority 1 of 5.

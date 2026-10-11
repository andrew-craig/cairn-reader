---
id: task_21bd
title: Align date formatting between mobile and web (use mobile relative format)
type: task
status: open
priority: 3
labels: [mobile,web]
blocked_by: []
parent: null
remote_task_url: null
created_at: 2026-10-11T21:46:03Z
updated_at: 2026-10-11T21:46:03Z
---
Mobile shows relative dates via formatDate (Today / Yesterday / N days|weeks|months|years ago); web uses formatPublishedDate (absolute toLocaleDateString) in ArticleRow and ReadArticle. Port the mobile formatDate approach to web (apps/web/src/utils/helpers.ts) and use it in both web places, with tests. Consider moving the shared helper to apps/shared if both apps can consume it.

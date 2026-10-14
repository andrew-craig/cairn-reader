---
id: feature_714e
title: Web + mobile: re-enable disabled RSS feeds
type: feature
status: open
priority: 3
labels: [web,mobile,rss]
blocked_by: []
parent: null
remote_task_url: null
created_at: 2026-10-14T20:28:44Z
updated_at: 2026-10-14T20:28:44Z
---
product_requirements §2.3 keeps 're-enable or remove disabled feeds' (owner decision 2026-10-15), but neither app offers re-enable. The fetcher has PATCH .../subscription/{id} with enabled:true, but it sits behind the internal API key, so it needs a user-facing route through the content service gateway. The fetcher's EnableFeed handler is currently unused, and PATCH enabled:false is a no-op (see the behaviour-gaps task).

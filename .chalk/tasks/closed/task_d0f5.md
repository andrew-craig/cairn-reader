---
id: task_d0f5
title: Feed retention: delete feed items after 30 days
type: task
status: closed
priority: 2
labels: [read,backend]
blocked_by: []
parent: epic_6e4d
remote_task_url: null
created_at: 2026-10-08T11:32:05Z
updated_at: 2026-10-10T04:10:05Z
---
Extend the existing content cleanup job: delete user_contents where list='feed' AND is_favorite=false AND added_at < now()-30 days. The existing orphan trigger + cleanup reclaims contents rows.
- Done when: cleanup job test covers feed-expired, feed-favorited (kept), reads (kept)

---
id: task_899b
title: Implement SenderService
type: task
status: open
priority: 1
labels: []
blocked_by: []
parent: epic_0c4d
created_at: 2026-03-23T07:17:57Z
updated_at: 2026-03-23T07:17:57Z
---
Implement services/read/email/internal/service/sender_service.go

## Requirements
- Upsert sender on email receipt (create or increment count)
- List senders for a user (paginated, ordered by last_received_at DESC)
- Open by default — no allowlist/blocklist logic needed
- Uses SenderRepository for persistence

## Interface
- UpsertOnReceipt(ctx, userID, senderEmail, senderName, receivedAt) -> (EmailSender, error)
- ListByUser(ctx, userID, limit, offset) -> ([]EmailSender, error)

## Tests
- Unit tests with mocked repository

---
id: task_899b
title: Implement SenderService
type: task
status: closed
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

## Implementation Notes
- SenderService interface with `UpsertOnReceipt` and `ListByUser` methods
- `UpsertOnReceipt` delegates to `SenderRepository.Upsert` (which handles ON CONFLICT upsert logic)
- Empty sender name converted to nil pointer to match model convention
- `ListByUser` is a thin pass-through to the repository (ordering handled at DB level)
- 6 unit tests covering: success, empty name, repo errors, empty results, pagination params

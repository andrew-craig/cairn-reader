---
id: task_ab9a
title: Implement AddressService
type: task
status: open
priority: 1
labels: []
blocked_by: []
parent: epic_0c4d
created_at: 2026-03-23T07:17:54Z
updated_at: 2026-03-23T07:17:54Z
---
Implement services/read/email/internal/service/address_service.go

## Requirements
- Generate unique 8-character lowercase alphanumeric local part (e.g. 'k7m2x9pq')
- Enforce 1 address per user (return existing if already created)
- GetOrCreate pattern: if user already has address, return it; otherwise generate new one
- GetByUserID: return user's address or nil
- Uses AddressRepository for persistence
- Retry on collision (regenerate if 8-char random string collides — unlikely but handle it)

## Interface
- GetOrCreate(ctx, userID) -> (EmailAddress, error)
- GetByUserID(ctx, userID) -> (EmailAddress, error)  
- ResolveRecipient(ctx, recipient string) -> (userID, error) — parse local part from email, look up user

## Tests
- Unit tests with mocked repository
- Test collision retry logic
- Test recipient parsing (extract local part from full email address)

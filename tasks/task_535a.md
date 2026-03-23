---
id: task_535a
title: Implement EmailService
type: task
status: open
priority: 1
labels: []
blocked_by: []
parent: epic_0c4d
created_at: 2026-03-23T07:18:01Z
updated_at: 2026-03-23T07:18:01Z
---
Implement services/read/email/internal/service/email_service.go

## Requirements
- Orchestrate email ingestion workflow
- Resolve recipient email address to user_id via AddressService.ResolveRecipient
- Store raw email to database via RawEmailRepository
- Return 'accepted' or 'rejected' (rejected if recipient unknown)
- Does NOT process the email — that's the worker's job

## Interface
- IngestEmail(ctx, IngestEmailRequest) -> (accepted bool, error)

## Dependencies
- AddressService (for recipient resolution)
- RawEmailRepository (for storage)

## Tests
- Unit tests with mocked dependencies
- Test unknown recipient rejection
- Test successful ingestion stores raw email with correct user_id

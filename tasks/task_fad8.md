---
id: task_fad8
title: Implement EmailProcessorWorker
type: task
status: open
priority: 1
labels: []
blocked_by: [task_8300,task_9276]
parent: epic_0c4d
created_at: 2026-03-23T07:18:54Z
updated_at: 2026-03-23T07:18:54Z
---
Implement services/read/email/internal/worker/email_processor_worker.go

## Requirements
Background worker that processes raw emails from the database.

### Processing loop:
1. Poll for pending raw emails (batch of 20, every 5s — configurable)
2. For each raw email:
   a. Mark as 'processing'
   b. Upsert sender via SenderService
   c. Run EmailCleaner.Clean on html_body
   d. Run ContentExtractor.Extract on cleaned HTML
   e. Build content payload for Content Service
   f. Write to content_outbox table
   g. Mark raw email as 'completed'
3. On failure: increment retry_count, store error, leave as 'pending' (or 'failed' if retries exhausted)

### Content payload structure:
{
  'url': generate synthetic URL (e.g. 'email://{raw_email_id}'),
  'html': sanitized_html,
  'title': subject,
  'author': sender_name or sender_email,
  'source_type': 'email',
  'published_at': received_at
}

### Graceful shutdown:
- Accept context cancellation
- Finish current batch, don't start new one
- Configurable number of worker goroutines (default 3)

## Dependencies
- SenderService, EmailCleaner, ContentExtractor
- RawEmailRepository, OutboxRepository

## Tests
- Unit tests with mocked dependencies
- Test full processing pipeline
- Test error handling and retry
- Test graceful shutdown

---
id: task_845e
title: Implement OutboxWorker and cleanup jobs
type: task
status: closed
priority: 1
labels: []
blocked_by: []
parent: epic_0c4d
created_at: 2026-03-23T07:19:04Z
updated_at: 2026-03-27T18:42:06Z
---
Implement three components:

## 1. OutboxWorker (worker/outbox_worker.go)
- Poll content_outbox for pending entries (batch of 10, every 10s)
- Deliver to Content Service via ContentServiceClient
- On success: mark as 'delivered', store content_service_id
- On failure: increment retry_count, calculate next_retry_at with exponential backoff
- Mark as 'failed' when retry_count >= max_retries
- Graceful shutdown support

## 2. RawEmailCleanupJob (jobs/raw_email_cleanup.go)
- Cron job: daily at 5 AM (configurable)
- Delete completed raw emails older than 7 days (configurable)
- Uses RawEmailRepository.DeleteProcessed

## 3. OutboxCleanupJob (jobs/outbox_cleanup.go)
- Cron job: daily at 6 AM (configurable)
- Delete delivered outbox entries older than 7 days (configurable)
- Uses OutboxRepository.DeleteDelivered

## Dependencies
- ContentServiceClient
- OutboxRepository, RawEmailRepository

## Tests
- Unit tests for outbox worker delivery logic
- Test exponential backoff calculation
- Test cleanup job age threshold

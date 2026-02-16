// Package worker provides background workers for the email ingest service.
package worker

// OutboxWorker delivers processed content to the Content Service.
// Responsibilities:
// - Poll for pending outbox entries
// - Deliver to Content Service via HTTP POST
// - Handle retries with exponential backoff
// - Circuit breaker pattern for resilience
// - Track delivery status

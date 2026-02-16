// Package jobs provides scheduled background jobs for the email ingest service.
package jobs

// OutboxCleanupJob periodically cleans up delivered outbox entries.
// Responsibilities:
// - Run daily at 6 AM
// - Delete delivered outbox entries older than 7 days
// - Free up storage space

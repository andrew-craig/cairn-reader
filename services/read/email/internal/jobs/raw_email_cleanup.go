// Package jobs provides scheduled background jobs for the email ingest service.
package jobs

// RawEmailCleanupJob periodically cleans up processed raw emails.
// Responsibilities:
// - Run daily at 5 AM
// - Delete completed raw emails older than 7 days
// - Free up storage space

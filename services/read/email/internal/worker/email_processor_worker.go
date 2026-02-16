// Package worker provides background workers for the email ingest service.
package worker

// EmailProcessorWorker processes raw emails from the database.
// Responsibilities:
// - Poll for pending raw emails
// - Resolve recipient to user_id
// - Resolve or create sender record
// - Email-specific pre-cleaning
// - Content extraction
// - Create outbox entry
// - Update processing status

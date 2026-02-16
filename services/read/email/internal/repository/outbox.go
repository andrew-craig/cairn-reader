// Package repository provides database access layer for the email ingest service.
package repository

// OutboxRepository provides CRUD operations for content outbox.
// Responsibilities:
// - Create outbox entries
// - Query pending deliveries
// - Update delivery status
// - Track retries and errors

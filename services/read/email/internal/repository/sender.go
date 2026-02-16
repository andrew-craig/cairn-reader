// Package repository provides database access layer for the email ingest service.
package repository

// SenderRepository provides CRUD operations for email senders.
// Responsibilities:
// - Create or update sender records
// - Increment email counts
// - List senders for a user
// - Track last received timestamps

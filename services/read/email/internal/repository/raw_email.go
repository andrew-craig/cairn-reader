// Package repository provides database access layer for the email ingest service.
package repository

// RawEmailRepository provides CRUD operations for raw emails.
// Responsibilities:
// - Create raw email records
// - Query pending emails for processing
// - Update processing status
// - Track retry counts and errors

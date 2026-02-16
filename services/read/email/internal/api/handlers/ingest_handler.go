// Package handlers provides HTTP request handlers for the email ingest service.
package handlers

// IngestHandler handles email ingestion from Cloudflare Email Worker.
// Responsibilities:
// - Receive raw emails via POST /api/v1/source/email/ingest
// - Validate API key authentication
// - Store raw emails to database
// - Return 202 Accepted immediately

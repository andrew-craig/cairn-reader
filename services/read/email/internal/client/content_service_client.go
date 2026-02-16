// Package client provides HTTP clients for external service communication.
package client

// ContentServiceClient handles communication with the Content Service.
// Responsibilities:
// - POST to /api/v1/internal/content/user/bulk
// - Retry logic with exponential backoff
// - Circuit breaker implementation
// - Timeout configuration

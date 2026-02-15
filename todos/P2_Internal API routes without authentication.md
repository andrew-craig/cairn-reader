# Internal API Routes Without Authentication

Issue: services/read/content/internal/api/router.go:104-109:

// Internal API routes - used by internal services (Ingest RSS, etc.)
// These routes do NOT require authentication as they are internal-only
r.Route("/api/v1/internal", func(r chi.Router) {
    r.Post("/content/user/bulk", bulkHandler.BulkAddToUsersInternal)
})

Risk: If internal network is compromised, these endpoints are unprotected.

Recommendation: Use service-to-service authentication:

    Mutual TLS (mTLS)
    API keys validated against Vault
    JWT tokens with service-level claims
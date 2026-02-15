# Error Messages May Leak Information

Issue: Detailed error messages returned to clients (gin_adapter.go:71-89):

"error": "token has expired"
"error": "invalid token signature"
"error": "invalid token issuer"

Recommendation: Return generic "unauthorized" message; log specific errors server-side.
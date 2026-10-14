---
id: bug_fa3d
title: Behaviour gaps found in docs review (need decisions)
type: bug
status: open
priority: 3
labels: []
blocked_by: []
parent: null
remote_task_url: null
created_at: 2026-10-14T20:28:44Z
updated_at: 2026-10-14T20:28:44Z
---
Found in chore_65f1. Each needs a decision: fix the code or narrow the contract.
- Fetcher PATCH subscription enabled:false returns 200 but does nothing ('may need to implement DisableFeed').
- Email worker uses only html_body (email_processor_worker.go ~133), so text-only email becomes empty content. Ingest returns 500 when an email has neither body.
- Users POST /auth/resend-verification returns 200 'verification email sent' but sends nothing; it only stores the token (email_verification_service.go:77-90).
- Users ChangePassword does not revoke refresh tokens (user_service.go:225-243), though DeleteUser does.
- Content BulkCreateContent never fills 'existing' (bulk_handler.go ~98), though the spec documents it.
- Content CreateContent error message says only rss|web, while ValidateSourceType accepts email (content_handler.go:55).
- services/read/content/cmd/worker/main.go:163,177 use log.Printf instead of the structured logger.
- Request log sample lines in LOGGING_STRATEGY show user_id, but ChiRequestLogger doesn't add it.

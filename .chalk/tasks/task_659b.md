---
id: task_659b
title: Audit 4: Security & auth hardening
type: task
status: open
priority: 2
labels: [audit,security]
blocked_by: []
parent: epic_9c21
remote_task_url: null
created_at: 2026-06-06T05:06:39Z
updated_at: 2026-06-06T05:06:39Z
---
Audit auth/authorization/security posture for a PUBLIC-facing service. services/users/internal/{auth,handlers,services,middleware}, pkg/auth, pkg/middleware, pkg/rss/sanitize.

Confirmed-good (verify, don't re-litigate): bcrypt cost 12, RS256 stateless JWT, refresh-token rotation, IDOR prevention via RequireSameUser, parameterized queries, HTML sanitization (bluemonday), security headers, HTTPS enforcement, no hardcoded secrets.

Gaps to assess for beta-blocking status:
1. HIGH: No account lockout / failed-login tracking — only IP-based rate limit (10/min). Distributed brute force bypasses it. Decide lockout + per-user attempt tracking.
2. HIGH: Email validation only checks for '@' — no RFC validation, no email verification flow. Public signup needs this.
3. MED: No password reset / forgot-password flow (no account recovery).
4. MED: Rate limiter is in-memory/per-instance — won't hold across multiple instances (cross-ref Audit 6).
5. LOW: No auth audit logging (logins, refreshes, account changes).
6. LOW: Service requires Vault at startup, no offline key fallback.
7. Consider CAPTCHA/abuse protection on public signup.

Deliverable: findings section with each gap, recommended control, effort estimate, and beta-blocking vs fast-follow. Note: a separate /security-review of the diff can follow remediation.

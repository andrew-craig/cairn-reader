# Security Testing Suite

This directory contains security testing scripts for the Cairn User Service. These scripts are designed to verify security controls and identify potential vulnerabilities.

## Overview

The security testing suite currently contains three bash suites, run by `run_all_tests.sh`:
- `sql_injection_test.sh`: SQL injection patterns against login, registration, mobile registration and refresh inputs
- `auth_bypass_test.sh`: missing, malformed, expired, tampered and `none`-algorithm tokens, plus a few injection and path traversal probes
- `rate_limit_test.sh`: rate limit headers, enforcement, response format, shared limit across auth endpoints, window reset, health endpoints exempt

## Prerequisites

Required tools:
- `curl` (HTTP client)
- `jq` (JSON processor)
- `bash` (shell)
- Running User Service instance

## Running Tests

### Quick Start

```bash
# Set service URL (script default: http://localhost:8080; the dev docker-compose publishes 8082)
export USER_SERVICE_URL="http://localhost:8080"

# Run all security tests (also: make test-security from services/users)
./run_all_tests.sh

# Run a specific test suite
./auth_bypass_test.sh
./rate_limit_test.sh
./sql_injection_test.sh
```

All `/api/v1/*` routes reject plain-HTTP requests with `403` unless they arrive over TLS or carry `X-Forwarded-Proto: https`, and the scripts do not send that header. Point `USER_SERVICE_URL` at an HTTPS endpoint (or a TLS-terminating proxy), otherwise most checks see `403` instead of the expected `400`/`401`/`429`.

### Individual Test Scripts

1. **Authentication Bypass Testing**
   ```bash
   ./auth_bypass_test.sh
   ```
   Tests: Missing tokens, expired tokens, invalid signatures, malformed headers, `none` algorithm

2. **Rate Limiting**
   ```bash
   ./rate_limit_test.sh
   ```
   Tests: Rate limit headers, enforcement, 429 response format, window refill. The script assumes a limit of 10 requests/minute, but the service default is 100 per minute (`RATE_LIMIT_REQUESTS` / `RATE_LIMIT_WINDOW`), so set `RATE_LIMIT_REQUESTS=10` on the service under test

3. **SQL Injection**
   ```bash
   ./sql_injection_test.sh
   ```
   Tests: Various SQL injection patterns in auth inputs

## Environment Variables

```bash
# Target service (default: http://localhost:8080)
USER_SERVICE_URL="http://localhost:8080"

# Optional
VERBOSE=1              # Enable verbose output
CONTINUE_ON_FAIL=0     # Stop on first failed suite (run_all_tests.sh default: 1, continue)
```

## Test Output

Tests print pass/fail for each check to the console and `run_all_tests.sh` ends with a suite summary; the exit code is non-zero if any suite failed. No report files are written.

### Example Output

```
=== Running Authentication Bypass Tests ===
✓ PASS: Missing Authorization header returns 401
✓ PASS: Empty token returns 401
✓ PASS: Malformed Bearer format returns 401
✓ PASS: Expired token returns 401
✓ PASS: Invalid signature returns 401
✓ PASS: Wrong issuer returns 401
✗ FAIL: Token reuse not detected

Tests: 7, Passed: 6, Failed: 1
```

## Expected Results

All tests should **PASS** against an HTTPS endpoint with the rate limit configured as described above.

## Security Test Categories

The scripts above cover input validation (SQL injection, null byte and encoding variants), authentication bypass and token tampering, and rate limiting. The following are not yet covered by any script: XSS, authorization (cross-user access), HTTPS enforcement and HSTS, token reuse/family revocation (covered by the Go integration tests in `../integration`), password strength, and concurrent sessions.

## Creating New Tests

Template for new test script:

```bash
#!/bin/bash

# Test Description: [Brief description]
# Category: [Authentication|Authorization|Input Validation|etc.]

set -e

BASE_URL="${USER_SERVICE_URL:-http://localhost:8080}"
VERBOSE="${VERBOSE:-0}"

# Color output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

pass_count=0
fail_count=0

function test_case() {
    local description="$1"
    local expected_status="$2"
    local actual_status="$3"

    if [ "$expected_status" -eq "$actual_status" ]; then
        echo -e "${GREEN}✓ PASS${NC}: $description"
        ((pass_count++))
    else
        echo -e "${RED}✗ FAIL${NC}: $description (expected $expected_status, got $actual_status)"
        ((fail_count++))
    fi
}

# Test implementation
echo "=== Test Suite Name ==="

# Example test
response=$(curl -s -w "\n%{http_code}" "$BASE_URL/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"email":"test@example.com","password":"test"}')
status=$(echo "$response" | tail -n1)
test_case "Valid login returns 200 or 401" "401" "$status"

# Summary
echo ""
echo "Tests: $((pass_count + fail_count)), Passed: $pass_count, Failed: $fail_count"
[ $fail_count -eq 0 ] && exit 0 || exit 1
```

## Continuous Integration

These scripts are not run in CI (`.github/workflows/` has no security workflow). To run them in a pipeline, start the service (with Vault and PostgreSQL) behind HTTPS and run `./run_all_tests.sh`.

## Interpreting Results

### Critical Failures
Tests that **MUST PASS** in production:
- Authentication bypass tests
- SQL injection tests
- Rate limiting tests

### Non-Critical Failures
Tests that may fail in specific environments:
- Everything, when the target is plain HTTP (see Quick Start)
- Some rate limit tests (timing-sensitive, depend on the configured limit)

### Reporting Security Issues

If you discover a security vulnerability:
1. **DO NOT** create a public GitHub issue
2. Email: security@example.com
3. Include: Test case, expected behavior, actual behavior
4. Expected response: 24 hours

## References

- [OWASP Testing Guide](https://owasp.org/www-project-web-security-testing-guide/)
- [JWT Security Best Practices](https://datatracker.ietf.org/doc/html/rfc8725)
- [API Security Checklist](https://github.com/shieldfy/API-Security-Checklist)

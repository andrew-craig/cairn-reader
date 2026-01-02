# Security Testing Suite

This directory contains security testing scripts for the Cairn User Service. These scripts are designed to verify security controls and identify potential vulnerabilities.

## Overview

The security testing suite covers:
- SQL injection attack simulations
- XSS vulnerability testing
- Authentication bypass attempts
- Authorization testing
- Rate limiting verification
- Token manipulation attempts
- Brute force protection
- HTTPS enforcement
- Concurrent request handling

## Prerequisites

Required tools:
- `curl` (HTTP client)
- `jq` (JSON processor)
- `bash` (shell)
- Running User Service instance

## Running Tests

### Quick Start

```bash
# Set service URL
export USER_SERVICE_URL="http://localhost:8080"

# Run all security tests
./run_all_tests.sh

# Run specific test suite
./auth_bypass_test.sh
./rate_limit_test.sh
./token_manipulation_test.sh
```

### Individual Test Scripts

1. **Authentication Bypass Testing**
   ```bash
   ./auth_bypass_test.sh
   ```
   Tests: Missing tokens, expired tokens, invalid signatures, malformed headers

2. **Authorization Testing**
   ```bash
   ./authorization_test.sh
   ```
   Tests: Cross-user access, missing auth, invalid UUIDs

3. **Rate Limiting**
   ```bash
   ./rate_limit_test.sh
   ```
   Tests: Brute force protection, rate limit headers, window refill

4. **Token Manipulation**
   ```bash
   ./token_manipulation_test.sh
   ```
   Tests: Payload modification, signature tampering, algorithm confusion

5. **SQL Injection**
   ```bash
   ./sql_injection_test.sh
   ```
   Tests: Various SQL injection patterns in all inputs

6. **XSS Testing**
   ```bash
   ./xss_test.sh
   ```
   Tests: Script injection in inputs, HTML rendering checks

7. **HTTPS Enforcement**
   ```bash
   ./https_test.sh
   ```
   Tests: HTTP rejection, redirect behavior

8. **Token Reuse Detection**
   ```bash
   ./token_reuse_test.sh
   ```
   Tests: Refresh token reuse, family revocation

## Environment Variables

```bash
# Required
USER_SERVICE_URL="http://localhost:8080"

# Optional
VERBOSE=1              # Enable verbose output
PARALLEL=1             # Run tests in parallel (default: serial)
CONTINUE_ON_FAIL=0     # Stop on first failure (default: continue)
```

## Test Output

Tests generate:
- Console output (pass/fail for each test)
- `test_results.json` (detailed results)
- `security_report.html` (formatted report)

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

All tests should **PASS** except:
- HTTPS enforcement tests (if running without TLS locally)
- Any tests marked as "Expected to fail in development"

## Security Test Categories

### 1. Input Validation Tests
- SQL injection patterns
- XSS payloads
- Command injection
- Path traversal
- Null byte injection
- Unicode normalization attacks

### 2. Authentication Tests
- Missing credentials
- Invalid credentials
- Expired tokens
- Malformed tokens
- Token reuse
- Concurrent login attempts

### 3. Authorization Tests
- Horizontal privilege escalation (user A accessing user B data)
- Vertical privilege escalation (not applicable - no admin roles yet)
- Missing authorization checks
- Parameter tampering

### 4. Session Management Tests
- Token expiration
- Token rotation
- Concurrent sessions
- Token revocation
- Family revocation on reuse

### 5. Rate Limiting Tests
- Authentication endpoint limits
- Brute force protection
- Rate limit headers
- Window refill behavior
- Per-IP vs per-user limiting

### 6. Cryptography Tests
- Weak password acceptance
- Password complexity requirements
- Token signature validation
- Algorithm confusion attacks
- Signature stripping

### 7. Transport Security Tests
- HTTPS enforcement
- HSTS headers
- TLS version requirements
- Certificate validation

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

These tests can be integrated into CI/CD pipelines:

```yaml
# .github/workflows/security-tests.yml
name: Security Tests
on: [push, pull_request]

jobs:
  security:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Start User Service
        run: docker-compose up -d
      - name: Run Security Tests
        run: |
          cd services/users/test/security
          ./run_all_tests.sh
```

## Interpreting Results

### Critical Failures
Tests that **MUST PASS** in production:
- All authentication bypass tests
- All authorization tests
- SQL injection tests
- XSS tests
- Rate limiting tests
- Token manipulation tests

### Non-Critical Failures
Tests that may fail in specific environments:
- HTTPS enforcement (development without TLS)
- Some rate limit tests (timing-sensitive)

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

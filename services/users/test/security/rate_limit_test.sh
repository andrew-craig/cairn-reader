#!/bin/bash

# Rate Limiting Testing
# Tests rate limit enforcement on authentication endpoints
# Set RATE_LIMIT_REQUESTS to the value the service runs with (same name and default)

set -e

BASE_URL="${USER_SERVICE_URL:-http://localhost:8080}"

# /api/v1 requires HTTPS (pkg/middleware.RequireHTTPS). Send the header a TLS-terminating
# proxy would set, so plain-HTTP runs reach the handlers instead of getting 403.
function curl() {
    command curl -H "X-Forwarded-Proto: https" "$@"
}
VERBOSE="${VERBOSE:-0}"
limit="${RATE_LIMIT_REQUESTS:-100}"

# Color output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

pass_count=0
fail_count=0

function log_verbose() {
    if [ "$VERBOSE" -eq 1 ]; then
        echo "$1"
    fi
}

function test_case() {
    local description="$1"
    local expected_status="$2"
    local actual_status="$3"

    if [ "$expected_status" -eq "$actual_status" ]; then
        echo -e "${GREEN}✓ PASS${NC}: $description"
        pass_count=$((pass_count + 1))
    else
        echo -e "${RED}✗ FAIL${NC}: $description (expected $expected_status, got $actual_status)"
        fail_count=$((fail_count + 1))
    fi
}

echo "========================================"
echo "  Rate Limiting Testing"
echo "========================================"
echo "Target: $BASE_URL"
echo "Limit: $limit requests per window"
echo ""

# Test 1: Rapid requests - should trigger rate limit
burst=$((limit + 5))
echo "Test 1: Rate limit enforcement (sending $burst rapid requests)"
echo "  Requests up to the limit should be processed (400/401)"
echo "  The rest should be rate limited (429)"

success_count=0
rate_limited_count=0

for ((i = 1; i <= burst; i++)); do
    response=$(curl -s -w "\n%{http_code}" "$BASE_URL/api/v1/auth/login" \
        -X POST \
        -H "Content-Type: application/json" \
        -d '{"email":"test@example.com","password":"wrong"}')

    status=$(echo "$response" | tail -n1)

    if [ "$status" -eq 429 ]; then
        rate_limited_count=$((rate_limited_count + 1))
        log_verbose "Request $i: Rate limited (429)"
    elif [ "$status" -eq 401 ] || [ "$status" -eq 400 ]; then
        success_count=$((success_count + 1))
        log_verbose "Request $i: Processed (${status})"
    else
        log_verbose "Request $i: Unexpected status (${status})"
    fi
done

echo "  Results: $success_count processed, $rate_limited_count rate limited"

# At least the 5 requests past the limit must be blocked
if [ $rate_limited_count -ge 5 ]; then
    echo -e "${GREEN}✓ PASS${NC}: Rate limiting enforced ($rate_limited_count requests blocked)"
    pass_count=$((pass_count + 1))
else
    echo -e "${RED}✗ FAIL${NC}: Rate limiting not working ($rate_limited_count requests blocked, expected >=5)"
    fail_count=$((fail_count + 1))
fi

# Test 2: Check 429 response format (the limit is exhausted, so this request is blocked)
echo ""
echo "Test 2: Rate limit response format"
response=$(curl -s -i "$BASE_URL/api/v1/auth/login" \
    -X POST \
    -H "Content-Type: application/json" \
    -d '{"email":"test@example.com","password":"wrong"}')

if echo "$response" | grep -q "retry_after"; then
    echo -e "${GREEN}✓ PASS${NC}: 429 response includes retry_after"
    pass_count=$((pass_count + 1))
    retry_after=$(echo "$response" | grep -o '"retry_after":"[^"]*"' | cut -d'"' -f4)
    log_verbose "Retry after: $retry_after"
else
    echo -e "${RED}✗ FAIL${NC}: 429 response missing retry_after"
    fail_count=$((fail_count + 1))
fi

for header in X-RateLimit-Limit X-RateLimit-Window; do
    if echo "$response" | grep -qi "^$header:"; then
        echo -e "${GREEN}✓ PASS${NC}: 429 response includes $header"
        pass_count=$((pass_count + 1))
    else
        echo -e "${RED}✗ FAIL${NC}: 429 response missing $header"
        fail_count=$((fail_count + 1))
    fi
done

# Test 3: Different endpoints independently rate limited
echo ""
echo "Test 3: Endpoint-specific rate limits"
echo "  Testing /auth/register after exhausting /auth/login limit"

# First, exhaust login endpoint
for ((i = 1; i <= limit + 2; i++)); do
    curl -s -o /dev/null "$BASE_URL/api/v1/auth/login" \
        -X POST \
        -H "Content-Type: application/json" \
        -d '{"email":"test@example.com","password":"wrong"}'
done

# Now test register endpoint - should still work if limits are per-endpoint
# But in this implementation, rate limit is per-IP for all auth endpoints
response=$(curl -s -w "\n%{http_code}" "$BASE_URL/api/v1/auth/register" \
    -X POST \
    -H "Content-Type: application/json" \
    -d '{"email":"newuser@example.com","password":"Password123!"}')
status=$(echo "$response" | tail -n1)

# With shared rate limit, this should also be 429
if [ "$status" -eq 429 ]; then
    echo -e "${GREEN}✓ PASS${NC}: Shared rate limit across auth endpoints (429 on /auth/register)"
    pass_count=$((pass_count + 1))
else
    echo -e "${YELLOW}⚠ INFO${NC}: Register endpoint not rate limited (status: $status)"
    echo "  This may indicate per-endpoint rate limiting instead of shared"
fi

# Test 4: Wait for window to reset
echo ""
echo "Test 4: Rate limit window reset"
echo -e "${YELLOW}INFO${NC}: Waiting 65 seconds for rate limit window to reset..."

# Show countdown
for i in {65..1}; do
    echo -ne "\r  Waiting: $i seconds remaining...  "
    sleep 1
done
echo ""

# Test if requests work again
response=$(curl -s -w "\n%{http_code}" "$BASE_URL/api/v1/auth/login" \
    -X POST \
    -H "Content-Type: application/json" \
    -d '{"email":"test@example.com","password":"wrong"}')
status=$(echo "$response" | tail -n1)

if [ "$status" -eq 401 ] || [ "$status" -eq 400 ]; then
    echo -e "${GREEN}✓ PASS${NC}: Rate limit window reset successfully (request processed)"
    pass_count=$((pass_count + 1))
elif [ "$status" -eq 429 ]; then
    echo -e "${RED}✗ FAIL${NC}: Rate limit still active after window (status: 429)"
    fail_count=$((fail_count + 1))
else
    echo -e "${YELLOW}⚠ WARN${NC}: Unexpected status after window reset: $status"
fi

# Test 5: Verify non-auth endpoints are not rate limited
echo ""
echo "Test 5: Health endpoints not rate limited"
health_limited=0
for i in {1..15}; do
    response=$(curl -s -w "\n%{http_code}" "$BASE_URL/health/live")
    status=$(echo "$response" | tail -n1)

    if [ "$status" -eq 429 ]; then
        health_limited=$((health_limited + 1))
    fi
done

if [ $health_limited -eq 0 ]; then
    echo -e "${GREEN}✓ PASS${NC}: Health endpoints not rate limited (15 requests succeeded)"
    pass_count=$((pass_count + 1))
else
    echo -e "${RED}✗ FAIL${NC}: Health endpoints incorrectly rate limited ($health_limited/15 blocked)"
    fail_count=$((fail_count + 1))
fi

# Summary
echo ""
echo "========================================"
echo "Summary: Tests: $((pass_count + fail_count)), Passed: $pass_count, Failed: $fail_count"
echo "========================================"

if [ $fail_count -eq 0 ]; then
    echo -e "${GREEN}All rate limiting tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed. Review rate limiting implementation.${NC}"
    exit 1
fi

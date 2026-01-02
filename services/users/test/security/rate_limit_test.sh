#!/bin/bash

# Rate Limiting Testing
# Tests rate limit enforcement on authentication endpoints
# Expected: 10 requests/minute per IP on auth endpoints

set -e

BASE_URL="${USER_SERVICE_URL:-http://localhost:8080}"
VERBOSE="${VERBOSE:-0}"

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
        ((pass_count++))
    else
        echo -e "${RED}✗ FAIL${NC}: $description (expected $expected_status, got $actual_status)"
        ((fail_count++))
    fi
}

echo "========================================"
echo "  Rate Limiting Testing"
echo "========================================"
echo "Target: $BASE_URL"
echo "Expected Limit: 10 requests/minute"
echo ""

# Test 1: Verify rate limit headers on first request
echo "Test 1: Rate limit headers present"
response=$(curl -s -i "$BASE_URL/api/v1/auth/login" \
    -X POST \
    -H "Content-Type: application/json" \
    -d '{"email":"test@example.com","password":"password"}')

if echo "$response" | grep -qi "X-RateLimit-Limit"; then
    echo -e "${GREEN}✓ PASS${NC}: X-RateLimit-Limit header present"
    ((pass_count++))
    limit=$(echo "$response" | grep -i "X-RateLimit-Limit" | cut -d' ' -f2 | tr -d '\r')
    log_verbose "Rate limit: $limit"
else
    echo -e "${RED}✗ FAIL${NC}: X-RateLimit-Limit header missing"
    ((fail_count++))
fi

if echo "$response" | grep -qi "X-RateLimit-Window"; then
    echo -e "${GREEN}✓ PASS${NC}: X-RateLimit-Window header present"
    ((pass_count++))
    window=$(echo "$response" | grep -i "X-RateLimit-Window" | cut -d' ' -f2 | tr -d '\r')
    log_verbose "Rate limit window: $window"
else
    echo -e "${RED}✗ FAIL${NC}: X-RateLimit-Window header missing"
    ((fail_count++))
fi

# Test 2: Rapid requests - should trigger rate limit
echo ""
echo "Test 2: Rate limit enforcement (sending 15 rapid requests)"
echo "  Requests 1-10 should succeed (401 - invalid credentials)"
echo "  Requests 11-15 should be rate limited (429)"

success_count=0
rate_limited_count=0

for i in {1..15}; do
    response=$(curl -s -w "\n%{http_code}" "$BASE_URL/api/v1/auth/login" \
        -X POST \
        -H "Content-Type: application/json" \
        -d '{"email":"test@example.com","password":"wrong"}')

    status=$(echo "$response" | tail -n1)

    if [ "$status" -eq 429 ]; then
        ((rate_limited_count++))
        log_verbose "Request $i: Rate limited (429)"
    elif [ "$status" -eq 401 ] || [ "$status" -eq 400 ]; then
        ((success_count++))
        log_verbose "Request $i: Processed (${status})"
    else
        log_verbose "Request $i: Unexpected status (${status})"
    fi
done

echo "  Results: $success_count processed, $rate_limited_count rate limited"

# Should have ~10 successful and ~5 rate limited
if [ $rate_limited_count -ge 3 ]; then
    echo -e "${GREEN}✓ PASS${NC}: Rate limiting enforced ($rate_limited_count requests blocked)"
    ((pass_count++))
else
    echo -e "${RED}✗ FAIL${NC}: Rate limiting not working ($rate_limited_count requests blocked, expected >=3)"
    ((fail_count++))
fi

# Test 3: Check 429 response format
echo ""
echo "Test 3: Rate limit response format"
response=$(curl -s "$BASE_URL/api/v1/auth/login" \
    -X POST \
    -H "Content-Type: application/json" \
    -d '{"email":"test@example.com","password":"wrong"}')

if echo "$response" | grep -q "retry_after"; then
    echo -e "${GREEN}✓ PASS${NC}: 429 response includes retry_after"
    ((pass_count++))
    retry_after=$(echo "$response" | grep -o '"retry_after":[0-9]*' | cut -d':' -f2)
    log_verbose "Retry after: $retry_after seconds"
else
    echo -e "${YELLOW}⚠ WARN${NC}: 429 response may not include retry_after (might not be rate limited yet)"
fi

# Test 4: Different endpoints independently rate limited
echo ""
echo "Test 4: Endpoint-specific rate limits"
echo "  Testing /auth/register after exhausting /auth/login limit"

# First, exhaust login endpoint
for i in {1..12}; do
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
    ((pass_count++))
else
    echo -e "${YELLOW}⚠ INFO${NC}: Register endpoint not rate limited (status: $status)"
    echo "  This may indicate per-endpoint rate limiting instead of shared"
fi

# Test 5: Wait for window to reset
echo ""
echo "Test 5: Rate limit window reset"
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
    ((pass_count++))
elif [ "$status" -eq 429 ]; then
    echo -e "${RED}✗ FAIL${NC}: Rate limit still active after window (status: 429)"
    ((fail_count++))
else
    echo -e "${YELLOW}⚠ WARN${NC}: Unexpected status after window reset: $status"
fi

# Test 6: Verify non-auth endpoints are not rate limited
echo ""
echo "Test 6: Health endpoints not rate limited"
health_limited=0
for i in {1..15}; do
    response=$(curl -s -w "\n%{http_code}" "$BASE_URL/health/live")
    status=$(echo "$response" | tail -n1)

    if [ "$status" -eq 429 ]; then
        ((health_limited++))
    fi
done

if [ $health_limited -eq 0 ]; then
    echo -e "${GREEN}✓ PASS${NC}: Health endpoints not rate limited (15 requests succeeded)"
    ((pass_count++))
else
    echo -e "${RED}✗ FAIL${NC}: Health endpoints incorrectly rate limited ($health_limited/15 blocked)"
    ((fail_count++))
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

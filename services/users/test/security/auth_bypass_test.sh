#!/bin/bash

# Authentication Bypass Testing
# Tests various authentication bypass attempts against the User Service
# All tests should return 401 Unauthorized (authentication required)

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
echo "  Authentication Bypass Testing"
echo "========================================"
echo "Target: $BASE_URL"
echo ""

# Test 1: Missing Authorization header
echo "Test 1: Missing Authorization header"
response=$(curl -s -w "\n%{http_code}" "$BASE_URL/api/v1/auth/logout-all" \
    -X POST \
    -H "Content-Type: application/json")
status=$(echo "$response" | tail -n1)
body=$(echo "$response" | head -n-1)
log_verbose "Response: $body"
test_case "Missing Authorization header returns 401" "401" "$status"

# Test 2: Empty Authorization header
echo "Test 2: Empty Authorization header"
response=$(curl -s -w "\n%{http_code}" "$BASE_URL/api/v1/auth/logout-all" \
    -X POST \
    -H "Content-Type: application/json" \
    -H "Authorization: ")
status=$(echo "$response" | tail -n1)
log_verbose "Response: $(echo "$response" | head -n-1)"
test_case "Empty Authorization header returns 401" "401" "$status"

# Test 3: Malformed Bearer format (no space)
echo "Test 3: Malformed Bearer format (no space)"
response=$(curl -s -w "\n%{http_code}" "$BASE_URL/api/v1/auth/logout-all" \
    -X POST \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearertoken")
status=$(echo "$response" | tail -n1)
log_verbose "Response: $(echo "$response" | head -n-1)"
test_case "Malformed Bearer format returns 401" "401" "$status"

# Test 4: Wrong auth scheme (Basic instead of Bearer)
echo "Test 4: Wrong auth scheme (Basic instead of Bearer)"
response=$(curl -s -w "\n%{http_code}" "$BASE_URL/api/v1/auth/logout-all" \
    -X POST \
    -H "Content-Type: application/json" \
    -H "Authorization: Basic dGVzdDp0ZXN0")
status=$(echo "$response" | tail -n1)
log_verbose "Response: $(echo "$response" | head -n-1)"
test_case "Basic auth instead of Bearer returns 401" "401" "$status"

# Test 5: Invalid JWT token (random string)
echo "Test 5: Invalid JWT token (random string)"
response=$(curl -s -w "\n%{http_code}" "$BASE_URL/api/v1/auth/logout-all" \
    -X POST \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer invalid_token_string")
status=$(echo "$response" | tail -n1)
log_verbose "Response: $(echo "$response" | head -n-1)"
test_case "Invalid JWT token returns 401" "401" "$status"

# Test 6: Malformed JWT (missing signature)
echo "Test 6: Malformed JWT (missing signature)"
response=$(curl -s -w "\n%{http_code}" "$BASE_URL/api/v1/auth/logout-all" \
    -X POST \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIn0")
status=$(echo "$response" | tail -n1)
log_verbose "Response: $(echo "$response" | head -n-1)"
test_case "Malformed JWT (missing signature) returns 401" "401" "$status"

# Test 7: Expired JWT token
echo "Test 7: Expired JWT token"
# This is a token that expired in 2020
EXPIRED_TOKEN="eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiZXhwIjoxNTc3ODM2ODAwfQ.invalid"
response=$(curl -s -w "\n%{http_code}" "$BASE_URL/api/v1/auth/logout-all" \
    -X POST \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $EXPIRED_TOKEN")
status=$(echo "$response" | tail -n1)
log_verbose "Response: $(echo "$response" | head -n-1)"
test_case "Expired JWT token returns 401" "401" "$status"

# Test 8: JWT with tampered payload
echo "Test 8: JWT with tampered payload"
# Valid structure but invalid signature
TAMPERED_TOKEN="eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiYWRtaW4iLCJleHAiOjk5OTk5OTk5OTl9.tampered_signature"
response=$(curl -s -w "\n%{http_code}" "$BASE_URL/api/v1/auth/logout-all" \
    -X POST \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TAMPERED_TOKEN")
status=$(echo "$response" | tail -n1)
log_verbose "Response: $(echo "$response" | head -n-1)"
test_case "Tampered JWT payload returns 401" "401" "$status"

# Test 9: JWT with 'none' algorithm
echo "Test 9: JWT with 'none' algorithm"
# eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0 = {"alg":"none","typ":"JWT"}
# eyJ1c2VyX2lkIjoiYWRtaW4ifQ = {"user_id":"admin"}
NONE_ALG_TOKEN="eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJ1c2VyX2lkIjoiYWRtaW4ifQ."
response=$(curl -s -w "\n%{http_code}" "$BASE_URL/api/v1/auth/logout-all" \
    -X POST \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $NONE_ALG_TOKEN")
status=$(echo "$response" | tail -n1)
log_verbose "Response: $(echo "$response" | head -n-1)"
test_case "JWT with 'none' algorithm returns 401" "401" "$status"

# Test 10: Case sensitivity bypass attempt
echo "Test 10: Case sensitivity bypass (bearer vs Bearer)"
response=$(curl -s -w "\n%{http_code}" "$BASE_URL/api/v1/auth/logout-all" \
    -X POST \
    -H "Content-Type: application/json" \
    -H "Authorization: bearer token")
status=$(echo "$response" | tail -n1)
log_verbose "Response: $(echo "$response" | head -n-1)"
# Should accept 'bearer' (case insensitive) but still reject invalid token
test_case "Lowercase 'bearer' with invalid token returns 401" "401" "$status"

# Test 11: SQL injection in login
echo "Test 11: SQL injection in email field"
response=$(curl -s -w "\n%{http_code}" "$BASE_URL/api/v1/auth/login" \
    -X POST \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"admin' OR '1'='1\",\"password\":\"password\"}")
status=$(echo "$response" | tail -n1)
log_verbose "Response: $(echo "$response" | head -n-1)"
test_case "SQL injection in email returns 400 or 401" "400" "$status"

# Test 12: Path traversal in user ID
echo "Test 12: Path traversal in user ID"
response=$(curl -s -w "\n%{http_code}" "$BASE_URL/api/v1/user/../../../etc/passwd" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer valid_token")
status=$(echo "$response" | tail -n1)
log_verbose "Response: $(echo "$response" | head -n-1)"
# Should return 400 (bad UUID format) or 401 (invalid token)
test_case "Path traversal in user ID returns 400 or 401" "401" "$status"

# Test 13: Account enumeration via timing
echo "Test 13: Account enumeration protection"
echo -e "${YELLOW}INFO${NC}: Testing login timing for existing vs non-existing accounts"
# Time login attempt with non-existent email
start1=$(date +%s%N)
curl -s -o /dev/null "$BASE_URL/api/v1/auth/login" \
    -X POST \
    -H "Content-Type: application/json" \
    -d '{"email":"nonexistent@example.com","password":"password"}'
end1=$(date +%s%N)
time1=$(( (end1 - start1) / 1000000 ))

# Time login attempt with potentially existing email
start2=$(date +%s%N)
curl -s -o /dev/null "$BASE_URL/api/v1/auth/login" \
    -X POST \
    -H "Content-Type: application/json" \
    -d '{"email":"admin@example.com","password":"password"}'
end2=$(date +%s%N)
time2=$(( (end2 - start2) / 1000000 ))

diff=$(( time1 > time2 ? time1 - time2 : time2 - time1 ))
log_verbose "Timing difference: ${diff}ms"

if [ $diff -lt 100 ]; then
    echo -e "${GREEN}✓ PASS${NC}: Constant time login (timing difference: ${diff}ms)"
    ((pass_count++))
else
    echo -e "${YELLOW}⚠ WARN${NC}: Potential timing attack vector (timing difference: ${diff}ms)"
    echo -e "  This may indicate account enumeration vulnerability"
fi

# Summary
echo ""
echo "========================================"
echo "Summary: Tests: $((pass_count + fail_count)), Passed: $pass_count, Failed: $fail_count"
echo "========================================"

if [ $fail_count -eq 0 ]; then
    echo -e "${GREEN}All authentication bypass tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed. Review security implementation.${NC}"
    exit 1
fi

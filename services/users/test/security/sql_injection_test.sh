#!/bin/bash

# SQL Injection Testing
# Tests various SQL injection patterns against all user inputs
# All tests should safely handle injection attempts

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

function test_sql_injection() {
    local description="$1"
    local endpoint="$2"
    local payload="$3"
    local method="${4:-POST}"

    response=$(curl -s -w "\n%{http_code}" -X "$method" "$BASE_URL$endpoint" \
        -H "Content-Type: application/json" \
        -d "$payload")

    status=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n-1)

    # Should return 400 (bad request) or 401 (invalid credentials)
    # Should NOT return 200 (SQL injection successful)
    # Should NOT return 500 (SQL error exposed)
    if [ "$status" -eq 400 ] || [ "$status" -eq 401 ] || [ "$status" -eq 409 ]; then
        echo -e "${GREEN}✓ PASS${NC}: $description (status: $status)"
        ((pass_count++))
    elif [ "$status" -eq 200 ]; then
        echo -e "${RED}✗ FAIL${NC}: $description (SQL injection may have succeeded - status: 200)"
        log_verbose "Response: $body"
        ((fail_count++))
    elif [ "$status" -eq 500 ]; then
        echo -e "${RED}✗ FAIL${NC}: $description (SQL error exposed - status: 500)"
        log_verbose "Response: $body"
        ((fail_count++))
    else
        echo -e "${YELLOW}⚠ WARN${NC}: $description (unexpected status: $status)"
        log_verbose "Response: $body"
    fi
}

echo "========================================"
echo "  SQL Injection Testing"
echo "========================================"
echo "Target: $BASE_URL"
echo ""

# Test 1: Classic SQL injection in email field
echo "Category: Classic SQL Injection"
test_sql_injection \
    "OR 1=1 in email" \
    "/api/v1/auth/login" \
    '{"email":"admin@example.com OR 1=1--","password":"password"}'

test_sql_injection \
    "OR '1'='1 in email" \
    "/api/v1/auth/login" \
    "{\"email\":\"admin' OR '1'='1\",\"password\":\"password\"}"

test_sql_injection \
    "UNION SELECT in email" \
    "/api/v1/auth/login" \
    "{\"email\":\"admin' UNION SELECT * FROM users--\",\"password\":\"password\"}"

# Test 2: Comment-based injection
echo ""
echo "Category: Comment-Based Injection"
test_sql_injection \
    "-- comment in email" \
    "/api/v1/auth/login" \
    '{"email":"admin@example.com--","password":"password"}'

test_sql_injection \
    "/* */ comment in email" \
    "/api/v1/auth/login" \
    '{"email":"admin@example.com/**/OR/**/1=1","password":"password"}'

test_sql_injection \
    "# comment in email" \
    "/api/v1/auth/login" \
    '{"email":"admin@example.com# OR 1=1","password":"password"}'

# Test 3: Time-based blind injection
echo ""
echo "Category: Time-Based Blind Injection"
test_sql_injection \
    "WAITFOR DELAY in email" \
    "/api/v1/auth/login" \
    "{\"email\":\"admin'; WAITFOR DELAY '00:00:05'--\",\"password\":\"password\"}"

test_sql_injection \
    "pg_sleep in email" \
    "/api/v1/auth/login" \
    "{\"email\":\"admin'; SELECT pg_sleep(5)--\",\"password\":\"password\"}"

# Test 4: Stacked queries
echo ""
echo "Category: Stacked Queries"
test_sql_injection \
    "DROP TABLE in email" \
    "/api/v1/auth/login" \
    "{\"email\":\"admin'; DROP TABLE users;--\",\"password\":\"password\"}"

test_sql_injection \
    "Multiple statements in email" \
    "/api/v1/auth/login" \
    "{\"email\":\"admin'; DELETE FROM users WHERE 1=1;--\",\"password\":\"password\"}"

# Test 5: Boolean-based blind injection
echo ""
echo "Category: Boolean-Based Blind Injection"
test_sql_injection \
    "Boolean condition in email" \
    "/api/v1/auth/login" \
    "{\"email\":\"admin' AND 1=1--\",\"password\":\"password\"}"

test_sql_injection \
    "Substring extraction in email" \
    "/api/v1/auth/login" \
    "{\"email\":\"admin' AND SUBSTRING((SELECT password FROM users LIMIT 1),1,1)='a\",\"password\":\"password\"}"

# Test 6: Quote escape attempts
echo ""
echo "Category: Quote Escape Attempts"
test_sql_injection \
    "Backslash escape in email" \
    "/api/v1/auth/login" \
    "{\"email\":\"admin\\\\' OR 1=1--\",\"password\":\"password\"}"

test_sql_injection \
    "Double quotes in email" \
    "/api/v1/auth/login" \
    "{\"email\":\"admin\\\" OR 1=1--\",\"password\":\"password\"}"

test_sql_injection \
    "Unicode escape in email" \
    "/api/v1/auth/login" \
    "{\"email\":\"admin\\u0027 OR 1=1--\",\"password\":\"password\"}"

# Test 7: Password field injection
echo ""
echo "Category: Password Field Injection"
test_sql_injection \
    "OR 1=1 in password" \
    "/api/v1/auth/login" \
    "{\"email\":\"admin@example.com\",\"password\":\"' OR '1'='1\"}"

test_sql_injection \
    "UNION SELECT in password" \
    "/api/v1/auth/login" \
    "{\"email\":\"admin@example.com\",\"password\":\"' UNION SELECT password FROM users--\"}"

# Test 8: Registration endpoint injection
echo ""
echo "Category: Registration Endpoint"
test_sql_injection \
    "SQL injection in registration email" \
    "/api/v1/auth/register" \
    "{\"email\":\"test' OR 1=1--@example.com\",\"password\":\"Password123!\"}"

test_sql_injection \
    "UNION in registration" \
    "/api/v1/auth/register" \
    "{\"email\":\"test@example.com'; DROP TABLE users;--\",\"password\":\"Password123!\"}"

# Test 9: Mobile registration injection
echo ""
echo "Category: Mobile Registration"
test_sql_injection \
    "SQL injection in device ID" \
    "/api/v1/auth/register/mobile" \
    "{\"expo_device_id\":\"device' OR 1=1--\"}"

test_sql_injection \
    "UNION in device ID" \
    "/api/v1/auth/register/mobile" \
    "{\"expo_device_id\":\"device'; SELECT * FROM users--\"}"

# Test 10: Second-order injection (via refresh token)
echo ""
echo "Category: Second-Order Injection"
test_sql_injection \
    "SQL injection in refresh token" \
    "/api/v1/auth/refresh" \
    "{\"refresh_token\":\"token' OR 1=1--\"}"

test_sql_injection \
    "UNION in refresh token" \
    "/api/v1/auth/refresh" \
    "{\"refresh_token\":\"token'; SELECT * FROM refresh_tokens--\"}"

# Test 11: Null byte injection
echo ""
echo "Category: Null Byte Injection"
test_sql_injection \
    "Null byte in email" \
    "/api/v1/auth/login" \
    "{\"email\":\"admin@example.com\\u0000\",\"password\":\"password\"}"

# Test 12: Hex encoding injection
echo ""
echo "Category: Encoded Injection"
test_sql_injection \
    "Hex encoded SQL in email" \
    "/api/v1/auth/login" \
    "{\"email\":\"admin' OR 0x313D31--\",\"password\":\"password\"}"

# Test 13: Stored procedure injection (PostgreSQL specific)
echo ""
echo "Category: Stored Procedure Injection"
test_sql_injection \
    "PostgreSQL function call" \
    "/api/v1/auth/login" \
    "{\"email\":\"admin'; SELECT current_user;--\",\"password\":\"password\"}"

test_sql_injection \
    "Database metadata query" \
    "/api/v1/auth/login" \
    "{\"email\":\"admin'; SELECT * FROM information_schema.tables;--\",\"password\":\"password\"}"

# Test 14: Error-based injection
echo ""
echo "Category: Error-Based Injection"
test_sql_injection \
    "Division by zero" \
    "/api/v1/auth/login" \
    "{\"email\":\"admin' AND 1/0--\",\"password\":\"password\"}"

test_sql_injection \
    "Type conversion error" \
    "/api/v1/auth/login" \
    "{\"email\":\"admin' AND CAST('a' AS INTEGER)--\",\"password\":\"password\"}"

# Test 15: Out-of-band injection (PostgreSQL specific)
echo ""
echo "Category: Out-of-Band Injection"
test_sql_injection \
    "COPY TO FILE attempt" \
    "/api/v1/auth/login" \
    "{\"email\":\"admin'; COPY users TO '/tmp/users.txt';--\",\"password\":\"password\"}"

# Summary
echo ""
echo "========================================"
echo "Summary: Tests: $((pass_count + fail_count)), Passed: $pass_count, Failed: $fail_count"
echo "========================================"

if [ $fail_count -eq 0 ]; then
    echo -e "${GREEN}All SQL injection tests passed!${NC}"
    echo "No SQL injection vulnerabilities detected."
    exit 0
else
    echo -e "${RED}CRITICAL: SQL injection vulnerabilities detected!${NC}"
    echo "Review database query implementation immediately."
    exit 1
fi

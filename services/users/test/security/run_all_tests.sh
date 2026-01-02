#!/bin/bash

# Run All Security Tests
# Executes the complete security testing suite

set -e

BASE_URL="${USER_SERVICE_URL:-http://localhost:8080}"
VERBOSE="${VERBOSE:-0}"
CONTINUE_ON_FAIL="${CONTINUE_ON_FAIL:-1}"

# Color output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

total_suites=0
passed_suites=0
failed_suites=0

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "========================================"
echo "  Cairn User Service Security Testing"
echo "========================================"
echo "Target: $BASE_URL"
echo "Continue on Failure: $CONTINUE_ON_FAIL"
echo "Start Time: $(date)"
echo ""

function run_test_suite() {
    local test_script="$1"
    local test_name="$2"

    ((total_suites++))

    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}Running: $test_name${NC}"
    echo -e "${BLUE}========================================${NC}"

    if [ -x "$SCRIPT_DIR/$test_script" ]; then
        if USER_SERVICE_URL="$BASE_URL" VERBOSE="$VERBOSE" "$SCRIPT_DIR/$test_script"; then
            echo -e "${GREEN}✓ Suite Passed: $test_name${NC}"
            ((passed_suites++))
        else
            echo -e "${RED}✗ Suite Failed: $test_name${NC}"
            ((failed_suites++))

            if [ "$CONTINUE_ON_FAIL" -eq 0 ]; then
                echo -e "${RED}Stopping on first failure${NC}"
                exit 1
            fi
        fi
    else
        echo -e "${RED}✗ Test script not found or not executable: $test_script${NC}"
        ((failed_suites++))
    fi
}

# Check if service is running
echo "Checking service availability..."
if curl -s -f "$BASE_URL/health/live" > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Service is running${NC}"
else
    echo -e "${RED}✗ Service is not responding at $BASE_URL${NC}"
    echo "Please start the User Service before running security tests"
    exit 1
fi

# Run test suites
run_test_suite "sql_injection_test.sh" "SQL Injection Tests"
run_test_suite "auth_bypass_test.sh" "Authentication Bypass Tests"
run_test_suite "rate_limit_test.sh" "Rate Limiting Tests"

# Summary
echo ""
echo "========================================"
echo "  Security Testing Summary"
echo "========================================"
echo "Total Test Suites: $total_suites"
echo -e "${GREEN}Passed: $passed_suites${NC}"
if [ $failed_suites -gt 0 ]; then
    echo -e "${RED}Failed: $failed_suites${NC}"
else
    echo -e "${GREEN}Failed: $failed_suites${NC}"
fi
echo "End Time: $(date)"
echo "========================================"

if [ $failed_suites -eq 0 ]; then
    echo -e "${GREEN}All security test suites passed!${NC}"
    echo "The User Service demonstrates strong security controls."
    exit 0
else
    echo -e "${RED}Some security test suites failed!${NC}"
    echo "Review the failures above and address security issues."
    exit 1
fi

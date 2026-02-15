#!/bin/bash

# Script to run integration tests for Cairn backend services
# This script ensures databases are running and executes integration tests

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== Cairn Integration Test Runner ===${NC}\n"

# Check if docker compose is available
if ! command -v docker compose &> /dev/null; then
    echo -e "${RED}Error: docker compose is not installed${NC}"
    exit 1
fi

# Function to check if database is ready
wait_for_db() {
    local container=$1
    local user=$2
    local max_attempts=30
    local attempt=0

    echo -e "${YELLOW}Waiting for $container to be ready...${NC}"

    while [ $attempt -lt $max_attempts ]; do
        if docker exec $container pg_isready -U $user &> /dev/null; then
            echo -e "${GREEN}$container is ready${NC}"
            return 0
        fi
        attempt=$((attempt + 1))
        sleep 2
    done

    echo -e "${RED}Timeout waiting for $container${NC}"
    return 1
}

# Start databases if not running
echo -e "${YELLOW}Checking database containers...${NC}"
if ! docker ps | grep -q cairn-content-db; then
    echo -e "${YELLOW}Starting content database...${NC}"
    docker compose up -d content-db
fi

if ! docker ps | grep -q cairn-rss-db; then
    echo -e "${YELLOW}Starting RSS database...${NC}"
    docker compose up -d rss-db
fi

# Wait for databases to be ready
wait_for_db "cairn-content-db" "cairn_content" || exit 1
wait_for_db "cairn-rss-db" "cairn_rss" || exit 1

echo ""

# Determine which service to test
SERVICE=${1:-all}

# Run Content Service integration tests
if [ "$SERVICE" = "all" ] || [ "$SERVICE" = "content" ]; then
    echo -e "${GREEN}=== Running Content Service Integration Tests ===${NC}"
    cd services/content-service

    if go test -tags=integration -v -count=1 ./...; then
        echo -e "${GREEN}✓ Content Service integration tests passed${NC}\n"
    else
        echo -e "${RED}✗ Content Service integration tests failed${NC}\n"
        exit 1
    fi

    cd ../..
fi

# Run RSS Fetcher Service integration tests
if [ "$SERVICE" = "all" ] || [ "$SERVICE" = "rss" ]; then
    echo -e "${GREEN}=== Running RSS Fetcher Service Integration Tests ===${NC}"
    cd services/rss-fetcher-service

    if go test -tags=integration -v -count=1 ./...; then
        echo -e "${GREEN}✓ RSS Fetcher Service integration tests passed${NC}\n"
    else
        echo -e "${RED}✗ RSS Fetcher Service integration tests failed${NC}\n"
        exit 1
    fi

    cd ../..
fi

echo -e "${GREEN}=== All Integration Tests Passed ===${NC}"

# Option to generate coverage report
if [ "$2" = "--coverage" ]; then
    echo -e "\n${YELLOW}Generating coverage reports...${NC}"

    cd services/content-service
    go test -tags=integration -coverprofile=integration-coverage.out -covermode=atomic ./...
    go tool cover -html=integration-coverage.out -o integration-coverage.html
    echo -e "${GREEN}Content Service coverage report: services/content-service/integration-coverage.html${NC}"
    cd ../..

    cd services/rss-fetcher-service
    go test -tags=integration -coverprofile=integration-coverage.out -covermode=atomic ./...
    go tool cover -html=integration-coverage.out -o integration-coverage.html
    echo -e "${GREEN}RSS Fetcher Service coverage report: services/rss-fetcher-service/integration-coverage.html${NC}"
    cd ../..
fi

echo -e "\n${GREEN}Done!${NC}"

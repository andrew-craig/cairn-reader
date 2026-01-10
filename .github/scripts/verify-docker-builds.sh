#!/bin/bash
# Verify that all Dockerfiles build successfully
# Run this before pushing to ensure GitHub Actions will succeed

set -e

echo "🔍 Verifying Docker builds for all Cairn services..."
echo ""

# Color codes
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Track results
PASSED=0
FAILED=0
SERVICES=()
RESULTS=()

# Function to build and test a service
build_service() {
    local service=$1
    local dockerfile=$2

    echo "----------------------------------------"
    echo "Building: $service"
    echo "Dockerfile: $dockerfile"
    echo "----------------------------------------"

    if docker build -f "$dockerfile" -t "cairn-test-$service:local" . > /tmp/build-$service.log 2>&1; then
        echo -e "${GREEN}✓ $service built successfully${NC}"
        PASSED=$((PASSED + 1))
        SERVICES+=("$service")
        RESULTS+=("✓")
    else
        echo -e "${RED}✗ $service failed to build${NC}"
        echo "  Check logs: /tmp/build-$service.log"
        tail -n 20 /tmp/build-$service.log
        FAILED=$((FAILED + 1))
        SERVICES+=("$service")
        RESULTS+=("✗")
    fi
    echo ""
}

# Change to repository root
cd "$(git rev-parse --show-toplevel)"

echo "Repository root: $(pwd)"
echo ""

# Build all services
build_service "user-service" "services/users/Dockerfile"
build_service "explore-recommender" "services/explore/recommender/Dockerfile"
build_service "explore-fetcher" "services/explore/fetcher/Dockerfile"
build_service "content-service" "services/read/content/Dockerfile"
build_service "content-worker" "services/read/content/Dockerfile.worker"
build_service "ingest-rss" "services/read/fetcher/Dockerfile"
build_service "ingest-rss-worker" "services/read/fetcher/Dockerfile.worker"

# Print summary
echo "========================================"
echo "Build Summary"
echo "========================================"
echo ""

for i in "${!SERVICES[@]}"; do
    if [ "${RESULTS[$i]}" = "✓" ]; then
        echo -e "${GREEN}${RESULTS[$i]} ${SERVICES[$i]}${NC}"
    else
        echo -e "${RED}${RESULTS[$i]} ${SERVICES[$i]}${NC}"
    fi
done

echo ""
echo "----------------------------------------"
echo -e "Passed: ${GREEN}$PASSED${NC}"
echo -e "Failed: ${RED}$FAILED${NC}"
echo "Total:  $((PASSED + FAILED))"
echo "----------------------------------------"

# Cleanup
echo ""
echo "🧹 Cleaning up test images..."
docker images | grep "cairn-test-" | awk '{print $1":"$2}' | xargs -r docker rmi > /dev/null 2>&1 || true
echo "✓ Cleanup complete"

# Exit with error if any builds failed
if [ $FAILED -gt 0 ]; then
    echo ""
    echo -e "${RED}❌ Some builds failed. Fix errors before pushing.${NC}"
    echo "Check logs in /tmp/build-*.log for details"
    exit 1
else
    echo ""
    echo -e "${GREEN}✅ All builds passed! Ready to push to GitHub.${NC}"
    echo ""
    echo "Next steps:"
    echo "  1. git add .github/ infrastructure/docker/docker-compose.prod.yml .dockerignore"
    echo "  2. git commit -m 'Add GitHub Actions CI/CD for Docker builds'"
    echo "  3. git push origin main"
    echo ""
    echo "Then monitor builds at: https://github.com/$(git remote get-url origin | sed 's/.*github.com[:/]\(.*\)\.git/\1/')/actions"
    exit 0
fi

#!/bin/bash
# Script to create GitHub issues from markdown files
# Requires: gh CLI (GitHub CLI) installed and authenticated

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}GitHub Issues Creation Script${NC}"
echo "========================================"
echo ""

# Check if gh is installed
if ! command -v gh &> /dev/null; then
    echo -e "${RED}Error: GitHub CLI (gh) is not installed${NC}"
    echo ""
    echo "Install instructions:"
    echo "  macOS:   brew install gh"
    echo "  Linux:   See https://github.com/cli/cli/blob/trunk/docs/install_linux.md"
    echo "  Windows: scoop install gh"
    echo ""
    exit 1
fi

# Check if authenticated
if ! gh auth status &> /dev/null; then
    echo -e "${RED}Error: Not authenticated with GitHub${NC}"
    echo ""
    echo "Run: gh auth login"
    echo ""
    exit 1
fi

# Function to create issue from markdown file
create_issue() {
    local file="$1"

    if [[ ! -f "$file" ]]; then
        echo -e "${RED}File not found: $file${NC}"
        return 1
    fi

    # Extract title (first line without leading #)
    local title=$(head -n 1 "$file" | sed 's/^# *//')

    # Extract labels (from "Labels: " line)
    local labels=$(grep "^Labels:" "$file" | sed 's/Labels: *//' | tr -d '\n')

    # Extract body (everything except first line and Labels line)
    local body=$(sed '1d;/^Labels:/d' "$file")

    echo -e "${YELLOW}Creating issue: ${NC}$title"
    echo -e "${YELLOW}Labels: ${NC}$labels"

    # Create the issue
    if gh issue create \
        --title "$title" \
        --label "$labels" \
        --body "$body"; then
        echo -e "${GREEN}✓ Issue created successfully${NC}"
        echo ""
        return 0
    else
        echo -e "${RED}✗ Failed to create issue${NC}"
        echo ""
        return 1
    fi
}

# Main execution
echo "This script will create GitHub issues from the following files:"
echo ""

issue_files=(
    "issue-01-explore-fetcher-errors.md"
    "issue-02-recommender-errors.md"
    "issue-03-mobile-type-safety.md"
    "issue-04-react-hook-deps.md"
    "issue-05-unused-functions.md"
    "issue-06-context-key-safety.md"
    "issue-07-mobile-cleanup.md"
    "issue-08-go-optimizations.md"
)

for file in "${issue_files[@]}"; do
    echo "  - $file"
done

echo ""
read -p "Create all issues? (y/N) " -n 1 -r
echo ""

if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Aborted."
    exit 0
fi

echo ""
echo "Creating issues..."
echo ""

success=0
failed=0

for file in "${issue_files[@]}"; do
    if create_issue "$file"; then
        ((success++))
    else
        ((failed++))
    fi
    sleep 1  # Rate limiting: wait 1 second between requests
done

echo "========================================"
echo -e "${GREEN}Successfully created: $success${NC}"
if [[ $failed -gt 0 ]]; then
    echo -e "${RED}Failed: $failed${NC}"
fi
echo ""
echo "View issues: gh issue list"
echo ""

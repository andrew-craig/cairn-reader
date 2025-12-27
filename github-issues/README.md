# GitHub Issues for Code Review Findings

This directory contains GitHub issue templates created from the code review analysis.

## How to Create Issues

### Option 1: Manual Creation (Recommended)
Copy the content from each `.md` file and create a new GitHub issue with the specified title and labels.

### Option 2: Using GitHub CLI
If you have `gh` CLI installed and authenticated:

```bash
cd github-issues

# Create all issues
for file in issue-*.md; do
    title=$(head -n 1 "$file" | sed 's/^# //')
    labels=$(grep "^Labels:" "$file" | sed 's/Labels: //')
    body=$(sed '1d;/^Labels:/d' "$file")
    gh issue create --title "$title" --label "$labels" --body "$body"
done
```

### Option 3: Using GitHub API
See `create-issues.sh` script for automated creation via GitHub API.

## Issues Overview

| Priority | Issue | Component | Est. Effort |
|----------|-------|-----------|-------------|
| Critical | Fix unchecked errors - Explore Fetcher | Go/Explore | 3-4 hours |
| Critical | Fix unchecked errors - Recommender | Go/Explore | 2-3 hours |
| Medium | Fix type safety issues | Mobile App | 1-2 hours |
| Medium | Fix React Hook dependencies | Mobile App | 30 min |
| Medium | Remove unused functions | Go/Users | 30 min |
| Medium | Fix context key type safety | Go/Users | 30 min |
| Low | Clean up unused code | Mobile App | 1 hour |
| Low | Optimize Go code | Go/Explore | 1 hour |

**Total Estimated Effort:** 9-14 hours

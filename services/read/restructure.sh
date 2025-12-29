#!/bin/bash
set -e

# Read Service Restructuring Script
# Automates the restructuring of read service to match cairn repository conventions

echo "=================================================="
echo "Read Service Restructuring Script"
echo "=================================================="
echo ""

# Verify we're in the correct directory
if [ ! -d "services" ]; then
    echo "ERROR: services/ directory not found"
    echo "Please run this script from services/read/"
    exit 1
fi

if [ ! -d "services/content-service" ] || [ ! -d "services/rss-fetcher-service" ]; then
    echo "ERROR: Expected services not found"
    echo "This script expects:"
    echo "  - services/content-service/"
    echo "  - services/rss-fetcher-service/"
    exit 1
fi

echo "✓ Directory structure validated"
echo ""

# Create backup
BACKUP_DIR="backup-$(date +%Y%m%d-%H%M%S)"
echo "Creating backup in ${BACKUP_DIR}..."
mkdir -p "${BACKUP_DIR}"
cp -r services "${BACKUP_DIR}/"
echo "✓ Backup created: ${BACKUP_DIR}/"
echo ""

# Phase 1: Move directories
echo "Phase 1: Reorganizing directories..."
echo "  - Moving services/content-service/ → content/"
mv services/content-service content

echo "  - Moving services/rss-fetcher-service/ → fetcher/"
mv services/rss-fetcher-service fetcher

echo "  - Removing empty services/ directory"
rmdir services

echo "✓ Phase 1 complete"
echo ""

# Phase 2: Rename commands
echo "Phase 2: Renaming command directories..."

# Check if fetcher has cmd/server
if [ -d "fetcher/cmd/server" ]; then
    echo "  - Renaming fetcher/cmd/server/ → fetcher/cmd/fetcher/"
    mv fetcher/cmd/server fetcher/cmd/fetcher
    echo "✓ Renamed fetcher command directory"
else
    echo "  ⚠ Warning: fetcher/cmd/server/ not found, skipping rename"
fi

# Check if content needs cmd/content
if [ ! -d "content/cmd/content" ] && [ ! -d "content/cmd/server" ]; then
    echo "  ℹ Note: content/cmd/content/ directory not found"
    echo "    You may need to create this manually if there's a server component"
fi

echo "✓ Phase 2 complete"
echo ""

# Phase 3: Create shared directories
echo "Phase 3: Creating shared directories..."
mkdir -p pkg/logging
mkdir -p pkg/models
mkdir -p bin
echo "  - Created pkg/logging/"
echo "  - Created pkg/models/"
echo "  - Created bin/"
echo "✓ Phase 3 complete"
echo ""

# Phase 4: Update import paths
echo "Phase 4: Updating import paths..."
echo "  - Updating imports in content/ ..."

# Count files to update
CONTENT_FILES=$(find content -name "*.go" -type f | wc -l)
FETCHER_FILES=$(find fetcher -name "*.go" -type f | wc -l)

echo "    Found ${CONTENT_FILES} Go files in content/"
find content -name "*.go" -type f -exec sed -i \
  's|github\.com/andrew-craig/cairn-read/services/content-service|github.com/andrew-craig/cairn-read/content|g' {} +

echo "  - Updating imports in fetcher/ ..."
echo "    Found ${FETCHER_FILES} Go files in fetcher/"
find fetcher -name "*.go" -type f -exec sed -i \
  's|github\.com/andrew-craig/cairn-read/services/rss-fetcher-service|github.com/andrew-craig/cairn-read/fetcher|g' {} +

echo "✓ Phase 4 complete"
echo ""

# Phase 5: Backup and remove individual go.mod files
echo "Phase 5: Managing Go module files..."

if [ -f "content/go.mod" ]; then
    echo "  - Backing up content/go.mod → content-go.mod.bak"
    cp content/go.mod content-go.mod.bak
    echo "  - Removing content/go.mod and content/go.sum"
    rm -f content/go.mod content/go.sum
else
    echo "  ⚠ Warning: content/go.mod not found"
fi

if [ -f "fetcher/go.mod" ]; then
    echo "  - Backing up fetcher/go.mod → fetcher-go.mod.bak"
    cp fetcher/go.mod fetcher-go.mod.bak
    echo "  - Removing fetcher/go.mod and fetcher/go.sum"
    rm -f fetcher/go.mod fetcher/go.sum
else
    echo "  ⚠ Warning: fetcher/go.mod not found"
fi

echo "✓ Phase 5 complete"
echo ""

# Summary
echo "=================================================="
echo "Automated Restructuring Complete!"
echo "=================================================="
echo ""
echo "Directory structure updated:"
echo "  services/read/"
echo "  ├── content/              (was services/content-service/)"
echo "  ├── fetcher/              (was services/rss-fetcher-service/)"
echo "  ├── pkg/"
echo "  │   ├── logging/"
echo "  │   └── models/"
echo "  └── bin/"
echo ""
echo "Import paths updated:"
echo "  ✓ ${CONTENT_FILES} files in content/"
echo "  ✓ ${FETCHER_FILES} files in fetcher/"
echo ""
echo "Backups created:"
echo "  - ${BACKUP_DIR}/services/ (full backup)"
if [ -f "content-go.mod.bak" ]; then
    echo "  - content-go.mod.bak"
fi
if [ -f "fetcher-go.mod.bak" ]; then
    echo "  - fetcher-go.mod.bak"
fi
echo ""
echo "=================================================="
echo "NEXT MANUAL STEPS REQUIRED:"
echo "=================================================="
echo ""
echo "1. Create unified go.mod file:"
echo "   • Review content-go.mod.bak and fetcher-go.mod.bak"
echo "   • Merge dependencies into a new services/read/go.mod"
echo "   • Set module name: github.com/andrew-craig/cairn-read"
echo ""
echo "2. Tidy and verify Go modules:"
echo "   cd services/read"
echo "   go mod tidy"
echo "   go mod verify"
echo ""
echo "3. Update docker-compose.yml:"
echo "   • Change build paths from 'services/content-service' to 'content'"
echo "   • Change build paths from 'services/rss-fetcher-service' to 'fetcher'"
echo ""
echo "4. Update Makefile:"
echo "   • Update build targets to use new paths"
echo "   • Update test targets to use new paths"
echo ""
echo "5. Update Dockerfiles:"
echo "   • Update COPY statements to include pkg/"
echo "   • Change paths to new structure"
echo ""
echo "6. Test the changes:"
echo "   make build-all"
echo "   make test-all"
echo "   docker-compose up --build"
echo ""
echo "=================================================="
echo "DOCUMENTATION:"
echo "=================================================="
echo ""
echo "For detailed instructions, see:"
echo "  • RESTRUCTURING_GUIDE.md     - Comprehensive guide (567 lines)"
echo "  • RESTRUCTURING_CHECKLIST.md - Track your progress"
echo "  • RESTRUCTURING_SUMMARY.md   - Quick reference"
echo ""
echo "To rollback:"
echo "  rm -rf content fetcher pkg bin"
echo "  cp -r ${BACKUP_DIR}/services ."
echo ""
echo "Good luck! 🚀"
echo ""

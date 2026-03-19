#!/bin/bash
set -euo pipefail

# Only run in remote (Claude Code on the web) environments
if [ "${CLAUDE_CODE_REMOTE:-}" != "true" ]; then
  exit 0
fi

echo "Setting up bd (beads issue tracker)..."

# Try binary download first, fall back to npm
if ! command -v bd &> /dev/null; then
    # Preferred: download pre-built binary (works in Claude Code Web)
    echo "Trying binary download..."
    BD_PATH="${HOME}/.local/bin/bd"
    mkdir -p "$(dirname "$BD_PATH")"
    if curl -fsSL https://raw.githubusercontent.com/btucker/bd-binaries/main/linux_amd64/bd -o "$BD_PATH" 2>/dev/null; then
        chmod +x "$BD_PATH"
        export PATH="${HOME}/.local/bin:$PATH"
        echo "Installed via binary download"
    elif npm install -g @beads/bd --quiet 2>/dev/null && command -v bd &> /dev/null; then
        echo "Installed via npm"
    else
        echo "Warning: failed to install bd"
    fi
fi

# Persist PATH so bd is available throughout the session
if [ -n "${CLAUDE_ENV_FILE:-}" ]; then
    echo "export PATH=\"${HOME}/.local/bin:\$PATH\"" >> "$CLAUDE_ENV_FILE"
fi

# Resolve project directory early (used by dolt setup and npm install)
PROJECT_DIR="${CLAUDE_PROJECT_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"

# Install dolt if not present (required by bd)
if ! command -v dolt &> /dev/null; then
    echo "Installing dolt..."
    if curl -fsSL https://github.com/dolthub/dolt/releases/latest/download/install.sh | bash 2>/dev/null; then
        echo "dolt installed"
    else
        echo "Warning: failed to install dolt"
    fi
fi

# Set up Dolt database from remote
DOLT_REMOTE="http://github.com/cairn-app/cairn-reader.git"
DOLT_DATA_DIR="${PROJECT_DIR}/.beads/dolt/beads_orcas"

echo "Setting up Dolt database..."
if [ ! -d "$DOLT_DATA_DIR" ]; then
    echo "Cloning Dolt database from remote..."
    dolt clone "$DOLT_REMOTE" "$DOLT_DATA_DIR"
else
    echo "Dolt database directory exists, ensuring remote is registered..."
    bd dolt remote add origin "$DOLT_REMOTE" 2>/dev/null || true
fi

bd dolt start || true
bd dolt pull || echo "Warning: dolt pull failed (will continue)"

# Verify and show version
bd version

echo "Installing npm dependencies..."
npm install --prefix "$PROJECT_DIR"
echo "npm install complete"

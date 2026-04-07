#!/bin/bash
set -euo pipefail

# Only run in remote Claude environments (e.g. Claude Code Web).
# CLAUDE_ENV_FILE is set by the remote harness; if absent we're running locally
# where chalk is already installed globally — nothing to do.
if [ -z "${CLAUDE_ENV_FILE:-}" ]; then
  exit 0
fi

curl -fsSL https://raw.githubusercontent.com/andrew-craig/chalk/main/install.sh | bash

echo "export PATH=\"${CHALK_INSTALL_DIR:-$HOME/.local/bin}:\$PATH\"" >> "$CLAUDE_ENV_FILE"

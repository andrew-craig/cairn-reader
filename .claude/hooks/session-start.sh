#!/bin/bash
set -euo pipefail

# Add tsk script to PATH for this session
if [ -n "${CLAUDE_ENV_FILE:-}" ]; then
  echo "export PATH=\"\$CLAUDE_PROJECT_DIR/.claude/skills/task-manager/scripts:\$PATH\"" >> "$CLAUDE_ENV_FILE"
fi

#!/bin/bash
set -euo pipefail

# Derive paths from script location, independent of env vars
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
TSK_DIR="$PROJECT_DIR/.claude/skills/task-manager/scripts"

# Add tsk script to PATH for this session
if [ -n "${CLAUDE_ENV_FILE:-}" ]; then
  # Harness supports env file injection — write an absolute path (not $CLAUDE_PROJECT_DIR)
  echo "export PATH=\"$TSK_DIR:\$PATH\"" >> "$CLAUDE_ENV_FILE"
else
  # Fallback: symlink into the global skills dir the harness already puts in PATH
  GLOBAL_SKILLS_DIR="/.claude/skills/task-manager/scripts"
  mkdir -p "$GLOBAL_SKILLS_DIR"
  ln -sf "$TSK_DIR/tsk" "$GLOBAL_SKILLS_DIR/tsk"
fi

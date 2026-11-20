#!/bin/bash
set -euo pipefail

# Install semble tool for code analysis (idempotent)
command -v semble &>/dev/null || uv tool install semble

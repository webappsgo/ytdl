#!/usr/bin/env bash
set -euo pipefail

# Auto-detect container runtime and run tests
# See AI.md PART 29 for testing specifications

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "=== ytdl Test Runner ==="
echo "Project: $PROJECT_ROOT"

# Auto-detect container runtime
if command -v incus &>/dev/null; then
  echo "Runtime: Incus (preferred)"
  exec "$SCRIPT_DIR/incus.sh" "$@"
elif command -v docker &>/dev/null; then
  echo "Runtime: Docker"
  exec "$SCRIPT_DIR/docker.sh" "$@"
else
  echo "ERROR: No container runtime found (need incus or docker)"
  exit 1
fi

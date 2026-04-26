#!/usr/bin/env bash
set -euo pipefail

# Incus-based full OS integration testing (PREFERRED)
# See AI.md PART 29 for testing specifications

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
PROJECTNAME="ytdl"
PROJECTORG="casapps"
INSTANCE_NAME="${PROJECTNAME}-test"

echo "=== ytdl Incus Integration Tests ==="

# Build binary first
echo "[1/6] Building binary..."
cd "$PROJECT_ROOT"
make local

BINARY="$PROJECT_ROOT/binaries/$PROJECTNAME"
if [ ! -f "$BINARY" ]; then
  echo "ERROR: Build failed - binary not found at $BINARY"
  exit 1
fi

# Create or reuse Incus instance
echo "[2/6] Setting up Incus instance..."
if incus info "$INSTANCE_NAME" &>/dev/null; then
  echo "Reusing existing instance: $INSTANCE_NAME"
  incus start "$INSTANCE_NAME" 2>/dev/null || true
else
  echo "Creating new instance: $INSTANCE_NAME"
  incus launch images:debian/12 "$INSTANCE_NAME"
  sleep 10
fi

# Push binary to instance
echo "[3/6] Deploying binary..."
incus file push "$BINARY" "${INSTANCE_NAME}/usr/local/bin/$PROJECTNAME"
incus exec "$INSTANCE_NAME" -- chmod +x "/usr/local/bin/$PROJECTNAME"

# Install dependencies
incus exec "$INSTANCE_NAME" -- bash -c "
  apt-get update -qq && apt-get install -y -qq curl python3 python3-pip ffmpeg >/dev/null 2>&1
  pip3 install --break-system-packages yt-dlp 2>/dev/null || true
"

# Start server
echo "[4/6] Starting server..."
incus exec "$INSTANCE_NAME" -- bash -c "
  $PROJECTNAME --mode development --port 8080 --address 0.0.0.0 &
  sleep 3
"

# Run tests
echo "[5/6] Running tests..."
PASS=0
FAIL=0

run_test() {
  local name="$1"
  local cmd="$2"
  if incus exec "$INSTANCE_NAME" -- bash -c "$cmd" >/dev/null 2>&1; then
    echo "  PASS: $name"
    PASS=$((PASS + 1))
  else
    echo "  FAIL: $name"
    FAIL=$((FAIL + 1))
  fi
}

run_test "Health check" "curl -q -LSsf http://localhost:8080/healthz | grep -q ok"
run_test "Version" "curl -q -LSsf http://localhost:8080/api/v1/version | grep -q version"
run_test "Home (HTML)" "curl -q -LSsf -H 'Accept: text/html' http://localhost:8080/ | grep -q ytdl"
run_test "Home (JSON)" "curl -q -LSsf -H 'Accept: application/json' http://localhost:8080/ | grep -q ytdl"
run_test "Downloads list" "curl -q -LSsf http://localhost:8080/api/v1/downloads | grep -q data"
run_test "Help flag" "/usr/local/bin/$PROJECTNAME --help | grep -q Usage"
run_test "Version flag" "/usr/local/bin/$PROJECTNAME --version | grep -q $PROJECTNAME"
run_test "Systemd ready" "which systemctl >/dev/null 2>&1"

echo "[6/6] Cleaning up..."
incus exec "$INSTANCE_NAME" -- bash -c "pkill $PROJECTNAME 2>/dev/null || true"
incus stop "$INSTANCE_NAME" 2>/dev/null || true

echo "Results: $PASS passed, $FAIL failed"

if [ "$FAIL" -gt 0 ]; then
  exit 1
fi

echo "All tests passed!"

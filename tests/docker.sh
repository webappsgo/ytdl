#!/usr/bin/env bash
set -euo pipefail

# Docker-based integration testing
# See AI.md PART 29 for testing specifications
# Uses --network host to access local SMTP for auto-detection per PART 18

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
PROJECTNAME="ytdl"
PROJECTORG="casapps"
TEST_PORT=64590

echo "=== ytdl Docker Integration Tests ==="

# Build binary first
echo "[1/7] Building binary..."
cd "$PROJECT_ROOT"
make dev
BUILD_DIR=$(ls -td "${TMPDIR:-/tmp}/${PROJECTORG}/${PROJECTNAME}-"*/ 2>/dev/null | head -1)

if [ -z "$BUILD_DIR" ] || [ ! -f "$BUILD_DIR/$PROJECTNAME" ]; then
  echo "ERROR: Build failed - binary not found"
  exit 1
fi

echo "Binary: $BUILD_DIR/$PROJECTNAME"

# Run binary in Docker container with host network (for SMTP access)
echo "[2/7] Starting server in Docker (host network for SMTP access)..."
CONTAINER_NAME="${PROJECTNAME}-test-$$"

docker run -d --rm \
  --name "$CONTAINER_NAME" \
  --network host \
  -v "$BUILD_DIR:/app:ro" \
  alpine:latest sh -c "
    apk add --no-cache curl bash >/dev/null 2>&1
    /app/$PROJECTNAME --address 0.0.0.0 --port $TEST_PORT --mode development 2>&1 &
    sleep 5
    tail -f /dev/null
  "

sleep 6
echo "Server starting on port $TEST_PORT"

# Run tests
echo "[3/7] Running API tests..."
PASS=0
FAIL=0

run_test() {
  local name="$1"
  local cmd="$2"
  if eval "$cmd" >/dev/null 2>&1; then
    echo "  PASS: $name"
    PASS=$((PASS + 1))
  else
    echo "  FAIL: $name"
    FAIL=$((FAIL + 1))
  fi
}

# Core endpoints
run_test "GET /healthz" \
  "curl -q -LSsf http://localhost:$TEST_PORT/healthz | grep -q ok"

run_test "GET /api/v1/version" \
  "curl -q -LSsf http://localhost:$TEST_PORT/api/v1/version | grep -q version"

run_test "GET / (HTML)" \
  "curl -q -LSsf -H 'Accept: text/html' http://localhost:$TEST_PORT/ | grep -q ytdl"

run_test "GET / (JSON)" \
  "curl -q -LSsf -H 'Accept: application/json' http://localhost:$TEST_PORT/ | grep -q ytdl"

# API endpoints
echo "[4/7] Running API endpoint tests..."

run_test "GET /api/v1/downloads" \
  "curl -q -LSsf http://localhost:$TEST_PORT/api/v1/downloads | grep -q data"

run_test "GET /api/v1/analytics" \
  "curl -q -LSsf http://localhost:$TEST_PORT/api/v1/analytics | grep -q total_downloads"

run_test "GET /api/v1/presets" \
  "curl -q -LSsf http://localhost:$TEST_PORT/api/v1/presets | grep -q data"

run_test "GET /api/v1/collections" \
  "curl -q -LSsf http://localhost:$TEST_PORT/api/v1/collections | grep -q data"

run_test "GET /api/v1/watch-rules" \
  "curl -q -LSsf http://localhost:$TEST_PORT/api/v1/watch-rules | grep -q data"

run_test "GET /openapi.json" \
  "curl -q -LSsf http://localhost:$TEST_PORT/openapi.json | grep -q openapi"

run_test "GET /metrics (Prometheus)" \
  "curl -q -LSsf http://localhost:$TEST_PORT/metrics | grep -q ytdl_uptime"

run_test "GET /robots.txt" \
  "curl -q -LSsf http://localhost:$TEST_PORT/robots.txt | grep -q Disallow"

run_test "GET /.well-known/security.txt" \
  "curl -q -LSsf http://localhost:$TEST_PORT/.well-known/security.txt | grep -q Contact"

run_test "GET /manifest.json (PWA)" \
  "curl -q -LSsf http://localhost:$TEST_PORT/manifest.json | grep -q ytdl"

# Security tests
echo "[5/7] Running security tests..."

run_test "CSRF cookie set" \
  "curl -q -LSsf -v http://localhost:$TEST_PORT/ 2>&1 | grep -qi 'ytdl_csrf'"

run_test "Security headers present" \
  "curl -q -LSsfI http://localhost:$TEST_PORT/ 2>&1 | grep -qi 'X-Frame-Options'"

run_test "CORS preflight" \
  "curl -q -LSsf -X OPTIONS -H 'Origin: http://test.com' -H 'Access-Control-Request-Method: POST' http://localhost:$TEST_PORT/api/v1/downloads 2>&1 | grep -q ''"

# Extended health check
echo "[6/7] Running extended health and SMTP tests..."

run_test "GET /healthz?detail=true (extended)" \
  "curl -q -LSsf 'http://localhost:$TEST_PORT/healthz?detail=true' | grep -q uptime"

# SMTP auto-detection test
# The server should have auto-detected the local SMTP server per PART 18
SMTP_LOG=$(docker logs "$CONTAINER_NAME" 2>&1 | grep -i "smtp" || true)
if echo "$SMTP_LOG" | grep -qi "auto-detected\|verified"; then
  echo "  PASS: SMTP auto-detected"
  PASS=$((PASS + 1))
else
  echo "  FAIL: SMTP not auto-detected (expected local SMTP on this host)"
  echo "    Server logs: $SMTP_LOG"
  FAIL=$((FAIL + 1))
fi

# Admin setup page (no admins yet = redirect to setup)
run_test "GET /admin/server/setup (setup page)" \
  "curl -q -LSsf http://localhost:$TEST_PORT/admin/server/setup | grep -qi 'setup'"

echo "[7/7] Cleaning up..."
docker rm -f "$CONTAINER_NAME" 2>/dev/null

echo ""
echo "Results: $PASS passed, $FAIL failed"

if [ "$FAIL" -gt 0 ]; then
  echo "SOME TESTS FAILED"
  exit 1
fi

echo "All tests passed!"

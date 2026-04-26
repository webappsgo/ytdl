# Development

## Prerequisites

- Docker (for containerized builds)
- Git

## Building

All builds use Docker containers. Go is NOT required on the local machine.

```bash
# Quick dev build (to temp directory)
make dev

# Local production build (to binaries/)
make local

# Full cross-platform build (all 8 platforms)
make build

# Run tests
make test
```

## Testing

```bash
# Auto-detect container runtime (Incus preferred, Docker fallback)
./tests/run_tests.sh

# Docker tests
./tests/docker.sh

# Incus tests (preferred - full OS with systemd)
./tests/incus.sh
```

## Project Structure

```
src/                          # Go source code
src/main.go                   # Entry point with CLI flags
src/config/                   # Configuration package
src/paths/                    # OS-specific path resolution
src/mode/                     # Application modes
src/scheduler/                # Built-in task scheduler
src/server/                   # HTTP server
src/server/handler/           # HTTP request handlers
src/server/service/           # Business logic (yt-dlp, queue, audio, media)
src/server/store/             # Data access layer (SQLite)
docker/                       # Docker configuration
tests/                        # Test scripts
docs/                         # Documentation
```

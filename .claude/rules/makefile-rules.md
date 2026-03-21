# Makefile Rules (PART 26)

⚠️ **These rules are NON-NEGOTIABLE. Violations are bugs.** ⚠️

## CRITICAL - NEVER DO

- NEVER use Makefile in CI/CD pipelines
- NEVER run go commands directly on local machine
- NEVER hardcode Go version in Makefile

## CRITICAL - ALWAYS DO

- ALWAYS use Makefile for local development only
- ALWAYS use Docker (golang:alpine) for builds
- ALWAYS use GODIR/GOCACHE for build speed
- ALWAYS build from ./src directory

## Required Targets

| Target | Purpose | Output |
|--------|---------|--------|
| `make dev` | Quick dev build | `${TMPDIR}/CASAPPS/ytdl-XXXXXX/` |
| `make local` | Production test build | `binaries/` (with version) |
| `make build` | Full release (8 platforms) | `binaries/` |
| `make test` | Unit tests | Coverage report |
| `make clean` | Clean build artifacts | - |

For complete details, see AI.md PART 26

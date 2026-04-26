# Project Structure Rules (PART 2, 3, 4)

⚠️ **These rules are NON-NEGOTIABLE. Violations are bugs.** ⚠️

## CRITICAL - NEVER DO

- NEVER use GPL/AGPL/LGPL dependencies
- NEVER hardcode projectname/projectorg - infer from git remote or path
- NEVER create forbidden files (CHANGELOG.md, .env, config files in repo)
- NEVER create forbidden dirs (config/, data/, logs/, tmp/ in root)
- NEVER use plural directory names (handlers/ → handler/)
- NEVER use -musl suffix in binary names
- NEVER put Dockerfile in project root

## CRITICAL - ALWAYS DO

- ALWAYS use MIT License with LICENSE.md in root
- ALWAYS embed 3rd party licenses in LICENSE.md
- ALWAYS support 8 platforms: linux/darwin/windows/freebsd × amd64/arm64
- ALWAYS use CGO_ENABLED=0 (pure Go, single static binary)
- ALWAYS use OS-specific paths (PART 4)
- ALWAYS use server.yml for config (not .yaml)
- ALWAYS resolve project root before path-sensitive operations

## Key Variables

| Variable | Value |
|----------|-------|
| `{projectname}` | `ytdl` |
| `{projectorg}` | `casapps` |
| `{PROJECTNAME}` | `YTDL` |
| `{PROJECTORG}` | `CASAPPS` |

## Directory Structure

```
src/           # Go source code
docker/        # Docker files (Dockerfile, docker-compose.yml)
tests/         # Test scripts
docs/          # ReadTheDocs (MkDocs)
binaries/      # Build output (gitignored)
releases/      # Release artifacts (gitignored)
rootfs/        # Runtime volumes (gitignored)
```

## Key Rules

| Rule | Description |
|------|-------------|
| **Project root** | All paths are relative to the git/project root, not the current cwd |
| **Runtime paths** | Use OS-specific config/data/cache/log/backup paths from PART 4 |
| **Docker paths** | `/config/**` and `/data/**` are Docker-only, not native host paths |
| **License** | MIT only, with third-party attribution in `LICENSE.md` |
| **Binary builds** | Single static binaries with `CGO_ENABLED=0` |

## Binary Naming

`ytdl-{os}-{arch}` (e.g., ytdl-linux-amd64, ytdl-darwin-arm64)

For complete details, see AI.md PART 2, 3, 4

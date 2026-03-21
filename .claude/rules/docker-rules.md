# Docker Rules (PART 27)

⚠️ **These rules are NON-NEGOTIABLE. Violations are bugs.** ⚠️

## CRITICAL - NEVER DO

- NEVER put Dockerfile in project root - use docker/Dockerfile
- NEVER modify ENTRYPOINT/CMD - customize via entrypoint.sh
- NEVER use .env files
- NEVER run docker-compose in project dir - use temp directory

## CRITICAL - ALWAYS DO

- ALWAYS use multi-stage Dockerfile (golang:alpine + alpine:latest)
- ALWAYS use STOPSIGNAL SIGRTMIN+3
- ALWAYS use tini as init process
- ALWAYS install required packages: git, curl, bash, tini, tor
- ALWAYS default to America/New_York timezone (TZ env override)
- ALWAYS use internal port 80 (external random 64xxx)

## Key Configuration

| Setting | Value |
|---------|-------|
| **Dockerfile** | `docker/Dockerfile` |
| **Internal port** | 80 |
| **STOPSIGNAL** | SIGRTMIN+3 |
| **ENTRYPOINT** | `["tini", "-p", "SIGTERM", "--", "/usr/local/bin/entrypoint.sh"]` |
| **Timezone** | America/New_York (override with TZ) |
| **Volumes** | `./rootfs/config:/config:z`, `./rootfs/data:/data:z` |

For complete details, see AI.md PART 27

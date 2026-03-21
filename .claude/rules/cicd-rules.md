# CI/CD Rules (PART 28)

⚠️ **These rules are NON-NEGOTIABLE. Violations are bugs.** ⚠️

## CRITICAL - NEVER DO

- NEVER use Makefile in CI/CD pipelines
- NEVER hardcode Go version (use go-version: 'stable')
- NEVER skip platforms in release builds

## CRITICAL - ALWAYS DO

- ALWAYS use explicit commands with all env vars in CI
- ALWAYS ensure GitHub/Gitea/Jenkins workflows match
- ALWAYS strip v prefix from semver tags only (v1.2.3 → 1.2.3, dev → dev)
- ALWAYS build Docker images on every push
- ALWAYS use proper LDFLAGS (-s -w -X main.Version/CommitID/BuildDate/OfficialSite)

## Docker Tags

| Trigger | Tags |
|---------|------|
| Any push | `devel`, `{commit}` |
| Beta branch | adds `beta` |
| Version tag | `{version}`, `latest`, `YYMM`, `{commit}` |

For complete details, see AI.md PART 28

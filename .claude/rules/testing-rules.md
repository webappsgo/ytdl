# Testing & Documentation Rules (PART 29, 30, 31)

⚠️ **These rules are NON-NEGOTIABLE. Violations are bugs.** ⚠️

## CRITICAL - NEVER DO

- NEVER run tests on local machine - containers only
- NEVER skip test verification before claiming done

## CRITICAL - ALWAYS DO

- ALWAYS use containers for testing (Incus preferred, Docker fallback)
- ALWAYS have tests/run_tests.sh that auto-detects container runtime
- ALWAYS have tests/docker.sh and tests/incus.sh
- ALWAYS use MkDocs Material theme for ReadTheDocs
- ALWAYS support I18N with Go embed for translations
- ALWAYS comply with WCAG 2.1 AA accessibility

## Testing Hierarchy

| Method | Best For |
|--------|----------|
| **Incus** (preferred) | Full integration, systemd, persistent |
| **Docker** (fallback) | Quick checks, ephemeral |
| `make test` | Unit tests |

## Documentation

| Component | Tool |
|-----------|------|
| ReadTheDocs | MkDocs with Material theme |
| API docs | Swagger/OpenAPI + GraphQL |
| Config | mkdocs.yml in project root |

For complete details, see AI.md PART 29, 30, 31

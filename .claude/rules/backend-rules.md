# Backend Rules (PART 9, 10, 11, 32)

⚠️ **These rules are NON-NEGOTIABLE. Violations are bugs.** ⚠️

## CRITICAL - NEVER DO

- NEVER use bcrypt for new passwords - use Argon2id
- NEVER store plaintext passwords/tokens
- NEVER use string concatenation for SQL - parameterized queries only
- NEVER log passwords, tokens, or secrets
- NEVER expose internal details in user-facing errors
- NEVER ignore errors

## CRITICAL - ALWAYS DO

- ALWAYS use Argon2id for password hashing (OWASP 2023 params)
- ALWAYS use SHA-256 for API token hashing
- ALWAYS use parameterized SQL queries
- ALWAYS wrap errors with context
- ALWAYS use early returns for error handling
- ALWAYS use structured logging
- ALWAYS implement rate limiting on all endpoints
- ALWAYS add security headers (CSP, HSTS, X-Frame-Options, etc.)

## Key Rules

| Rule | Description |
|------|-------------|
| **Default DB** | SQLite (modernc.org/sqlite) - never mattn/go-sqlite3 |
| **Password hash** | Argon2id (time=3, memory=64MB, threads=4, keylen=32) |
| **Token hash** | SHA-256 (fast lookup, tokens are high-entropy) |
| **Valkey/Redis** | Required for caching/clustering support |
| **Error handling** | Wrap with context, early returns, never ignore |
| **Logging** | Structured JSON in production, readable in development |
| **Audit log** | Who/what/when for security events |
| **Tor** | Optional hidden service support (PART 32) |

## Error Message Levels

| Destination | Detail |
|-------------|--------|
| User (WebUI/API) | Minimal, helpful, no internals |
| Admin panel | Actionable, no stack traces |
| Console | Full detail for debugging |
| Log file | Full + context, structured |

For complete details, see AI.md PART 9, 10, 11, 32

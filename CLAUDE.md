# ytdl - Project Rules

## Source of Truth

- **AI.md** is the implementation spec (PARTS 0-37) - READ-ONLY
- **IDEA.md** defines project features - update when features change
- If IDEA.md conflicts with AI.md, AI.md wins

## Critical Rules

- **NEVER** run Go commands locally - use `make dev`, `make local`, `make build`, `make test`
- **NEVER** use Makefile in CI/CD - explicit commands with env vars
- **NEVER** use bcrypt - use Argon2id for passwords
- **NEVER** use CGO - CGO_ENABLED=0 always
- **NEVER** put comments inline - always above code
- **NEVER** guess or assume - ask when uncertain
- **NEVER** run `git add`, `git commit`, `git push` - write `.git/COMMIT_MESS` instead

## Project Variables

- projectname: ytdl
- projectorg: casapps
- Official site: https://dl.csj.rocks

## Key Patterns

- Config: `server.yml`, hierarchy CLI > env > file > defaults, env prefix `YTDL_`
- Boolean: `config.ParseBool()` (40+ variants), never `strconv.ParseBool()`
- Database: SQLite (modernc.org/sqlite), Argon2id passwords, SHA-256 tokens
- Router: go-chi/chi/v5
- Scheduler: robfig/cron/v3 (built-in, never external cron)

## Before Every Task

1. Read relevant AI.md PARTs
2. Check `.claude/rules/` for quick reference
3. Verify every 3-5 changes against spec

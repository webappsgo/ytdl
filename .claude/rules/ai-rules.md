# AI Assistant & Critical Rules (PART 0, 1)

⚠️ **These rules are NON-NEGOTIABLE. Violations are bugs.** ⚠️

## CRITICAL - NEVER DO

- NEVER guess or assume - ASK when uncertain
- NEVER modify AI.md PARTS 0-36
- NEVER create report files (AUDIT.md, COMPLIANCE.md, SUMMARY.md, etc.)
- NEVER rely on memory - re-read the spec
- NEVER add unrequested features
- NEVER "improve" or "optimize" the spec
- NEVER run Go commands locally - use make targets only
- NEVER put comments inline - ALWAYS above code
- NEVER include AI attribution anywhere (code, commits, PRs, docs)
- NEVER run `git add`, `git commit`, `git push` - write COMMIT_MESS instead

## CRITICAL - ALWAYS DO

- ALWAYS read relevant PARTs before implementing
- ALWAYS re-read spec before each task
- ALWAYS verify every 3-5 changes against spec
- ALWAYS read file before modifying
- ALWAYS search before creating
- ALWAYS test before claiming done
- ALWAYS follow spec exactly as written
- ALWAYS ask when uncertain (50x cheaper than guessing wrong)
- ALWAYS use container-only development (Docker/Incus)

## Key Rules

| Rule | Description |
|------|-------------|
| **Full web app** | Every feature works via browser, PWA, API, CLI |
| **Security-first** | Never trust input, defense in depth, fail secure |
| **Container-only dev** | make dev, make local, make build, make test |
| **Comments above** | All comments go ABOVE code, never inline |
| **Spec is truth** | AI.md is source of truth, IDEA.md defines features |
| **No AI attribution** | All output appears as human senior developer work |

For complete details, see AI.md PART 0, 1

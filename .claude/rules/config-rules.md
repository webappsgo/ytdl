# Configuration Rules (PART 5, 6, 12)

⚠️ **These rules are NON-NEGOTIABLE. Violations are bugs.** ⚠️

## CRITICAL - NEVER DO

- NEVER put YAML comments inline - ALWAYS above the setting
- NEVER use `strconv.ParseBool()` - use `config.ParseBool()`
- NEVER hardcode config values from dev machine
- NEVER store config files in the repo (runtime-generated)
- NEVER require SSH/CLI for config changes

## CRITICAL - ALWAYS DO

- ALWAYS normalize and validate all paths
- ALWAYS use config.ParseBool() (40+ boolean variations: yes/no, true/false, 1/0, on/off, enable/disable)
- ALWAYS make settings configurable via admin WebUI
- ALWAYS support live reload for config changes (no restart)
- ALWAYS use 2-space indentation in YAML
- ALWAYS keep `server.yml` as the config source of truth for single-instance mode

## Key Rules

| Rule | Description |
|------|-------------|
| **Config file** | `server.yml` (never .yaml) |
| **Comments** | Above settings, never inline |
| **Boolean parsing** | `config.ParseBool()` accepts 40+ variations |
| **Admin WebUI** | 100% of settings editable via web interface |
| **Live reload** | Changes apply immediately without restart |
| **Restart exceptions** | Only port/address changes may require restart (with warning) |
| **App modes** | production and development |
| **Config path** | Root: `/etc/casapps/ytdl/server.yml`; user: `~/.config/casapps/ytdl/server.yml` |
| **First-run port** | Random unused `64xxx` port is selected once and persisted |
| **Path safety** | Validate and normalize all config/request/file paths before use |

## YAML Comment Style

```yaml
# CORRECT
# Enable feature
enabled: true

# WRONG
enabled: true  # Enable feature
```

## Boolean Handling

- Use `config.ParseBool()` or `config.IsTruthy()` for env vars, config values, CLI flags, API params, and form fields
- Empty input uses the default value
- Invalid input must return an error instead of silently defaulting

## Admin Configuration Rules

- Every `server.yml` setting must be editable in the admin WebUI
- Config changes must live reload immediately unless the setting explicitly requires restart
- Port, address, and database driver changes may require restart and must warn the user

For complete details, see AI.md PART 5, 6, 12

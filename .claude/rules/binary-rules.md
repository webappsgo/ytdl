# Binary Requirements Rules (PART 7, 8, 33)

⚠️ **These rules are NON-NEGOTIABLE. Violations are bugs.** ⚠️

## CRITICAL - NEVER DO

- NEVER use CGO (CGO_ENABLED=0 always)
- NEVER add short flags except -h (help) and -v (version)
- NEVER change/rename/remove the standard CLI flags
- NEVER run go commands directly - use Makefile targets

## CRITICAL - ALWAYS DO

- ALWAYS build single static binary with embedded assets (Go embed)
- ALWAYS build from ./src directory
- ALWAYS name binaries: ytdl-{os}-{arch} (windows adds .exe)
- ALWAYS support all required CLI flags

## Required CLI Flags

```
--help (-h)              --version (-v)
--mode {production|development}
--config {config_dir}    --data {data_dir}
--log {log_dir}          --pid {pid_file}
--address {listen}       --port {port}
--baseurl {path}         --debug
--status                 --daemon
--service {start,restart,stop,reload,--install,--uninstall,--disable,--help}
--maintenance {backup,restore,update,mode,setup,--help}
--update [check|yes|branch {stable|beta|daily}]
```

## Binaries

| Binary | Naming | Required |
|--------|--------|----------|
| Server | `ytdl-{os}-{arch}` | YES |
| Client | `ytdl-cli-{os}-{arch}` | YES |
| Agent | `ytdl-agent-{os}-{arch}` | OPTIONAL |

For complete details, see AI.md PART 7, 8, 33

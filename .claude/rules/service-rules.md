# Service Rules (PART 24, 25)

⚠️ **These rules are NON-NEGOTIABLE. Violations are bugs.** ⚠️

## CRITICAL - NEVER DO

- NEVER assume root/admin privileges
- NEVER skip privilege checks before service operations

## CRITICAL - ALWAYS DO

- ALWAYS support --service flag with: start, restart, stop, reload, --install, --uninstall, --disable, --help
- ALWAYS detect OS and use appropriate service manager
- ALWAYS handle privilege escalation gracefully

## Service Managers by OS

| OS | Service Manager | Service File |
|----|----------------|--------------|
| Linux | systemd | /etc/systemd/system/ytdl.service |
| macOS | launchd | /Library/LaunchDaemons/com.casapps.ytdl.plist |
| Windows | Windows Service Manager | Windows Service |
| BSD | rc.d | /usr/local/etc/rc.d/ytdl |

For complete details, see AI.md PART 24, 25

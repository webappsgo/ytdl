# Features Rules (PART 18-23)

⚠️ **These rules are NON-NEGOTIABLE. Violations are bugs.** ⚠️

## CRITICAL - NEVER DO

- NEVER use external cron, Task Scheduler, or external schedulers
- NEVER hardcode email templates
- NEVER skip GeoIP database updates

## CRITICAL - ALWAYS DO

- ALWAYS use built-in scheduler (robfig/cron/v3)
- ALWAYS support email notifications with templates
- ALWAYS use ip-location-db for GeoIP
- ALWAYS expose Prometheus metrics
- ALWAYS implement backup/restore via --maintenance flag
- ALWAYS implement self-update via --update flag

## Key Features

| Feature | Description |
|---------|-------------|
| **Email** | Templates, SMTP support, notification system |
| **Scheduler** | Built-in cron (PART 19), never external |
| **GeoIP** | ip-location-db, weekly updates via scheduler |
| **Metrics** | Prometheus format at /metrics |
| **Backup** | --maintenance backup/restore, tar.gz archives |
| **Update** | --update with stable/beta/daily branches |

For complete details, see AI.md PART 18-23

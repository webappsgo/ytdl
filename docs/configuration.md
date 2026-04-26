# Configuration

Configuration is auto-generated on first run at the OS-appropriate location. All settings are editable via the admin panel.

## Config File Location

| OS | Root | User |
|----|------|------|
| Linux | `/etc/casapps/ytdl/server.yml` | `~/.config/casapps/ytdl/server.yml` |
| macOS | `/Library/Application Support/casapps/ytdl/server.yml` | `~/Library/Application Support/casapps/ytdl/server.yml` |
| Docker | `/config/ytdl/server.yml` | - |

## Key Settings

| Setting | Default | Description |
|---------|---------|-------------|
| `server.mode` | `production` | Application mode |
| `server.address` | `0.0.0.0` | Listen address |
| `server.port` | Random 64xxx | Listen port |
| `server.admin_path` | `admin` | Admin panel URL path |
| `download.workers` | `2` | Concurrent download workers |
| `download.retention_hours` | `72` | File retention (0 = forever) |
| `download.default_audio_bitrate` | `320k` | Default MP3 bitrate (CBR) |
| `download.default_video_quality` | `1080` | Default video quality |
| `download.default_subtitle_languages` | `en,es` | Auto-download subtitles |

## Environment Variables

All settings can be overridden with `YTDL_` prefixed environment variables:

```bash
YTDL_MODE=development
YTDL_PORT=8080
YTDL_DEBUG=true
YTDL_WORKERS=4
```

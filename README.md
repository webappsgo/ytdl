# ytdl

[![Build](https://github.com/casapps/ytdl/actions/workflows/build.yml/badge.svg)](https://github.com/casapps/ytdl/actions/workflows/build.yml)
[![Release](https://img.shields.io/github/v/release/casapps/ytdl)](https://github.com/casapps/ytdl/releases)
[![License](https://img.shields.io/github/license/casapps/ytdl)](LICENSE.md)
[![Docs](https://readthedocs.org/projects/casapps-ytdl/badge/?version=latest)](https://casapps-ytdl.readthedocs.io)

## About

A self-hosted media downloader web application and API powered by yt-dlp. Download video, audio, and subtitles from 1000+ supported sites with a YouTube-like web interface. Features queue-based downloads, real-time progress, playlist support, embedded lyrics, subtitle tracks, and a full media library.

## Official Site

https://dl.csj.rocks

## Features

- Download video, audio, and subtitles from 1000+ sites via yt-dlp
- Queue-based downloads with configurable worker pool and priority levels
- MP3 with full ID3 tags (CBR 320kbps), embedded lyrics (EN/ES), volume normalization
- Auto-download English and Spanish subtitles, embedded as selectable video tracks
- Full transcode support via ffmpeg (remux, codec change, resolution, bitrate)
- In-app search across any yt-dlp searchable site
- Media library with grid/list views, search, filter, and inline player
- Playlist and batch URL support with individual item tracking
- Channel/playlist watching with scheduled auto-download
- Download presets for quick format/quality selection
- Manual and smart auto-collections
- Shareable download links with optional expiration
- RSS/podcast feed generation from audio downloads
- Real-time progress via WebSocket
- Admin panel with full settings management
- Plex/Jellyfin library integration and webhook support
- Browser extension API compatibility
- PWA with share-target and clipboard URL detection
- Analytics dashboard with time-series charts and CSV/JSON export
- Dark/light/auto theme support, mobile-first responsive design

## Production

### Docker (Recommended)

```bash
docker run -d \
  --name ytdl \
  -p 64580:80 \
  -v ./rootfs/config:/config:z \
  -v ./rootfs/data:/data:z \
  ghcr.io/casapps/ytdl:latest
```

### Docker Compose

```bash
curl -q -LSsf -O https://raw.githubusercontent.com/casapps/ytdl/main/docker/docker-compose.yml
docker compose up -d
```

### Binary

```bash
# Download latest release
curl -q -LSsf -O https://github.com/casapps/ytdl/releases/latest/download/ytdl-linux-amd64

# Make executable and run
chmod +x ytdl-linux-amd64
./ytdl-linux-amd64
```

### CLI Companion

```bash
# Download the companion CLI
curl -q -LSsf -O https://github.com/casapps/ytdl/releases/latest/download/ytdl-cli-linux-amd64

# Make executable and query a running server
chmod +x ytdl-cli-linux-amd64
./ytdl-cli-linux-amd64 --server https://dl.csj.rocks health
```

## Configuration

Configuration is auto-generated on first run. Edit via admin panel at `https://dl.csj.rocks/admin` (admin_path defaults to "admin").

Key settings:
- `server.port` - Listen port (default: random 64xxx)
- `server.mode` - production or development
- `download.workers` - Concurrent download workers (default: 2)
- `download.retention_hours` - File retention period (default: 72, 0 = forever)
- `download.default_audio_bitrate` - Audio bitrate (default: 320k)
- `download.default_video_quality` - Video quality (default: 1080)

## API

API documentation available at `https://dl.csj.rocks/api/v1/` when running.

| Endpoint | Description |
|----------|-------------|
| `GET https://dl.csj.rocks/healthz` | Health check |
| `GET https://dl.csj.rocks/api/v1/version` | Version info |
| `POST https://dl.csj.rocks/api/v1/downloads` | Submit download |
| `GET https://dl.csj.rocks/api/v1/downloads` | List downloads |
| `GET https://dl.csj.rocks/api/v1/search?q=query` | Search sites |
| `GET https://dl.csj.rocks/api/v1/library` | Browse media library |
| `GET https://dl.csj.rocks/api/v1/feed/rss` | RSS/podcast feed |

### Examples

```bash
# Health check
curl -q -LSsf https://dl.csj.rocks/healthz

# Submit download
curl -q -LSsf -X POST -H "Content-Type: application/json" \
  -d '{"url":"https://youtube.com/watch?v=example","format":"mp3","quality":"best"}' \
  https://dl.csj.rocks/api/v1/downloads

# Search YouTube
curl -q -LSsf "https://dl.csj.rocks/api/v1/search?q=example&site=youtube&limit=5"

# List downloads
curl -q -LSsf https://dl.csj.rocks/api/v1/downloads?status=completed
```

## Other

### Troubleshooting

- Check logs: `docker logs ytdl`
- Health check: `curl -q -LSsf https://dl.csj.rocks/healthz`
- Server status: `./ytdl --status`

## Development

**Development instructions are for contributors only.**

### Prerequisites

- Docker (for containerized builds)

### Build

```bash
# Clone
git clone https://github.com/casapps/ytdl
cd ytdl

# Quick dev build (outputs to OS temp dir)
make dev

# Full build (all platforms, outputs to binaries/)
make build

# Test
make test
```

### Project Structure

```
src/           # Source code
tests/         # Test files
docker/        # Docker configuration
docs/          # Documentation
binaries/      # Built binaries (gitignored)
```

## Disclaimer

This software is provided "as is" without warranty of any kind. Use at your own risk.

- **No Warranty**: The authors are not responsible for any damages, data loss, or issues arising from use of this software
- **Not Professional Advice**: This software does not constitute legal, financial, medical, or other professional advice
- **Third-Party Services**: If this software connects to external APIs or services, their terms of service apply separately
- **Security**: While we strive to follow security best practices, no software is guaranteed to be free of vulnerabilities
- **Production Use**: Evaluate thoroughly before deploying in production environments

By using this software, you acknowledge that you have read and understood this disclaimer.

## License

MIT - See [LICENSE.md](LICENSE.md)

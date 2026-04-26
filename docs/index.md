# ytdl

Self-hosted media downloader powered by yt-dlp.

## Overview

ytdl is a web application and API for downloading video, audio, and subtitles from 1000+ supported sites. It features a queue-based download engine, real-time progress tracking, a media library, and comprehensive metadata management including embedded lyrics and subtitles.

## Quick Start

```bash
docker run -d --name ytdl -p 64580:80 \
  -v ./rootfs/config:/config:z \
  -v ./rootfs/data:/data:z \
  ghcr.io/casapps/ytdl:latest
```

Then open `http://localhost:64580` in your browser.

## Key Features

- Download from 1000+ sites via yt-dlp
- Queue-based downloads with priority levels
- MP3 with full ID3 tags and embedded lyrics (EN/ES)
- Subtitle embedding (EN/ES) as selectable tracks
- Media library with search, filter, and inline player
- Channel/playlist watching with auto-download
- RSS/podcast feed generation
- Admin panel with full settings management
- Dark/light/auto theme support

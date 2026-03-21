# ytdl

## Project Description

A self-hosted media downloader web application and API powered by yt-dlp. Provides a YouTube-like web interface for searching, downloading, and managing video, audio, and subtitles from 1000+ supported sites. Downloads are queued and processed in the background with real-time progress tracking. Features a full media library for browsing and playing downloaded content, with support for playlists, batch downloads, channel watching, and comprehensive metadata management including embedded lyrics and subtitles in English and Spanish.

**Target Users:**
- Self-hosted enthusiasts who want a private media downloader
- Small teams needing a shared download service
- Users who want a web UI for yt-dlp without command-line knowledge
- Non-technical users who want simple paste-and-download functionality

---

## Project-Specific Features

- **Media Downloads**: Queue-based downloading from 1000+ sites via yt-dlp with format/quality selection
- **Audio Processing**: MP3 with full ID3 tags (CBR 320kbps default), volume normalization, silence trimming
- **Lyrics**: Embedded synced (SYLT) and unsynced (USLT) lyrics from yt-dlp metadata and external APIs (LRCLIB), English and Spanish
- **Subtitles**: Auto-download English and Spanish subtitles (SRT + VTT), embedded into video containers (MKV/MP4) as selectable tracks
- **Video**: Default 1080p Full HD, full transcode support via ffmpeg
- **In-App Search**: Search any yt-dlp searchable site directly from the web UI
- **Media Library**: Full media manager with grid/list views, album/artist grouping, smart filters, inline player
- **Channel Watching**: Monitor channels/playlists on schedule, configurable auto-download or notify-only per watch rule
- **Collections**: Manual and smart auto-collections based on configurable rules
- **Metadata Editor**: Auto-tag from source metadata plus manual ID3 tag editor in web UI
- **Sharing**: Shareable direct download links with optional expiration, RSS/podcast feed generation
- **Integrations**: Webhook support, Plex/Jellyfin library integration, browser extension API compatibility
- **Analytics**: Detailed time-series charts, per-site breakdowns, storage trends, CSV/JSON export
- **PWA**: Installable with share-target API, clipboard URL detection, offline queue

---

## Detailed Specification

### Data Models

- **Download**: id, url, title, description, source_site, channel_name, channel_url, thumbnail_url, duration, status (queued/downloading/processing/completed/failed/cancelled/paused), format, quality, bitrate, priority (high/normal/low), progress_percent, file_size, file_path, thumbnail_path, error_message, retry_count, proxy_config, created_at, started_at, completed_at, expires_at
- **PlaylistItem**: id, download_id (parent playlist), item_index, url, title, status, file_path, thumbnail_path
- **WatchRule**: id, name, url (channel/playlist), check_interval, last_checked_at, action (auto_download/notify), preset_id, enabled, created_at
- **DownloadPreset**: id, name, format, quality, bitrate, audio_only, subtitle_languages, embed_subtitles, embed_lyrics, normalize_audio, trim_silence, output_template, is_default
- **Collection**: id, name, description, type (manual/smart), rules_json (for smart collections), cover_image_path, created_at
- **CollectionItem**: id, collection_id, download_id, sort_order, added_at
- **MediaMetadata**: id, download_id, title, artist, album, year, genre, track_number, cover_art_path, lyrics_synced, lyrics_unsynced, lyrics_language
- **ScheduleRule**: id, name, start_time, end_time, days_of_week, speed_limit, pause_downloads, enabled
- **AnalyticsEvent**: id, event_type, download_id, site, format, file_size, duration_seconds, created_at

### Business Rules

**Download Engine:**
- yt-dlp binary must be available in PATH or configured path (Docker image bundles it, auto-updates on startup and via scheduler)
- Downloads are queued and processed by configurable number of concurrent workers (default: 2, scales with CPU)
- Maximum queue size is configurable (default: 100)
- Priority levels: high (jumps queue), normal (default), low (processed last)
- Failed downloads auto-retry (configurable max retries, default: 3)
- Download progress reported via WebSocket for real-time UI updates
- Full resilience: resume interrupted downloads, auto-retry on network failure, queue persists across server restarts, crash recovery

**Duplicate Detection:**
- URL-based: warn if same URL already in queue or completed within retention period
- Title matching: fuzzy title matching across different sources
- Audio/video fingerprinting: optional cross-source duplicate detection

**Audio Defaults:**
- MP3: CBR 320kbps (user-configurable bitrate)
- Full ID3 tags: title, artist, album, year, genre, track number, cover art, lyrics
- Volume normalization (loudnorm) and silence trimming available per preset
- Lyrics: prefer synced (SYLT) from yt-dlp metadata, fall back to external API (LRCLIB), then unsynced (USLT)
- Lyrics languages: English and Spanish required

**Video Defaults:**
- Default quality: 1080p Full HD (user-configurable)
- Full transcode support via ffmpeg (codec, bitrate, resolution changes)
- Container remux (MKV to MP4) without re-encoding when possible
- Subtitles: auto-download English and Spanish (SRT + VTT), embed into container as selectable tracks

**File Management:**
- Configurable retention period (default: 72 hours, 0 = keep forever)
- Expired files cleaned up by scheduler
- Configurable output path template (e.g., {site}/{channel}/{title}.{ext}) via admin settings
- Maximum file size limit configurable (default: 0 = unlimited)
- Disk usage dashboard with storage trends
- Auto-cleanup rules (age, size, total usage limit)
- Archive to external/network storage (NFS, S3-compatible, local path)

**Proxy Support:**
- HTTP proxy, SOCKS5, and Tor (Tor binary available in Docker per spec)
- Global proxy configuration or per-download override

**Scheduling:**
- Bandwidth limits by time of day (e.g., 2MB/s daytime, unlimited overnight)
- Quiet hours: configurable periods where no new downloads start
- Delay downloads until specific times
- Global download speed limit (admin-configurable)

**Channel Watching:**
- Monitor channels/playlists on configurable schedule
- Per-watch-rule action: auto-download with preset or notify-only
- Detect new content since last check

**Authentication Sources:**
- Import browser cookies (Netscape format) for private/age-restricted content
- Optional site credentials (stored securely, not required)
- Public content works without any authentication

### Features

**Core Downloads:**
- Submit URL for download with format/quality/subtitle selection
- View download queue with real-time progress via WebSocket
- Cancel, pause, resume queued or active downloads
- Retry failed downloads
- Delete completed downloads (removes file and record)
- Batch submit: paste multiple URLs (one per line) to queue all
- Playlist detection and expansion with individual item tracking
- Download priority levels (high/normal/low)

**Search:**
- Search any yt-dlp searchable site directly from web UI
- Display results with thumbnails, titles, channel info
- Click result to queue download with preset selection

**Media Library:**
- Browse downloaded media in grid view (thumbnails) or detailed list view
- Search, filter, sort by all metadata fields (title, artist, site, format, date, size)
- Album/artist grouping for audio content
- Smart filters (configurable rules)
- Cover art display
- Inline media player: video and audio playback with metadata, lyrics, cover art display

**Metadata and Tags:**
- Auto-tag from source metadata (title, artist, album art from source)
- Manual ID3 tag editor in web UI (edit title, artist, album, year, genre, cover art, lyrics)
- Embedded lyrics (synced SYLT + unsynced USLT) in MP3 files
- Embedded subtitles (EN + ES) as selectable tracks in video containers

**Collections:**
- Create named manual collections, add/remove downloads
- Smart auto-collections based on configurable rules (by site, format, date range, language)
- Collection cover images

**Channel Watching:**
- Add channel/playlist URLs to watch list
- Configurable check interval per watch rule
- Action per rule: auto-download with preset or notify-only
- View watch rule status and last checked time

**Presets:**
- Save named download presets (format, quality, bitrate, subtitles, lyrics, normalization)
- Set default preset
- Apply preset when submitting downloads

**Sharing and Export:**
- Generate shareable direct download links with optional expiration
- RSS/podcast feed generation from audio downloads
- Import/export download history, presets, watch rules as backup

**Integrations:**
- Webhook support: send events on download complete/fail for external automation
- Plex/Jellyfin: auto-organize into library structure, trigger library scan
- Browser extension API: endpoint for external extensions to submit URLs
- PWA share-target: share URLs from mobile apps directly to ytdl
- Clipboard URL detection in PWA

**Analytics:**
- Dashboard with download statistics (total downloads, total size, per-site breakdowns)
- Time-series charts for bandwidth usage and download activity
- Storage trends and usage monitoring
- Most used formats, popular sites
- Export stats as CSV/JSON

**Notifications:**
- Real-time WebSocket notifications in web UI - see PART 18
- Browser push notifications (PWA)
- Optional email notifications for completed/failed downloads - see PART 18

### Endpoints

- Submit download request (URL, format, quality, preset) - see PART 14
- List downloads (filterable by status, paginated) - see PART 14
- Get download details and progress - see PART 14
- Download/stream completed file - see PART 14
- Cancel/pause/resume download - see PART 14
- Retry failed download - see PART 14
- Delete download - see PART 14
- Batch submit URLs - see PART 14
- Search sites via yt-dlp - see PART 14
- List/create/update/delete download presets - see PART 14
- List/create/update/delete watch rules - see PART 14
- List/create/update/delete collections - see PART 14
- Get/update media metadata and tags - see PART 14
- Generate shareable download link - see PART 14
- RSS/podcast feed endpoint - see PART 14
- Media library browse/search/filter - see PART 14
- Analytics data endpoints - see PART 14
- WebSocket endpoint for real-time progress and notifications - see PART 14
- Get yt-dlp version and supported sites - see PART 14
- Import/export backup data - see PART 14

### Data Sources

- yt-dlp binary for media extraction, downloading, and site search
- External lyrics API (LRCLIB) for synced/unsynced lyrics
- Database for download queue, history, presets, collections, watch rules, analytics - see PART 10
- Filesystem for downloaded media files ({data_dir}/downloads/)
- yt-dlp site extractors list for supported sites display
- ffmpeg for transcoding, audio processing, subtitle embedding

### Admin-Specific Configuration

- Concurrent download workers count
- Maximum queue size
- Global download speed limit
- File retention period (hours, 0 = forever)
- Maximum file size limit
- Default output path template
- yt-dlp binary path and update schedule
- ffmpeg binary path
- Download directory path
- Default audio bitrate (CBR 320kbps)
- Default video quality (1080p)
- Default subtitle languages (en, es)
- Proxy configuration (HTTP, SOCKS5, Tor)
- Scheduling rules (bandwidth by time, quiet hours)
- Plex/Jellyfin server URL and library path
- Webhook URLs and events
- Archive storage configuration (local path, NFS, S3)
- Auto-cleanup rules (max age, max size, max total usage)
- Cookie import settings

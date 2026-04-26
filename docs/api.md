# API Reference

## Base URL

All API endpoints are under `/api/v1/`.

## Endpoints

### Health

| Method | Path | Description |
|--------|------|-------------|
| GET | `/healthz` | Health check |
| GET | `/api/v1/healthz` | Health check (API) |
| GET | `/api/v1/version` | Version info |

### Downloads

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/downloads` | Submit download |
| POST | `/api/v1/downloads/batch` | Batch submit URLs |
| GET | `/api/v1/downloads` | List downloads |
| GET | `/api/v1/downloads/{id}` | Get download details |
| DELETE | `/api/v1/downloads/{id}` | Delete download |
| POST | `/api/v1/downloads/{id}/cancel` | Cancel download |
| POST | `/api/v1/downloads/{id}/pause` | Pause download |
| POST | `/api/v1/downloads/{id}/resume` | Resume download |
| POST | `/api/v1/downloads/{id}/retry` | Retry failed download |
| GET | `/api/v1/downloads/{id}/file` | Download file |
| POST | `/api/v1/downloads/{id}/share` | Create share link |
| GET | `/api/v1/downloads/{id}/metadata` | Get metadata |
| PUT | `/api/v1/downloads/{id}/metadata` | Update metadata |

### Search

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/search?q=query&site=youtube&limit=10` | Search sites |

### Library

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/library` | Browse media library |
| GET | `/api/v1/analytics` | Analytics data |
| GET | `/api/v1/feed/rss` | RSS/podcast feed |

### Collections

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/collections` | List collections |
| POST | `/api/v1/collections` | Create collection |
| DELETE | `/api/v1/collections/{id}` | Delete collection |
| POST | `/api/v1/collections/{id}/items` | Add to collection |

### Presets

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/presets` | List presets |
| POST | `/api/v1/presets` | Create preset |
| DELETE | `/api/v1/presets/{id}` | Delete preset |

### Watch Rules

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/watch-rules` | List watch rules |
| POST | `/api/v1/watch-rules` | Create watch rule |
| DELETE | `/api/v1/watch-rules/{id}` | Delete watch rule |

### WebSocket

Connect to `/ws` for real-time download progress updates.

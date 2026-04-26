# Installation

## Docker (Recommended)

```bash
docker run -d --name ytdl -p 64580:80 \
  -v ./rootfs/config:/config:z \
  -v ./rootfs/data:/data:z \
  ghcr.io/casapps/ytdl:latest
```

## Docker Compose

```bash
curl -q -LSsf -O https://raw.githubusercontent.com/casapps/ytdl/main/docker/docker-compose.yml
docker compose up -d
```

## Binary

Download the latest release for your platform from the [releases page](https://github.com/casapps/ytdl/releases).

```bash
curl -q -LSsf -O https://github.com/casapps/ytdl/releases/latest/download/ytdl-linux-amd64
chmod +x ytdl-linux-amd64
./ytdl-linux-amd64
```

## First Run

On first run, ytdl will:

1. Create default configuration
2. Generate a one-time setup token (displayed in console)
3. Start serving immediately

Navigate to `/admin/server/setup` and enter the setup token to create your admin account.

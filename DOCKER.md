# 🐳 Docker Setup for Strix

## Quick Start

### Using Docker Compose (Recommended)

```bash
# Start Strix
docker-compose up -d

# View logs
docker-compose logs -f strix

# Stop Strix
docker-compose down
```

Access: http://localhost:4567

### Using Docker Run

```bash
docker run -d \
  --name strix \
  -p 4567:4567 \
  eduard256/strix:latest
```

## Configuration

### Using Environment Variables

```bash
docker run -d \
  --name strix \
  -p 8080:8080 \
  -e STRIX_API_LISTEN=:8080 \
  -e STRIX_LOG_LEVEL=debug \
  eduard256/strix:latest
```

### Using Config File

```bash
# Create strix.yaml
cat > strix.yaml <<EOF
api:
  listen: ":8080"
EOF

# Run with mounted config
docker run -d \
  --name strix \
  -p 8080:8080 \
  -v $(pwd)/strix.yaml:/app/strix.yaml:ro \
  eduard256/strix:latest
```

## Full Stack (Strix + go2rtc + Frigate)

```bash
docker-compose -f docker-compose.full.yml up -d
```

Services:
- Strix: http://localhost:4567
- go2rtc: http://localhost:1984
- Frigate: http://localhost:5000

## Podman

Strix uses raw sockets for network scanning. Podman drops these capabilities by default,
so you need to add them explicitly. Rootless mode does not support host network scanning —
run with `sudo`.

### Using Podman Run

```bash
sudo podman run -d \
  --name strix \
  --network host \
  --cap-add=NET_RAW \
  --cap-add=NET_ADMIN \
  --restart unless-stopped \
  eduard256/strix:latest
```

- `NET_RAW` — required for network scanning (ARP, ICMP) to discover cameras
- `NET_ADMIN` — required for network interface and routing operations

### Using Podman Compose

```yaml
version: '3'

services:
  strix:
    image: eduard256/strix:latest
    container_name: strix
    restart: unless-stopped
    network_mode: host
    cap_add:
      - NET_RAW
      - NET_ADMIN
    environment:
      - STRIX_LOG_LEVEL=info
      - STRIX_LOG_FORMAT=json
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:4567/api/v1/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 10s
```

```bash
sudo podman-compose up -d
```

### Using Quadlet (systemd)

Recommended for production. Create `/etc/containers/systemd/strix.container`:

```ini
[Unit]
Description=Strix Camera Stream Discovery
After=network-online.target
Wants=network-online.target

[Container]
Image=docker.io/eduard256/strix:latest
ContainerName=strix
Network=host
AddCapability=CAP_NET_RAW CAP_NET_ADMIN
Environment=STRIX_LOG_LEVEL=info
Environment=STRIX_LOG_FORMAT=json
AutoUpdate=registry

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now strix
sudo systemctl status strix
```

Quadlet auto-generates a systemd service from the `.container` file.
The container starts on boot and restarts on failure automatically.

## Building Locally

```bash
# Build for your platform
docker build -t strix:local .

# Build for multiple platforms
docker buildx build --platform linux/amd64,linux/arm64 -t strix:multi .
```

## Image Information

- **Image**: `eduard256/strix:latest`
- **Platforms**: linux/amd64, linux/arm64
- **Size**: ~80-90MB
- **Base**: Alpine Linux
- **User**: Non-root (strix:1000)

## Included Dependencies

- ffmpeg/ffprobe (stream validation)
- ca-certificates (HTTPS support)
- tzdata (timezone support)
- wget (healthcheck)
- Camera database (3600+ models)

## Health Check

```bash
# Check container health
docker inspect --format='{{.State.Health.Status}}' strix

# Manual health check
docker exec strix wget -q -O- http://localhost:4567/api/v1/health
```

## Troubleshooting

### View logs
```bash
docker logs strix
docker logs -f strix  # Follow logs
```

### Check if ffprobe works
```bash
docker exec strix ffprobe -version
```

### Inspect container
```bash
docker exec -it strix sh
```

### Restart container
```bash
docker restart strix
```

## Security

- Runs as non-root user (UID 1000)
- Minimal attack surface (Alpine base)
- No unnecessary packages
- Health checks enabled

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `STRIX_API_LISTEN` | `:4567` | Server listen address |
| `STRIX_LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |
| `STRIX_LOG_FORMAT` | `json` | Log format (json, text) |
| `STRIX_DATA_PATH` | `./data` | Camera database path |

## Volumes

```bash
# Optional: Custom configuration
-v ./strix.yaml:/app/strix.yaml:ro

# Optional: Custom camera database
-v ./data:/app/data:ro
```

## Docker Hub

Pre-built images available at: https://hub.docker.com/r/eduard256/strix

Tags:
- `latest` - Latest stable release
- `v0.1.0` - Specific version
- `0.1` - Minor version
- `0` - Major version
- `main` - Development branch

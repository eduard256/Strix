# Strix for Developers

Strix is a single static binary with embedded web UI and SQLite camera database. No config files, no external dependencies (except optional `ffmpeg` for H264/H265 screenshot conversion). Designed to run alongside your project the same way [go2rtc](https://github.com/AlexxIT/go2rtc) does.

For development and testing without real cameras, use [StrixCamFake](https://github.com/eduard256/StrixCamFake) - IP camera emulator with RTSP, HTTP, RTMP, Bubble and more.

## Binary

Download from [GitHub Releases](https://github.com/eduard256/Strix/releases). Two platforms: `linux/amd64` and `linux/arm64`.

```bash
chmod +x strix-linux-amd64
./strix-linux-amd64
```

The binary needs `cameras.db` in the working directory. Download it from [StrixCamDB](https://github.com/eduard256/StrixCamDB/releases):

```bash
curl -fsSL https://github.com/eduard256/StrixCamDB/releases/latest/download/cameras.db -o cameras.db
./strix-linux-amd64
```

## Docker

```bash
docker run -d --name strix --network host eduard256/strix:latest
```

Database is already embedded in the image.

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `STRIX_LISTEN` | `:4567` | HTTP listen address |
| `STRIX_DB_PATH` | `cameras.db` | Path to SQLite database |
| `STRIX_LOG_LEVEL` | `info` | `trace`, `debug`, `info`, `warn`, `error` |
| `STRIX_FRIGATE_URL` | auto-discovery | Frigate URL, e.g. `http://localhost:5000` |
| `STRIX_GO2RTC_URL` | auto-discovery | go2rtc URL, e.g. `http://localhost:1984` |

## Integration Flow

Typical automation flow using the API:

```
1. Probe device        GET  /api/probe?ip=192.168.1.100
2. Search database     GET  /api/search?q=hikvision
3. Build stream URLs   GET  /api/streams?ids=b:hikvision&ip=192.168.1.100&user=admin&pass=12345
4. Test streams        POST /api/test  {sources: {streams: [...]}}
5. Poll results        GET  /api/test?id=xxx
6. Generate config     POST /api/generate  {mainStream: "rtsp://...", subStream: "rtsp://..."}
```

All endpoints return JSON. CORS is enabled. No authentication.

---

## API Reference

### System

#### `GET /api`

```json
{"version": "2.0.0", "platform": "amd64"}
```

#### `GET /api/health`

```json
{"version": "2.0.0", "uptime": "1h30m0s"}
```

#### `GET /api/log`

Returns in-memory log in `application/jsonlines` format. Passwords are masked automatically.

#### `DELETE /api/log`

Clears in-memory log. Returns `204`.

---

### Search

#### `GET /api/search?q={query}`

Search camera database by brand, model, or preset name. Empty `q` returns all presets + first brands (limit 50).

```bash
curl "localhost:4567/api/search?q=hikvision"
```

```json
{
  "results": [
    {"type": "brand", "id": "b:hikvision", "name": "Hikvision"},
    {"type": "model", "id": "m:hikvision:DS-2CD2032", "name": "Hikvision: DS-2CD2032"}
  ]
}
```

Result types:

| Type | ID format | Description |
|------|-----------|-------------|
| `preset` | `p:{preset_id}` | Curated URL pattern sets (e.g. "ONVIF", "Popular RTSP") |
| `brand` | `b:{brand_id}` | All URL patterns for a brand |
| `model` | `m:{brand_id}:{model}` | URL patterns for a specific model |

Multi-word queries match independently: `hikvision DS-2CD` matches brand "Hikvision" AND model containing "DS-2CD".

#### `GET /api/streams`

Build full stream URLs from database patterns with credentials and placeholders substituted.

| Param | Required | Description |
|-------|----------|-------------|
| `ids` | yes | Comma-separated IDs from search results |
| `ip` | yes | Camera IP address |
| `user` | no | Username (URL-encoded automatically) |
| `pass` | no | Password (URL-encoded automatically) |
| `channel` | no | Channel number, default `0` |
| `ports` | no | Comma-separated port filter (only return URLs matching these ports) |

```bash
curl "localhost:4567/api/streams?ids=b:hikvision&ip=192.168.1.100&user=admin&pass=12345"
```

```json
{
  "streams": [
    "rtsp://admin:12345@192.168.1.100/Streaming/Channels/101",
    "rtsp://admin:12345@192.168.1.100/Streaming/Channels/102",
    "http://admin:12345@192.168.1.100/ISAPI/Streaming/channels/101/picture"
  ]
}
```

Maximum 20,000 URLs per request. URLs are deduplicated.

---

### Testing

#### `POST /api/test`

Create a test session. 20 parallel workers connect to each URL, extract codecs, capture screenshots.

```bash
curl -X POST localhost:4567/api/test -d '{
  "sources": {
    "streams": [
      "rtsp://admin:12345@192.168.1.100/Streaming/Channels/101",
      "rtsp://admin:12345@192.168.1.100/Streaming/Channels/102"
    ]
  }
}'
```

```json
{"session_id": "a1b2c3d4e5f6g7h8"}
```

#### `GET /api/test`

List all active and completed sessions.

```json
{
  "sessions": [
    {
      "session_id": "a1b2c3d4",
      "status": "running",
      "total": 604,
      "tested": 341,
      "alive": 191,
      "with_screenshot": 191
    }
  ]
}
```

#### `GET /api/test?id={session_id}`

Get session details with full results. Poll this endpoint to track progress.

```json
{
  "session_id": "a1b2c3d4",
  "status": "done",
  "total": 604,
  "tested": 604,
  "alive": 375,
  "with_screenshot": 375,
  "results": [
    {
      "source": "rtsp://admin:***@192.168.1.100/Streaming/Channels/101",
      "codecs": ["H264", "PCMA"],
      "width": 1920,
      "height": 1080,
      "latency_ms": 45,
      "screenshot": "api/test/screenshot?id=a1b2c3d4&i=0"
    }
  ]
}
```

- `status`: `running` or `done`
- `codecs`: detected media codecs (H264, H265, PCMA, PCMU, OPUS, etc.)
- `width`, `height`: resolution extracted from JPEG screenshot
- `screenshot`: relative URL to fetch the JPEG image
- Sessions expire 30 minutes after completion

#### `DELETE /api/test?id={session_id}`

Cancel a running session and delete it.

```json
{"status": "deleted"}
```

#### `GET /api/test/screenshot?id={session_id}&i={index}`

Returns raw JPEG image. `Content-Type: image/jpeg`.

---

### Config Generation

#### `POST /api/generate`

Generate Frigate config from stream URLs.

```bash
curl -X POST localhost:4567/api/generate -d '{
  "mainStream": "rtsp://admin:12345@192.168.1.100/Streaming/Channels/101",
  "subStream": "rtsp://admin:12345@192.168.1.100/Streaming/Channels/102",
  "name": "front_door",
  "objects": ["person", "car"]
}'
```

```json
{
  "config": "mqtt:\n  enabled: false\n\nrecord:\n  enabled: true\n\ngo2rtc:\n  streams:\n    ...",
  "added": [1, 2, 3, 4, 5]
}
```

- `config`: complete Frigate YAML
- `added`: 1-based line numbers of new lines (for highlighting in UI)

**Merge into existing config** - pass `existingConfig` field:

```json
{
  "mainStream": "rtsp://...",
  "existingConfig": "go2rtc:\n  streams:\n    existing_cam:\n      - rtsp://...\n\ncameras:\n  existing_cam:\n    ..."
}
```

Strix finds the right insertion points in go2rtc streams and cameras sections. Camera and stream names are deduplicated automatically.

<details>
<summary>Full request schema</summary>

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `mainStream` | string | **yes** | Main stream URL |
| `subStream` | string | no | Sub stream URL for detect role |
| `name` | string | no | Camera name (auto-generated from IP if empty) |
| `existingConfig` | string | no | Existing Frigate YAML to merge into |
| `objects` | string[] | no | Objects to track (default: `["person"]`) |
| `go2rtc` | object | no | `{mainStreamName, subStreamName, mainStreamSource, subStreamSource}` |
| `frigate` | object | no | `{mainStreamPath, subStreamPath, mainStreamInputArgs, subStreamInputArgs}` |
| `detect` | object | no | `{enabled, fps, width, height}` |
| `record` | object | no | `{enabled, retain_days, mode, alerts_days, detections_days, pre_capture, post_capture}` |
| `motion` | object | no | `{enabled, threshold, contour_area}` |
| `snapshots` | object | no | `{enabled}` |
| `audio` | object | no | `{enabled, filters[]}` |
| `ffmpeg` | object | no | `{hwaccel, gpu}` |
| `live` | object | no | `{height, quality}` |
| `birdseye` | object | no | `{enabled, mode}` |
| `onvif` | object | no | `{host, port, user, password, autotracking, required_zones[]}` |
| `ptz` | object | no | `{enabled, presets{}}` |
| `notifications` | object | no | `{enabled}` |
| `ui` | object | no | `{order, dashboard}` |

</details>

---

### Probe

#### `GET /api/probe?ip={ip}`

Probe a network device. Runs 6 checks in parallel within 100ms: port scan, ICMP ping, ARP + OUI vendor lookup, reverse DNS, mDNS/HomeKit query, HTTP probe.

```bash
curl "localhost:4567/api/probe?ip=192.168.1.100"
```

```json
{
  "ip": "192.168.1.100",
  "reachable": true,
  "latency_ms": 2.5,
  "type": "standard",
  "probes": {
    "ping": {"latency_ms": 2.5},
    "ports": {"open": [80, 554, 8080]},
    "dns": {"hostname": "ipcam.local"},
    "arp": {"mac": "C0:56:E3:AA:BB:CC", "vendor": "Hikvision"},
    "mdns": null,
    "http": {"port": 80, "status_code": 401, "server": "Hikvision-Webs"}
  }
}
```

- `type`: `standard`, `homekit`, or `unreachable`
- `ports.open`: scanned from 189 ports known in the camera database
- `arp.vendor`: looked up from OUI table in SQLite database
- HomeKit cameras return `mdns` with `name`, `model`, `category` (`camera` or `doorbell`), `device_id`, `paired`, `port`
- ICMP ping requires `CAP_NET_RAW` capability. Falls back to port scan only.

---

### Frigate

#### `GET /api/frigate/config`

Get current Frigate config. Frigate is discovered automatically by probing known addresses (`localhost:5000`, `ccab4aaf-frigate:5000`) or via `STRIX_FRIGATE_URL`.

```json
{"connected": true, "url": "http://localhost:5000", "config": "mqtt:\n  enabled: false\n  ..."}
```

```json
{"connected": false, "config": ""}
```

#### `POST /api/frigate/config/save?save_option={option}`

Save config to Frigate. Request body is plain text (YAML config).

| Option | Description |
|--------|-------------|
| `saveonly` | Save config without restart (default) |
| `restart` | Save config and restart Frigate |

---

### go2rtc

#### `PUT /api/go2rtc/streams?name={name}&src={source}`

Add a stream to go2rtc. Proxied to local go2rtc instance (discovered automatically or via `STRIX_GO2RTC_URL`).

```bash
curl -X PUT "localhost:4567/api/go2rtc/streams?name=front_door&src=rtsp://admin:12345@192.168.1.100/Streaming/Channels/101"
```

```json
{"success": true}
```

```json
{"success": false, "error": "go2rtc not found"}
```

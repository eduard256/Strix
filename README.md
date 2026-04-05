<p align="center">
  <a href="https://github.com/eduard256/strix/stargazers"><img src="https://img.shields.io/github/stars/eduard256/strix?style=social" alt="GitHub Stars"></a>
  <a href="https://hub.docker.com/r/eduard256/strix"><img src="https://img.shields.io/docker/pulls/eduard256/strix" alt="Docker Pulls"></a>
  <br><br>
  <img src="https://github.com/eduard256/Strix/releases/download/v2.0.0/icon-192.png" width="96">
  <br>
  <b>Strix</b>
</p>

Finds working camera streams. Generates Frigate config. 30 seconds.

3,600+ brands. 100,000+ URL patterns. RTSP, HTTP, RTMP, Bubble, DVRIP.

<a href="https://youtu.be/JgVWsl4NApE">
  <img src="https://github.com/eduard256/Strix/releases/download/v2.0.0/demo.gif" width="100%">
</a>

## Install

Any Linux, one command:

```bash
curl -fsSL https://raw.githubusercontent.com/eduard256/Strix/main/install.sh | sudo bash
```

Open `http://YOUR_IP:4567`

## How it works

Enter camera IP. Strix probes the device -- open ports, MAC vendor, mDNS, HTTP server.

![](https://github.com/eduard256/Strix/releases/download/v2.0.0/01-enter-ip.png)

Search camera model in database. Enter credentials if needed.

![](https://github.com/eduard256/Strix/releases/download/v2.0.0/02-camera-config.png)

Strix builds all possible stream URLs from database patterns.

![](https://github.com/eduard256/Strix/releases/download/v2.0.0/03-stream-urls.png)

20 parallel workers test every URL. Live screenshots, codecs, resolution, latency.

![](https://github.com/eduard256/Strix/releases/download/v2.0.0/04-testing.png)

Pick main and sub streams from results.

![](https://github.com/eduard256/Strix/releases/download/v2.0.0/05-results.png)

Generate ready Frigate config. Copy, download, or save directly to Frigate.

![](https://github.com/eduard256/Strix/releases/download/v2.0.0/06-frigate-config.png)

Camera works in Frigate. Done.

![](https://github.com/eduard256/Strix/releases/download/v2.0.0/07-frigate-result.png)

## Other install methods

### Docker

```bash
docker run -d --name strix --network host --restart unless-stopped eduard256/strix:latest
```

### Home Assistant Add-on

1. **Settings** > **Add-ons** > **Add-on Store**
2. Menu (top right) > **Repositories** > add `https://github.com/eduard256/hassio-strix`
3. Install **Strix**, enable **Start on boot** and **Show in sidebar**

### Binary

Download from [GitHub Releases](https://github.com/eduard256/Strix/releases). No dependencies except `ffmpeg` for screenshot conversion.

```bash
chmod +x strix-linux-amd64
STRIX_LISTEN=:4567 ./strix-linux-amd64
```

## Supported protocols

| Protocol | Port | Description |
|----------|------|-------------|
| RTSP | 554 | Most IP cameras |
| RTSPS | 322 | RTSP over TLS |
| HTTP/HTTPS | 80/443 | MJPEG, JPEG snapshots, HLS, MPEG-TS |
| RTMP | 1935 | Some Chinese NVRs |
| Bubble | 80 | XMeye/NetSurveillance cameras |
| DVRIP | 34567 | Sofia protocol DVR/NVR |

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `STRIX_LISTEN` | `:4567` | HTTP listen address |
| `STRIX_DB_PATH` | `cameras.db` | Path to SQLite camera database |
| `STRIX_LOG_LEVEL` | `info` | Log level: `debug`, `info`, `warn`, `error`, `trace` |
| `STRIX_FRIGATE_URL` | auto-discovery | Frigate URL, e.g. `http://localhost:5000` |
| `STRIX_GO2RTC_URL` | auto-discovery | go2rtc URL, e.g. `http://localhost:1984` |

## Camera database

SQLite database with 3,600+ brands and 100,000+ URL patterns. Maintained separately in [StrixCamDB](https://github.com/eduard256/StrixCamDB). Database is embedded in Docker image and bundled with binary releases.

Three entity types:
- **Presets** -- curated sets of popular URL patterns (e.g. "ONVIF", "Popular RTSP")
- **Brands** -- all URL patterns for a brand (e.g. "Hikvision", "Dahua")
- **Models** -- URL patterns for a specific model within a brand

## API Reference

All endpoints return JSON. CORS enabled. Base path: `/`.

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

Returns in-memory log in `application/jsonlines` format.

#### `DELETE /api/log`

Clears in-memory log. Returns `204`.

### Search

#### `GET /api/search?q={query}`

Search camera database. Empty `q` returns all presets + first brands.

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

Types: `preset`, `brand`, `model`. ID prefixes: `p:`, `b:`, `m:brandId:model`.

#### `GET /api/streams?ids={ids}&ip={ip}&user={user}&pass={pass}&channel={n}&ports={ports}`

Build stream URLs from database patterns.

| Param | Required | Description |
|-------|----------|-------------|
| `ids` | yes | Comma-separated IDs from search results |
| `ip` | yes | Camera IP address |
| `user` | no | Username |
| `pass` | no | Password |
| `channel` | no | Channel number (default 0) |
| `ports` | no | Comma-separated port filter |

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

### Testing

#### `POST /api/test`

Create a test session. Launches 20 parallel workers.

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

List all sessions.

```json
{
  "sessions": [
    {"session_id": "a1b2c3d4", "status": "running", "total": 604, "tested": 341, "alive": 191, "with_screenshot": 191}
  ]
}
```

#### `GET /api/test?id={session_id}`

Get session details with results.

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

Status: `running` or `done`.

#### `DELETE /api/test?id={session_id}`

Cancel and delete session.

#### `GET /api/test/screenshot?id={session_id}&i={index}`

Returns JPEG image. `Content-Type: image/jpeg`.

### Config Generation

#### `POST /api/generate`

Generate Frigate config.

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
  "config": "mqtt:\n  enabled: false\n\nrecord:\n  ...",
  "added": [1, 2, 3, 4]
}
```

`added` -- 1-based line numbers of new lines. Useful for highlighting in UI.

To merge into existing config, pass `existingConfig`:

```json
{
  "mainStream": "rtsp://...",
  "existingConfig": "go2rtc:\n  streams:\n    ...\ncameras:\n  ..."
}
```

Strix finds the right insertion points, deduplicates camera and stream names.

<details>
<summary>Full request schema</summary>

| Field | Type | Description |
|-------|------|-------------|
| `mainStream` | string | **Required.** Main stream URL |
| `subStream` | string | Sub stream URL |
| `name` | string | Camera name |
| `existingConfig` | string | Existing Frigate YAML to merge into |
| `objects` | string[] | Objects to track (default: `["person"]`) |
| `go2rtc` | object | Override stream names/sources |
| `frigate` | object | Override Frigate input paths/args |
| `detect` | object | `{enabled, fps, width, height}` |
| `record` | object | `{enabled, retain_days, mode, alerts_days, detections_days, pre_capture, post_capture}` |
| `motion` | object | `{enabled, threshold, contour_area}` |
| `snapshots` | object | `{enabled}` |
| `audio` | object | `{enabled, filters[]}` |
| `ffmpeg` | object | `{hwaccel, gpu}` |
| `live` | object | `{height, quality}` |
| `birdseye` | object | `{enabled, mode}` |
| `onvif` | object | `{host, port, user, password, autotracking, required_zones[]}` |
| `ptz` | object | `{enabled, presets{}}` |
| `notifications` | object | `{enabled}` |
| `ui` | object | `{order, dashboard}` |

</details>

### Probe

#### `GET /api/probe?ip={ip}`

Probe a network device. Runs all checks in parallel within 100ms.

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

Type: `standard`, `homekit`, or `unreachable`.

HomeKit cameras return `mdns` with `name`, `model`, `category`, `device_id`, `paired`, `port`.

### Frigate

#### `GET /api/frigate/config`

Get Frigate config. Frigate is discovered automatically via known candidates or `STRIX_FRIGATE_URL`.

```json
{"connected": true, "url": "http://localhost:5000", "config": "mqtt:\n  ..."}
```

```json
{"connected": false, "config": ""}
```

#### `POST /api/frigate/config/save?save_option={option}`

Save config to Frigate. Body: plain text (YAML). Options: `saveonly`, `restart`.

### go2rtc

#### `PUT /api/go2rtc/streams?name={name}&src={source}`

Add stream to go2rtc. Proxied to local go2rtc instance.

```bash
curl -X PUT "localhost:4567/api/go2rtc/streams?name=front_door&src=rtsp://admin:12345@192.168.1.100/Streaming/Channels/101"
```

```json
{"success": true}
```

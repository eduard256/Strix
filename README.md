<h1 align="center">
  <a href="https://github.com/eduard256/Strix">
    <img src="https://github.com/eduard256/Strix/releases/download/v2.0.0/icon-192.png" width="64" alt="Strix" valign="middle">
  </a>
  &nbsp;|&nbsp;
  STRIX
</h1>
<p align="center">
  <a href="https://github.com/eduard256/strix/stargazers"><img src="https://img.shields.io/github/stars/eduard256/strix?style=flat-square&logo=github" alt="GitHub Stars"></a>
  <a href="https://hub.docker.com/r/eduard256/strix"><img src="https://img.shields.io/docker/pulls/eduard256/strix?style=flat-square&logo=docker&logoColor=white&label=pulls" alt="Docker Pulls"></a>
  <a href="https://github.com/eduard256/Strix/releases"><img src="https://img.shields.io/github/downloads/eduard256/Strix/total?color=blue&style=flat-square&logo=github" alt="GitHub Downloads"></a>
  <a href="https://github.com/eduard256/Strix/blob/main/LICENSE"><img src="https://img.shields.io/badge/License-MIT-blue.svg?style=flat-square" alt="License"></a>
</p>

Camera stream discovery and Frigate config generator.

- 3,600+ camera brands with 100,000+ [URL patterns](#streams) in SQLite database
- automatic device [probing](#probe) in 100ms: ports, ARP/OUI, mDNS/HomeKit, HTTP
- 20 parallel workers [test every URL](#testing) with live screenshots
- supports [RTSP, HTTP, RTMP, Bubble, DVRIP](#supported-protocols) and more
- ready [Frigate config](#config-generation) with smart merge into existing setup
- auto-discovery of Frigate and [go2rtc](https://github.com/AlexxIT/go2rtc) on local network
- zero-dependency static [binary](#binary) for Linux amd64/arm64
- can be used as [standalone app](#binary), [Docker](#docker), or [Home Assistant add-on](#home-assistant-add-on)

---

<a href="https://youtu.be/JgVWsl4NApE">
  <img src="https://github.com/eduard256/Strix/releases/download/v2.0.0/demo.gif" width="100%">
</a>

<p align="center">
  <a href="https://gostrix.github.io/demo.html"><b>Live Demo</b></a>
  &nbsp;&bull;&nbsp;
  <a href="https://gostrix.github.io/"><b>Supported Cameras</b></a>
  &nbsp;&bull;&nbsp;
  <a href="https://youtu.be/JgVWsl4NApE"><b>Video</b></a>
  &nbsp;&bull;&nbsp;
  <a href="DEVELOPERS.md"><b>API Docs</b></a>
</p>

## Install

Any Linux or Proxmox, one command:

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/eduard256/Strix/main/install.sh)
```

Run as root (or with `sudo`). Interactive installer detects your system (Linux / Proxmox) and guides you through setup.

Open `http://YOUR_IP:4567`

## How it works

<a id="probe"></a>

Enter camera IP. Strix probes the device - open ports, MAC vendor, mDNS, HTTP server.

![](https://github.com/eduard256/Strix/releases/download/v2.0.0/01-enter-ip.png)

<a id="search"></a>

Search camera model in database. Enter credentials if needed.

![](https://github.com/eduard256/Strix/releases/download/v2.0.0/02-camera-config.png)

<a id="streams"></a>

Strix builds all possible stream URLs from database patterns.

![](https://github.com/eduard256/Strix/releases/download/v2.0.0/03-stream-urls.png)

<a id="testing"></a>

20 parallel workers test every URL. Live screenshots, codecs, resolution, latency.

![](https://github.com/eduard256/Strix/releases/download/v2.0.0/04-testing.png)

Pick main and sub streams from results.

![](https://github.com/eduard256/Strix/releases/download/v2.0.0/05-results.png)

<a id="config-generation"></a>

Generate ready Frigate config. Copy, download, or save directly to Frigate.

![](https://github.com/eduard256/Strix/releases/download/v2.0.0/06-frigate-config.png)

Camera works in Frigate. Done.

![](https://github.com/eduard256/Strix/releases/download/v2.0.0/07-frigate-result.png)

## Other install methods

### Docker

```bash
docker run -d --name strix --network host --restart unless-stopped eduard256/strix:latest
```

### Docker Compose

Strix only:

```bash
curl -O https://raw.githubusercontent.com/eduard256/Strix/main/docker-compose.yml
docker compose up -d
```

Strix + [Frigate](https://github.com/blakeblackshear/frigate):

```bash
curl -O https://raw.githubusercontent.com/eduard256/Strix/main/docker-compose.frigate.yml
docker compose -f docker-compose.frigate.yml up -d
```

Strix + [go2rtc](https://github.com/AlexxIT/go2rtc):

```bash
curl -O https://raw.githubusercontent.com/eduard256/Strix/main/docker-compose.go2rtc.yml
docker compose -f docker-compose.go2rtc.yml up -d
```

### Podman

Podman drops `NET_RAW` and `NET_ADMIN` by default, which Strix needs for network scanning. Add them explicitly:

```bash
podman run -d \
  --name strix \
  --network host \
  --cap-add=NET_RAW \
  --cap-add=NET_ADMIN \
  --restart unless-stopped \
  eduard256/strix:latest
```

Or run with `--privileged` if you prefer.

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
| HomeKit | 51826 | Apple HomeKit cameras via HAP |

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

[Browse supported cameras](https://gostrix.github.io/) - search by brand or model to check if your camera is in the database.

Three entity types:
- **Presets** - curated sets of popular URL patterns (e.g. "ONVIF", "Popular RTSP")
- **Brands** - all URL patterns for a brand (e.g. "Hikvision", "Dahua")
- **Models** - URL patterns for a specific model within a brand

Camera not in the database? [Add it here](https://gostrix.github.io/#/contribute).

**Developers:** integrate [Strix HTTP API](DEVELOPERS.md) into your smart home platform.

**Testing:** [StrixCamFake](https://github.com/eduard256/StrixCamFake) - IP camera emulator for development and testing. [StrixAHKCamFake](https://github.com/eduard256/StrixAHKCamFake) - Apple HomeKit camera emulator.

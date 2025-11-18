# Strix

## Spent 2 years googling URL for your Chinese camera?

**Strix finds working streams automatically. In 30 seconds.**

- **67,288** camera models
- **3,636** brands (from Hikvision to AliExpress no-name)
- **102,787** URL patterns (RTSP, HTTP, MJPEG, JPEG, BUBBLE)

[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Docker Pulls](https://img.shields.io/docker/pulls/eduard256/strix)](https://hub.docker.com/r/eduard256/strix)

![Demo](assets/main.gif)

---

## Your Problem?

- ❌ Bought ZOSI NVR, zero documentation
- ❌ Camera has no RTSP, only weird JPEG snapshots
- ❌ Frigate eating 70% CPU
- ❌ Config breaks after adding each camera
- ❌ Don't understand Frigate syntax

## Solution

- ✅ **Auto-discovery** - tests 102,787 URL variations in parallel
- ✅ **Any protocol** - No RTSP? Finds HTTP MJPEG
- ✅ **Config generation** - ready Frigate.yml in 2 minutes
- ✅ **Sub/Main streams** - CPU from 70% → 8%
- ✅ **Smart merging** - adds camera to existing config with 500+ cameras

---

## 🚀 Installation (One Command)

### Ubuntu / Debian

```bash
sudo apt update && command -v docker >/dev/null 2>&1 || curl -fsSL https://get.docker.com | sudo sh && docker run -d --name strix -p 4567:4567 --restart unless-stopped eduard256/strix:latest
```

Open **http://YOUR_SERVER_IP:4567**

### Docker Compose

```bash
curl -fsSL https://raw.githubusercontent.com/eduard256/Strix/main/docker-compose.yml -o docker-compose.yml && docker-compose up -d
```

### Home Assistant Add-on (Beta)

⚠️ **Status:** Experimental (SSE has bugs, Docker recommended)

**Installation:**

1. Go to **Settings** → **Add-ons** → **Add-on Store**
2. Click **⋮** (top right) → **Repositories**
3. Add: `https://github.com/eduard256/hassio-strix`
4. Find **"Strix"** in store
5. Click **Install**
6. Enable **"Start on boot"** and **"Show in sidebar"**
7. Click **Start**

**Known Issues:**
- Real-time progress may not display (Ingress SSE limitation)
- Use Docker installation for better experience

---

## How to Use

### Step 1: Open Web Interface

```
http://YOUR_SERVER_IP:4567
```

### Step 2: Enter Camera Details

- **IP Address**: `192.168.1.100`
- **Username**: `admin` (if required)
- **Password**: your camera password
- **Model**: optional, improves accuracy

### Step 3: Discover Streams

Click **"Discover Streams"**

Watch real-time progress:
- Which URL is being tested
- How many tested
- Found streams appear instantly

Wait 30-60 seconds.

### Step 4: Choose Stream

Strix shows details for each stream:

| Stream | Details |
|--------|---------|
| **Protocol** | RTSP, HTTP, MJPEG, JPEG |
| **Resolution** | 1920x1080, 640x480 |
| **FPS** | 25, 15, 10 |
| **Codec** | H264, H265, MJPEG |
| **Audio** | Yes / No |

### Step 5: Generate Frigate Config

Click **"Use Stream"** → **"Generate Frigate Config"**

You get ready config:

```yaml
go2rtc:
  streams:
    '192_168_1_100_main':
      - http://admin:pass@192.168.1.100:8000/video.mjpg
    '192_168_1_100_sub':
      - http://admin:pass@192.168.1.100:8000/video2.mjpg

cameras:
  camera_192_168_1_100:
    ffmpeg:
      inputs:
        - path: rtsp://127.0.0.1:8554/192_168_1_100_sub
          roles: [detect]      # CPU 8% instead of 70%
        - path: rtsp://127.0.0.1:8554/192_168_1_100_main
          roles: [record]      # HD recording
    objects:
      track: [person, car, cat, dog]
    record:
      enabled: true
```

**Smart Merging:**
- Paste your existing `frigate.yml` with 500 cameras
- Strix adds camera #501 correctly
- Doesn't break structure
- Preserves all settings

### Step 6: Add to Frigate

Copy config → Paste to `frigate.yml` → Restart Frigate

**Done!**

---

## Features

### Exotic Camera Support

90% of Chinese cameras don't have RTSP. Strix supports everything:

- **HTTP MJPEG** - most old cameras
- **JPEG snapshots** - auto-converted to stream via FFmpeg
- **RTSP** - if available
- **HTTP-FLV** - some Chinese brands
- **BUBBLE** - proprietary Chinese NVR/DVR protocol
- **ONVIF** - auto-discovery

### Camera Database

**67,288 models from 3,636 brands:**

- **Known brands**: Hikvision, Dahua, Axis, Foscam, TP-Link
- **Chinese no-names**: ZOSI, Escam, Sricam, Wanscam, Besder
- **AliExpress junk**: cameras without name, OEM models
- **Old systems**: NVR/DVR with proprietary protocols

### Discovery Methods

Strix tries all methods in parallel:

**1. ONVIF** (30% success rate)
- Asks camera directly for stream URLs
- Works for ONVIF-compatible cameras

**2. Database Lookup** (60% success rate)
- 67,288 models with known working URLs
- Brand and model-specific patterns

**3. Popular Patterns** (90% success rate)
- 206 most common URL paths
- Works even for unknown cameras

**Result: Finds stream for 95% of cameras**

### Frigate Config Generation

**What you get:**

✅ **Main/Sub streams**
- Main (HD) for recording
- Sub (low res) for object detection
- CPU usage reduced 5-10x

✅ **Ready go2rtc config**
- Stream multiplexing
- Protocol conversion
- JPEG → RTSP via FFmpeg

✅ **Smart config merging**
- Add to existing config
- Preserve structure
- No manual YAML editing

✅ **Pre-configured detection**
- person, car, cat, dog
- Ready motion recording
- 7 days retention

### Speed

- Tests **20 URLs in parallel**
- Average discovery time: **30-60 seconds**
- Complex cameras: **2-3 minutes**
- Real-time progress updates via SSE

---

## Advanced Configuration

### Docker Environment Variables

```yaml
environment:
  - STRIX_API_LISTEN=:8080        # Custom port
  - STRIX_LOG_LEVEL=debug         # Detailed logs
  - STRIX_LOG_FORMAT=json         # JSON logging
```

### Config File

Create `strix.yaml`:

```yaml
api:
  listen: ":8080"
```

Example: [strix.yaml.example](strix.yaml.example)

### Discovery Parameters

In web UI under **Advanced**:

- **Channel** - for NVR systems (usually 0)
- **Timeout** - max discovery time (default: 240s)
- **Max Streams** - stop after N streams (default: 10)

---

## FAQ

### No streams found?

**Check network:**
```bash
ping 192.168.1.100
```

Camera must be reachable.

**Verify credentials:**
- Username/password correct?
- Try without credentials (some cameras are open)

**Try without model:**
- Strix will run ONVIF + 206 popular patterns
- Works for cameras not in database

### Camera not in database?

**No problem.**

Strix will still find stream via:
1. ONVIF (if supported)
2. 206 popular URL patterns
3. Common ports and paths
4. HTTP MJPEG on various ports
5. JPEG snapshot endpoints

**Help the project:**
- Found working stream? [Create Issue](https://github.com/eduard256/Strix/issues)
- Share model and URL
- We'll add to database

### Found only JPEG snapshots?

**Normal for old cameras.**

Strix auto-converts JPEG to stream via FFmpeg:

```yaml
go2rtc:
  streams:
    camera_main:
      - exec:ffmpeg -loop 1 -framerate 10 -i http://192.168.1.100/snapshot.jpg -c:v libx264 -f rtsp {output}
```

Frigate gets normal 10 FPS stream.

### Stream found but doesn't work in Frigate?

**Try another stream:**
- Strix usually finds 3-10 variants
- Some may need special FFmpeg parameters

**Use sub stream:**
- For object detection
- Less CPU load
- Better performance

### How does config generation work?

**For new config:**
- Strix creates complete `frigate.yml` from scratch
- Includes go2rtc, camera, object detection

**For existing config:**
- Paste your current `frigate.yml`
- Strix adds new camera
- Preserves all existing cameras
- Doesn't break structure

**Main/Sub streams:**
- Main (HD) - for recording
- Sub (low res) - for detection
- CPU savings 5-10x

### Is it safe to enter passwords?

**Yes.**

- Strix runs locally on your network
- Nothing sent to external servers
- Passwords not saved
- Open source - check the code yourself

### Works offline?

**Yes.**

- Database embedded in Docker image
- Internet only needed to download image
- Runs offline after that

---

## API Reference

REST API available for automation:

### Health Check

```bash
GET /api/v1/health
```

### Search Cameras

```bash
POST /api/v1/cameras/search

{
  "query": "hikvision",
  "limit": 10
}
```

### Discover Streams (SSE)

```bash
POST /api/v1/streams/discover

{
  "target": "192.168.1.100",
  "username": "admin",
  "password": "12345",
  "model": "DS-2CD2xxx",
  "timeout": 240,
  "max_streams": 10
}
```

Returns Server-Sent Events with real-time progress.

**Full API documentation:** [DOCKER.md](DOCKER.md)

---

## Technical Details

### Architecture

- **Language:** Go 1.24
- **Database:** 3,636 JSON files
- **Image size:** 80-90 MB (Alpine Linux)
- **Dependencies:** FFmpeg/FFprobe for validation
- **Concurrency:** Worker pool (20 parallel tests)
- **Real-time:** Server-Sent Events (SSE)

### Build from Source

```bash
git clone https://github.com/eduard256/Strix
cd Strix
make build
./bin/strix
```

**Requirements:**
- Go 1.21+
- FFprobe (optional, for stream validation)

### Docker Platforms

- linux/amd64
- linux/arm64

Auto-built and published to Docker Hub on every push to `main`.

---

## Use Cases

### Home Automation

- Add cheap cameras to Home Assistant
- Integrate with Frigate NVR
- Object detection with low CPU
- Motion recording

### Security Systems

- Discover streams in old NVR systems
- Find backup cameras without docs
- Migrate from proprietary DVR to Frigate
- Reduce hardware requirements

### IP Camera Testing

- Test cameras before deployment
- Verify stream quality
- Find optimal resolution/FPS
- Check codec compatibility

---

## Troubleshooting

### Frigate still eating CPU?

**Use sub stream:**
1. Find both main and sub streams with Strix
2. Generate config with both
3. Sub for detect, main for record
4. CPU drops 5-10x

**Example:**
```yaml
inputs:
  - path: rtsp://127.0.0.1:8554/camera_sub   # 640x480 for detect
    roles: [detect]
  - path: rtsp://127.0.0.1:8554/camera_main  # 1920x1080 for record
    roles: [record]
```

### Can't find specific stream quality?

**In web UI:**
- Strix shows all found streams
- Filter by resolution
- Choose optimal FPS
- Select codec (H264 recommended for Frigate)

### Stream works but no audio in Frigate?

**Check Strix stream details:**
- "Has Audio" field shows if audio present
- Some cameras have video-only streams
- Try different stream URL from Strix results

### Discovery takes too long?

**Reduce search scope:**
- Specify exact camera model (faster database lookup)
- Lower "Max Streams" (stops after N found)
- Reduce timeout (default 240s)

**In Advanced settings:**
```
Max Streams: 5     (instead of 10)
Timeout: 120       (instead of 240)
```

---

## Contributing

### Add Your Camera

Found working stream for camera not in database?

1. [Create Issue](https://github.com/eduard256/Strix/issues)
2. Provide:
   - Camera brand and model
   - Working URL pattern
   - Protocol (RTSP/HTTP/etc)
3. We'll add to database

### Report Bugs

- [GitHub Issues](https://github.com/eduard256/Strix/issues)
- Include logs (set `STRIX_LOG_LEVEL=debug`)
- Camera model and IP (if possible)

### Feature Requests

- [GitHub Discussions](https://github.com/eduard256/Strix/discussions)
- Describe use case
- Explain expected behavior

---

## Credits

- **Camera database:** [ispyconnect.com](https://www.ispyconnect.com)
- **Inspiration:** [go2rtc](https://github.com/AlexxIT/go2rtc) by AlexxIT
- **Community:** Home Assistant, Frigate NVR users

---

## License

MIT License - use commercially, modify, distribute freely.

See [LICENSE](LICENSE) file for details.

---

## Support

- **Issues:** [GitHub Issues](https://github.com/eduard256/Strix/issues)
- **Discussions:** [GitHub Discussions](https://github.com/eduard256/Strix/discussions)
- **Docker:** [Docker Hub](https://hub.docker.com/r/eduard256/strix)

---

**Made for people tired of cameras without documentation**

*Tested on Chinese AliExpress junk that finally works now.*

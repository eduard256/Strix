# Strix Camera Discovery - Documentation

## Installation

### Method 1: Add Repository (Recommended)

1. Navigate to **Supervisor** → **Add-on Store** in your Home Assistant
2. Click the **⋮** menu (top right) → **Repositories**
3. Add repository URL: `https://github.com/eduard256/Strix`
4. Find **Strix Camera Discovery** in the store
5. Click **Install**
6. Configure the add-on (optional)
7. Click **Start**
8. Click **Open Web UI**

### Method 2: Manual Installation

1. SSH into your Home Assistant server
2. Navigate to the addons directory:
   ```bash
   cd /addons
   ```
3. Clone the repository:
   ```bash
   git clone https://github.com/eduard256/Strix
   cd Strix/homeassistant-addon
   ```
4. Restart Home Assistant Supervisor
5. Find the add-on in the **Local Add-ons** section

## Configuration

The add-on can be configured through the Home Assistant UI:

```yaml
log_level: info
port: 4567
strict_validation: true
```

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `log_level` | string | `info` | Logging level: `debug`, `info`, `warn`, `error` |
| `port` | integer | `4567` | Port for web interface and API |
| `strict_validation` | boolean | `true` | Enable strict stream validation |

### Advanced Configuration

For advanced users, you can modify environment variables:

- `STRIX_LOG_LEVEL` - Log level (debug, info, warn, error)
- `STRIX_LOG_FORMAT` - Log format (json, text)
- `STRIX_API_LISTEN` - Server listen address (set via `port` option)
- `STRIX_DATA_PATH` - Camera database path (default: `/app/data`)

## Usage

### Quick Start Guide

1. **Open the Web UI**
   - Click "Open Web UI" in the add-on panel
   - Or navigate to: `http://homeassistant.local:4567`

2. **Find Your Camera Model**
   - Use the search bar to find your camera
   - Example: "Hikvision DS-2CD2032"
   - Supports fuzzy search (typos are okay!)

3. **Discover Streams**
   - Enter camera IP address (e.g., `192.168.1.100`)
   - Enter credentials (username/password)
   - Select discovered camera model
   - Click "Discover Streams"

4. **Real-time Progress**
   - Watch live updates as Strix tests different URLs
   - See which streams are working
   - Get detailed validation results

5. **Copy Stream URLs**
   - Copy working URLs to use in Home Assistant
   - Supports RTSP, HTTP, MJPEG, JPEG snapshots

### Camera Search

The search functionality includes:

- **3,600+ camera models** in database
- **Fuzzy matching** - handles typos and variations
- **Brand and model search** - search by manufacturer or model number
- **Popular cameras** - common models are prioritized

Example searches:
- "hikvision" - finds all Hikvision cameras
- "ds-2cd2032" - finds specific model
- "axis m1045" - finds AXIS camera
- "dahua ipc" - finds Dahua IP cameras

### Stream Discovery

Discovery process:

1. **ONVIF Discovery** - Attempts automatic detection via ONVIF protocol
2. **Model Patterns** - Tests URL patterns specific to camera model
3. **Popular Patterns** - Tests 150+ common stream URL patterns
4. **Validation** - Verifies each stream using ffprobe

Stream types discovered:
- RTSP streams (`rtsp://`)
- HTTP streams (`http://`)
- MJPEG streams (`http://.../video.cgi`)
- JPEG snapshots (`http://.../snapshot.jpg`)

### API Usage

#### Health Check

```bash
curl http://homeassistant.local:4567/api/v1/health
```

Response:
```json
{
  "status": "ok",
  "timestamp": "2025-01-15T10:30:00Z"
}
```

#### Camera Search

```bash
curl -X POST http://homeassistant.local:4567/api/v1/cameras/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "hikvision",
    "limit": 10
  }'
```

Response:
```json
{
  "cameras": [
    {
      "brand": "Hikvision",
      "model": "DS-2CD2032-I",
      "score": 0.95
    }
  ],
  "count": 1
}
```

#### Stream Discovery (Server-Sent Events)

```bash
curl -N -X POST http://homeassistant.local:4567/api/v1/streams/discover \
  -H "Content-Type: application/json" \
  -d '{
    "target": "192.168.1.100",
    "model": "hikvision ds-2cd2032",
    "username": "admin",
    "password": "password",
    "timeout": 240,
    "max_streams": 10
  }'
```

SSE Events:
```
event: progress
data: {"message": "Testing RTSP stream...", "percent": 25}

event: stream_found
data: {"url": "rtsp://192.168.1.100:554/stream1", "type": "rtsp"}

event: complete
data: {"total_found": 3, "duration": 45.2}
```

## Integration with Home Assistant

### Generic Camera Platform

```yaml
camera:
  - platform: generic
    name: Front Door
    still_image_url: http://192.168.1.100/snapshot.jpg
    stream_source: rtsp://admin:password@192.168.1.100:554/stream1
    verify_ssl: false
```

### go2rtc Integration

```yaml
go2rtc:
  streams:
    front_door:
      - rtsp://admin:password@192.168.1.100:554/stream1
    back_yard:
      - rtsp://admin:password@192.168.1.101:554/stream1
```

### Frigate Integration

```yaml
cameras:
  front_door:
    ffmpeg:
      inputs:
        - path: rtsp://admin:password@192.168.1.100:554/stream1
          roles:
            - detect
            - record
```

## Troubleshooting

### Add-on won't start

Check the logs:
1. Go to **Supervisor** → **Strix Camera Discovery** → **Log**
2. Look for error messages
3. Common issues:
   - Port 4567 already in use
   - Insufficient resources
   - Database files missing

### Can't find camera model

- Try different search terms (brand name, model number)
- Use partial model numbers
- Check the full database at: `/app/data/brands/`
- If camera not in database, use "Generic" or similar brand camera

### Discovery finds no streams

Possible causes:
1. **Wrong IP address** - Verify camera is reachable: `ping 192.168.1.100`
2. **Wrong credentials** - Double-check username/password
3. **Firewall blocking** - Ensure RTSP port (554) is accessible
4. **ONVIF disabled** - Enable ONVIF in camera settings
5. **Network isolation** - Camera and HA must be on same network

Debug steps:
```bash
# Test if camera responds
curl -u admin:password http://192.168.1.100/

# Test RTSP stream manually
ffprobe rtsp://admin:password@192.168.1.100:554/stream1
```

### FFProbe warnings

If you see "ffprobe not found" warnings:
- This is normal if ffprobe isn't installed
- Stream validation will be limited to HTTP checks
- RTSP streams may not be validated properly
- The add-on includes ffprobe by default

### Slow discovery

Discovery can take 2-4 minutes because:
- Testing 150+ URL patterns
- Validating each stream with ffprobe
- Network latency to camera
- Camera response time

To speed up:
- Select specific camera model (reduces URLs to test)
- Reduce `timeout` value (default: 240 seconds)
- Reduce `max_streams` (stops after N streams found)

### Port conflicts

If port 4567 is in use:
1. Change the `port` option in add-on configuration
2. Restart the add-on
3. Access Web UI at new port

## Performance

### Resource Usage

Typical resource consumption:
- **Memory**: 50-100 MB
- **CPU**: Low (spikes during discovery)
- **Disk**: ~50 MB (including database)
- **Network**: Depends on discovery activity

### Concurrent Discoveries

The add-on can handle multiple concurrent discovery requests:
- Uses worker pool (20 concurrent workers)
- Queues excess requests
- No limit on simultaneous users

## Security

### Network Security

- Add-on runs in Home Assistant network
- No external internet access required
- All traffic is local to your network

### Credentials

- Camera credentials never stored
- Sent only during discovery session
- Not logged (even in debug mode)
- Transmitted over local network only

### Container Security

- Runs as non-root user (UID 1000)
- Minimal attack surface (Alpine base)
- No unnecessary packages
- Read-only filesystem where possible

## Database

### Camera Database

Location: `/app/data/brands/`

Contains:
- 3,600+ camera models
- Organized by brand (JSON files)
- Stream URL patterns
- Query parameter variations

Format example:
```json
{
  "brand": "Hikvision",
  "models": [
    {
      "model": "DS-2CD2032-I",
      "patterns": [
        "/Streaming/channels/101",
        "/h264/ch1/main/av_stream"
      ]
    }
  ]
}
```

### Updating Database

Database updates come with add-on updates:
- Check for updates in Add-on Store
- Updates include new camera models
- No manual database updates needed

## Support

### Getting Help

1. **Documentation**: Read this guide thoroughly
2. **Logs**: Check add-on logs for errors
3. **GitHub Issues**: https://github.com/eduard256/Strix/issues
4. **Community**: Home Assistant Community Forum

### Reporting Bugs

Include in bug reports:
1. Home Assistant version
2. Add-on version
3. Full logs from add-on
4. Camera brand/model
5. Steps to reproduce

### Feature Requests

Submit feature requests on GitHub with:
- Clear description of feature
- Use case / why it's needed
- Any relevant examples

## Changelog

See [CHANGELOG.md](CHANGELOG.md) for version history.

## License

MIT License - See [LICENSE](https://github.com/eduard256/Strix/blob/main/LICENSE)

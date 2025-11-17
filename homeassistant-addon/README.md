# Home Assistant Add-on: Strix Camera Discovery

![Supports aarch64 Architecture][aarch64-shield]
![Supports amd64 Architecture][amd64-shield]
![Supports armv7 Architecture][armv7-shield]

Strix is a smart IP camera stream discovery system that automatically finds and validates camera streams. It eliminates the need for manual URL configuration by using ONVIF discovery, comprehensive camera database, and intelligent stream testing.

## About

This add-on provides Strix - an intelligent camera discovery service for Home Assistant. It includes:

- **3,600+ Camera Models Database** - Comprehensive coverage of IP camera brands and models
- **ONVIF Discovery** - Automatic camera detection on your network
- **Smart Stream Testing** - Validates RTSP, HTTP, MJPEG, and JPEG snapshot URLs
- **Real-time Updates** - Server-Sent Events (SSE) for live discovery progress
- **Web Interface** - Beautiful UI for easy camera management
- **RESTful API** - Integrate with your automation workflows

## Installation

1. Add this repository to your Home Assistant Add-on Store:
   - Click on "Add-on Store" in the Home Assistant Supervisor panel
   - Click the three dots menu (top right) and select "Repositories"
   - Add the URL: `https://github.com/eduard256/Strix`
   - Click "Add"

2. Find "Strix Camera Discovery" in the add-on store and click "Install"

3. After installation, click "Start" to run the add-on

4. Open the Web UI by clicking "Open Web UI" button

## Configuration

```yaml
log_level: info
port: 4567
strict_validation: true
```

### Option: `log_level`

The `log_level` option controls the level of log output by the addon.

- `debug` - Shows detailed debug information
- `info` - Normal (default) log level
- `warn` - Only warnings and errors
- `error` - Only errors

### Option: `port`

The `port` option allows you to change the port on which Strix runs. Default is `4567`.

### Option: `strict_validation`

When enabled (default), Strix performs stricter stream validation:
- Verifies minimum image sizes for snapshots
- Requires at least one video stream in RTSP sources
- Reduces false positives

## How to use

1. **Open the Web UI** - Click "Open Web UI" in the add-on panel or navigate to `http://homeassistant.local:4567`

2. **Search for Camera** - Enter your camera brand/model (e.g., "Hikvision DS-2CD2032")

3. **Discover Streams** - Enter camera IP, credentials, and click "Discover"

4. **Get Stream URLs** - Copy working stream URLs for use in Home Assistant

## Example: Adding discovered camera to Home Assistant

After discovering a camera stream, add it to your `configuration.yaml`:

```yaml
camera:
  - platform: generic
    name: Front Door Camera
    still_image_url: http://192.168.1.100/snapshot.jpg
    stream_source: rtsp://admin:password@192.168.1.100:554/stream1
```

Or use with go2rtc for better performance:

```yaml
go2rtc:
  streams:
    front_door: rtsp://admin:password@192.168.1.100:554/stream1
```

## API Endpoints

The add-on exposes the following API endpoints:

### Health Check
```bash
GET http://homeassistant.local:4567/api/v1/health
```

### Camera Search
```bash
POST http://homeassistant.local:4567/api/v1/cameras/search
Content-Type: application/json

{
  "query": "hikvision",
  "limit": 10
}
```

### Stream Discovery (SSE)
```bash
POST http://homeassistant.local:4567/api/v1/streams/discover
Content-Type: application/json

{
  "target": "192.168.1.100",
  "model": "hikvision ds-2cd2032",
  "username": "admin",
  "password": "password",
  "timeout": 240,
  "max_streams": 10
}
```

## Support

Got questions or issues?

- [GitHub Issues](https://github.com/eduard256/Strix/issues)
- [Home Assistant Community](https://community.home-assistant.io/)

## Contributing

This is an active open-source project. We are always open to people who want to
use the code or contribute to it.

## License

MIT License - see the [LICENSE](https://github.com/eduard256/Strix/blob/main/LICENSE) file for details

[aarch64-shield]: https://img.shields.io/badge/aarch64-yes-green.svg
[amd64-shield]: https://img.shields.io/badge/amd64-yes-green.svg
[armv7-shield]: https://img.shields.io/badge/armv7-yes-green.svg

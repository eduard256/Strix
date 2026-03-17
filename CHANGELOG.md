# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.10] - 2026-03-17

### Added
- Device probe endpoint (GET /api/v1/probe) for network device inspection
- HTTP prober for detecting camera web interfaces
- mDNS discovery for local network devices
- ARP/OUI vendor identification with camera OUI database (2,400+ entries)
- Probe integration into frontend with modal UI
- Added Trassir and ZOSI to OUI database

### Changed
- Removed CI/CD pipelines (GitHub Actions), replaced with local Docker builds
- Removed GoReleaser, unified Docker image for Docker Hub and HA add-on
- Application version now injected at build time via ldflags
- HA add-on reads /data/options.json natively (no more entrypoint script)
- Optimized mDNS discovery timeout

### Fixed
- Removed experimental SSE warning from Home Assistant Add-on documentation
- Clear probe-filled fields when navigating back in frontend

## [1.0.9] - 2025-12-11

### Fixed
- Fixed real-time SSE streaming in Home Assistant Ingress mode
- SSE events now arrive immediately instead of being buffered until completion

### Technical
- Added automatic detection of Home Assistant Ingress via X-Ingress-Path header
- Implemented 64KB padding for SSE events to overcome aiohttp buffer in HA Supervisor
- Adjusted progress update interval to 3 seconds in Ingress mode to reduce traffic
- Normal mode (Docker/direct access) remains unchanged

## [1.0.8] - 2025-11-26

### Changed
- Updated Docker deployment to use host network mode for better compatibility
- Modified docker-compose.yml to use `network_mode: host`
- Updated installation commands to use `--network host` flag
- Removed port mappings as they are not needed with host network mode

### Improved
- Better compatibility with unprivileged LXC containers
- Simplified Docker networking configuration
- Direct network access for improved camera discovery performance

## [1.0.7] - 2025-11-23

### Fixed
- Fixed channel numbering for Hikvision-style cameras (reported by @sergbond_com)
- Removed invalid test data from Hikvision database
- Fixed brand+model search matching in stream discovery

### Added
- Universal `[CHANNEL+1]` placeholder support for flexible channel numbering
- Support for both 0-based (channel=0 → 101) and 1-based (channel=1 → 101) channel selection
- Added 6 high-priority Hikvision patterns to popular stream patterns database

### Changed
- Updated 14 camera brands with universal channel patterns (Hikvision, Hiwatch, Annke, Swann, Abus, 7links, LevelOne, AlienDVR, Oswoo, AV102IP-40, Acvil, TBKVision, Deltaco, Night Owl)
- Hikvision: replaced 10 hardcoded patterns with 6 universal patterns
- Hiwatch: replaced 4 hardcoded patterns with 8 universal patterns (including ISAPI variants)
- Universal patterns now tested first for faster discovery, hardcoded patterns kept as fallback
- Improved stream discovery performance with intelligent pattern ordering

### Technical
- Added support for `[CHANNEL+1]`, `[channel+1]`, `{CHANNEL+1}`, `{channel+1}` placeholders in URL builder
- Modified 16 files: +2448 additions, -1954 deletions

## [0.1.0] - 2025-11-06

### Added
- 🦉 Initial release of Strix
- 🌐 Web-based user interface for camera stream discovery
- 🔍 Automatic RTSP stream discovery for IP cameras
- 📹 Support for multiple camera manufacturers
- 🎯 ONVIF device discovery and PTZ endpoint detection
- 🔐 Credential embedding in stream URLs
- 📊 Camera model database with autocomplete search
- 🎨 Modern, responsive UI with purple owl logo
- ⚙️ Configuration export for Go2RTC and Frigate
- 🔄 Dual-stream support with optional sub-stream selection
- 📡 Server-Sent Events (SSE) for real-time discovery progress
- 🚀 RESTful API for camera search and stream discovery
- 📦 Cross-platform support (Linux, Windows, macOS)
- 🏗️ Built with Go for high performance

### Features
- **Web Interface**: Clean, intuitive UI for camera configuration
- **Stream Discovery**: Automatically finds working RTSP streams
- **ONVIF Support**: Discovers ONVIF devices and PTZ capabilities
- **Multi-Platform**: Binaries for Linux (amd64, arm64, arm/v7), Windows, and macOS
- **Easy Integration**: Export configs for popular NVR systems

[0.1.0]: https://github.com/eduard256/Strix/releases/tag/v0.1.0

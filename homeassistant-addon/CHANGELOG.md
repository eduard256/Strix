# Changelog

All notable changes to this Home Assistant add-on will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.1] - 2025-11-17

### Fixed
- GitHub Actions permissions for publishing Docker images to ghcr.io
- Added `packages: write` permission to build job
- Added `contents: write` permission to update-repository job

## [1.0.0] - 2025-01-15

### Added
- Initial release of Strix Home Assistant Add-on
- Support for aarch64, amd64, and armv7 architectures
- Web UI integration with Home Assistant panel
- Ingress support for seamless integration
- Configurable port and logging options
- Strict validation mode toggle
- Health check monitoring
- Comprehensive documentation
- Multi-arch Docker builds via GitHub Actions
- Automatic updates through Home Assistant Supervisor

### Features
- 3,600+ camera models in database
- ONVIF discovery support
- Real-time stream discovery via SSE
- RESTful API for automation
- Fuzzy search for camera models
- Multiple stream protocol support (RTSP, HTTP, MJPEG, JPEG)
- FFProbe integration for stream validation
- Concurrent stream testing with worker pool
- Camera database with popular stream patterns

### Security
- Runs as non-root user (UID 1000)
- Minimal Alpine-based container
- No credential storage
- Local network only operation
- Read-only filesystem where possible

### Documentation
- Complete installation guide
- Configuration reference
- API documentation
- Troubleshooting guide
- Integration examples for HA, go2rtc, and Frigate

## [Unreleased]

### Planned
- Auto-discovery integration with Home Assistant
- Automatic camera entity creation
- go2rtc configuration generator
- Frigate configuration generator
- ONVIF event monitoring
- Motion detection API
- Camera snapshot gallery
- Network scanner for bulk discovery
- Custom camera database additions

---

**Full Changelog**: https://github.com/eduard256/Strix/commits/main/homeassistant-addon

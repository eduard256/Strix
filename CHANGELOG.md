# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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

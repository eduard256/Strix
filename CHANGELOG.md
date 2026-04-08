# Changelog

## [2.1.0] - 2026-04-08

### Added
- ONVIF protocol support: auto-discovery via unicast WS-Discovery, stream resolution through ONVIF profiles
- ONVIF probe detector: detects ONVIF cameras during network probe (4-7ms response time, no auth required)
- ONVIF camera page (onvif.html): credentials form with option to also test popular stream patterns
- ONVIF stream handler: resolves all camera profiles, tests each via RTSP, returns paired results (onvif:// + rtsp://) with shared screenshots
- Design system reference (design-system.html) with all UI components documented

### Changed
- ONVIF has highest probe priority (above HomeKit and Standard)
- JPEG-only streams (no H264/H265) are classified as Alternative in test results
- HomeKit page redesigned: Apple HomeKit logo, centered layout, floating back button
- Hardened create.html against undefined/null URL values in query parameters

## [2.0.0] - 2025-04-05

### Added
- Complete rewrite as a single Go binary with modular architecture
- DVRIP protocol support
- RTMP protocol support
- Bubble protocol support
- HTTP/HTTPS protocol support for snapshots and streams
- Direct stream URL input in web UI
- Frigate config proxy with auto-discovery via HA Supervisor API
- Frigate connectivity check endpoint
- go2rtc module with auto-discovery
- Network probe system: port scanning, ICMP ping, ARP/OUI lookup, mDNS/HomeKit detection, HTTP probing
- Camera stream tester with automatic JPEG screenshot capture and resolution extraction
- Frigate config generator from camera database
- Web UI pages: search, test, config, URLs, go2rtc streams, HomeKit
- SQLite camera database loaded from external StrixCamDB repository
- Universal Linux installer script with Docker/Compose auto-setup
- In-memory log viewer API endpoint
- Dockerfile with multi-stage build and healthcheck

### Fixed
- Screenshot URL path: removed leading slash
- Credentials with special characters are now URL-encoded in stream URLs
- Credentials no longer leak in debug logs

### Changed
- Version is now injected at build time via ldflags (no hardcoded version in source)
- Pure Go build with no CGO dependency (switched from mattn/go-sqlite3 to modernc.org/sqlite)
- Port is always included in URL for protocols with raw TCP dial
- Structured logging with zerolog, separate from human-readable output

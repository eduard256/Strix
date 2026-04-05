# Changelog

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

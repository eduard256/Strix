# 🦉 Strix - Smart IP Camera Stream Discovery System

![Strix Demo](assets/main.gif?v=2)

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![API Version](https://img.shields.io/badge/API-v1-green.svg)](https://github.com/eduard256/Strix)

Strix is an intelligent IP camera stream discovery system that acts as a bridge between users and streaming servers like go2rtc. It automatically discovers and validates camera streams, eliminating the need for manual URL configuration.

## 🎯 Features

- **Intelligent Camera Search**: Fuzzy search across 3,600+ camera models
- **Automatic Stream Discovery**: ONVIF, database patterns, and popular URL detection
- **Real-time Updates**: Server-Sent Events (SSE) for live discovery progress
- **Universal Protocol Support**: RTSP, HTTP, MJPEG, JPEG snapshots, and more
- **Smart URL Building**: Automatic placeholder replacement and authentication handling
- **Concurrent Testing**: Fast parallel stream validation with ffprobe
- **Memory Efficient**: Streaming JSON parsing for large camera databases
- **API-First Design**: RESTful API with comprehensive documentation

## 🚀 Quick Start

### Docker (Recommended)

```bash
# Using Docker Compose (recommended)
docker-compose up -d

# Or using Docker directly
docker run -d \
  --name strix \
  -p 4567:4567 \
  eduard256/strix:latest

# Access at http://localhost:4567
```

See [Docker documentation](DOCKER.md) for more options.

### Build from Source

Prerequisites:
- Go 1.21 or higher
- ffprobe (optional, for enhanced stream validation)

```bash
# Clone the repository
git clone https://github.com/eduard256/Strix
cd strix

# Install dependencies
make deps

# Build the application
make build

# Run the application
make run

# The server will start on http://localhost:4567
# Open your browser and navigate to http://localhost:4567
```

## 📡 API Endpoints

### Health Check
```bash
GET /api/v1/health
```

### Camera Search
```bash
POST /api/v1/cameras/search

{
  "query": "zosi zg23213m",
  "limit": 10
}
```

### Stream Discovery (SSE)
```bash
POST /api/v1/streams/discover

{
  "target": "192.168.1.100",      # IP or stream URL
  "model": "zosi zg23213m",        # Optional camera model
  "username": "admin",             # Optional
  "password": "password",          # Optional
  "timeout": 240,                  # Seconds (default: 240)
  "max_streams": 10,               # Maximum streams to find
  "channel": 0                     # For NVR systems
}
```

## 🔍 How It Works

1. **Camera Search**: Intelligent fuzzy matching across brand and model database
2. **URL Collection**: Combines ONVIF discovery, model-specific patterns, and popular URLs
3. **Stream Validation**: Concurrent testing using ffprobe and HTTP requests
4. **Real-time Updates**: SSE streams provide instant feedback on discovered streams
5. **Smart Filtering**: Deduplicates URLs and prioritizes working streams

## 📁 Project Structure

```
strix/
├── cmd/strix/           # Application entry point
├── internal/            # Private application code
│   ├── api/            # HTTP handlers and routing
│   ├── camera/         # Camera database and discovery
│   │   ├── database/   # Database loading and search
│   │   ├── discovery/  # ONVIF and stream discovery
│   │   └── stream/     # URL building and validation
│   ├── config/         # Configuration management
│   └── models/         # Data structures
├── pkg/                # Public packages
│   └── sse/           # Server-Sent Events
├── data/              # Camera database (3,600+ models)
│   ├── brands/        # Brand-specific JSON files
│   ├── popular_stream_patterns.json
│   └── query_parameters.json
└── go.mod
```

## 🛠️ Configuration

Strix can be configured via `strix.yaml` file or environment variables.

### Configuration File (strix.yaml)

Create a `strix.yaml` file in the same directory as the binary:

```yaml
# API Server Configuration
api:
  listen: ":4567"  # Format: ":port" or "host:port"
```

Examples:
```yaml
api:
  listen: ":4567"           # All interfaces, port 4567 (default)
  # listen: "127.0.0.1:4567"  # Localhost only
  # listen: ":8080"           # Custom port
```

### Environment Variables

Environment variables override config file values:

```bash
STRIX_API_LISTEN=":4567"    # Server listen address (overrides strix.yaml)
STRIX_LOG_LEVEL=info        # Log level: debug, info, warn, error
STRIX_LOG_FORMAT=json       # Log format: json, text
```

### Configuration Priority

1. **Environment variable** `STRIX_API_LISTEN` (highest priority)
2. **Config file** `strix.yaml`
3. **Default value** `:4567` (lowest priority)

### Quick Start with Custom Port

```bash
# Using environment variable
STRIX_API_LISTEN=":8080" ./strix

# Or using config file
cp strix.yaml.example strix.yaml
# Edit strix.yaml, then:
./strix
```

## 📊 Camera Database

The system includes a comprehensive database of camera models:

- **3,600+ camera brands**
- **150+ popular stream patterns**
- **258 query parameter variations**
- **Automatic placeholder replacement**

## 🔧 Development

```bash
# Run tests
make test

# Format code
make fmt

# Run linter
make lint

# Build for all platforms
make build-all

# Development mode with live reload
make dev
```

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🙏 Acknowledgments

- Camera database sourced from ispyconnect.com
- Inspired by go2rtc project
- Built with Go and Chi router

---

Made with ❤️ for the home automation community
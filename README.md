# 🦉 Strix - Smart IP Camera Stream Discovery System

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![API Version](https://img.shields.io/badge/API-v1-green.svg)](https://github.com/strix-project/strix)

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

### Prerequisites

- Go 1.21 or higher
- ffprobe (optional, for enhanced stream validation)

### Installation

```bash
# Clone the repository
git clone https://github.com/strix-project/strix
cd strix

# Install dependencies
make deps

# Build the application
make build

# Run the application
make run
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

Environment variables:

```bash
STRIX_HOST=0.0.0.0          # Server host (default: 0.0.0.0)
STRIX_PORT=8080             # Server port (default: 8080)
STRIX_LOG_LEVEL=info        # Log level: debug, info, warn, error
STRIX_LOG_FORMAT=json       # Log format: json, text
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
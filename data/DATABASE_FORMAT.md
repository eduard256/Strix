# 📹 IoT2mqtt Camera Database Format Specification

**Version:** 1.0.0
**Last Updated:** 2025-10-17

---

## 🎯 Overview

The camera database is a collection of JSON files containing URL patterns and connection details for IP cameras from various manufacturers. This format is designed to be:

- **Universal**: Works with any IP camera brand
- **Extensible**: Easy to add new models and protocols
- **Human-readable**: Simple JSON structure
- **Parseable**: Straightforward for automated tools

---

## 📁 Directory Structure

```
connectors/cameras/data/brands/
├── index.json           # Master list of all brands
├── d-link.json          # D-Link camera models
├── hikvision.json       # Hikvision camera models
├── dahua.json           # Dahua camera models
├── axis.json            # Axis camera models
└── ...                  # Additional brands
```

---

## 📋 File Formats

### 1. **index.json** - Brand Directory

Lists all available camera brands with metadata.

```json
[
  {
    "value": "d-link",
    "label": "D-Link",
    "models_count": 250,
    "entries_count": 85,
    "logo": "/assets/brands/d-link.svg"
  },
  {
    "value": "hikvision",
    "label": "Hikvision",
    "models_count": 320,
    "entries_count": 95,
    "logo": "/assets/brands/hikvision.svg"
  }
]
```

**Fields:**
- `value` (string, required): Brand identifier (lowercase, URL-safe)
- `label` (string, required): Display name
- `models_count` (integer): Total number of camera models
- `entries_count` (integer): Number of URL pattern entries
- `logo` (string, optional): Path to brand logo

---

### 2. **{brand}.json** - Brand Camera Database

Contains all URL patterns and connection details for a specific brand.

```json
{
  "brand": "D-Link",
  "brand_id": "d-link",
  "last_updated": "2025-10-17",
  "source": "ispyconnect.com",
  "website": "https://www.dlink.com",
  "entries": [
    {
      "models": ["DCS-930L", "DCS-930LB", "DCS-930LB1"],
      "type": "FFMPEG",
      "protocol": "rtsp",
      "port": 554,
      "url": "live3.sdp",
      "notes": "Main HD stream"
    },
    {
      "models": ["DCS-930L", "DCS-932L"],
      "type": "MJPEG",
      "protocol": "http",
      "port": 80,
      "url": "video.cgi?resolution=VGA",
      "notes": "Medium quality fallback"
    }
  ]
}
```

**Root Fields:**
- `brand` (string, required): Brand display name
- `brand_id` (string, required): Brand identifier (must match filename)
- `last_updated` (string, ISO 8601 date): When database was last updated
- `source` (string): Where the data came from (e.g., "ispyconnect.com")
- `website` (string, optional): Manufacturer's official website
- `entries` (array, required): List of URL pattern entries

---

### 3. **Entry Object** - URL Pattern Entry

Each entry represents a specific URL pattern that works for one or more camera models.

```json
{
  "models": ["DCS-930L", "DCS-930LB", "DCS-930LB1"],
  "type": "FFMPEG",
  "protocol": "rtsp",
  "port": 554,
  "url": "live3.sdp",
  "auth_required": true,
  "notes": "Main HD stream with audio"
}
```

**Fields:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `models` | array[string] | ✅ Yes | List of camera model names/numbers this URL works for |
| `type` | string | ✅ Yes | Stream type: `FFMPEG`, `MJPEG`, `JPEG`, `VLC`, `H264` |
| `protocol` | string | ✅ Yes | Protocol: `rtsp`, `http`, `https` |
| `port` | integer | ✅ Yes | Port number (554 for RTSP, 80/443 for HTTP) |
| `url` | string | ✅ Yes | URL path (without protocol/host/port) |
| `auth_required` | boolean | No | Whether authentication is needed (default: true) |
| `notes` | string | No | Human-readable description |

---

## 🔧 URL Template Variables

URL paths support the following template variables:

| Variable | Description | Example |
|----------|-------------|---------|
| `{username}` | Camera username | `admin` |
| `{password}` | Camera password | `12345` |
| `{ip}` | Camera IP address | `192.168.1.100` |
| `{port}` | Port number | `554` |
| `{channel}` | Camera channel (for DVRs) | `1` |
| `{width}` | Video width | `1920` |
| `{height}` | Video height | `1080` |

**Example:**
```
Template: rtsp://{username}:{password}@{ip}:{port}/live3.sdp
Result:   rtsp://admin:12345@192.168.1.100:554/live3.sdp
```

---

## 📊 Stream Types

### FFMPEG (Recommended)
- **Protocol**: RTSP, HTTP
- **Format**: H.264, H.265
- **Use case**: High-quality video with audio
- **Priority**: 🥇 First choice

### MJPEG
- **Protocol**: HTTP
- **Format**: Motion JPEG
- **Use case**: Medium quality, wide compatibility
- **Priority**: 🥈 Second choice

### JPEG
- **Protocol**: HTTP
- **Format**: Still images
- **Use case**: Snapshot-only cameras or fallback
- **Priority**: 🥉 Last resort

### VLC
- **Protocol**: RTSP, HTTP
- **Format**: Various (VLC-specific)
- **Use case**: Compatibility with VLC player

---

## 🎯 Priority Order for Testing

When testing multiple URLs for a camera model, use this priority:

1. **RTSP (type="FFMPEG")** - Best quality, supports audio
2. **HTTP MJPEG** - Good compatibility
3. **HTTP JPEG** - Snapshot fallback

**Example:**
```python
def get_urls_for_model(brand_data, model_name):
    entries = [e for e in brand_data["entries"] if model_name in e["models"]]

    # Sort by priority
    priority = {"FFMPEG": 1, "MJPEG": 2, "JPEG": 3, "VLC": 4}
    entries.sort(key=lambda e: priority.get(e["type"], 99))

    return entries
```

---

## 🔍 Search and Lookup

### By Brand
```python
# Load brand file
with open(f"data/brands/{brand_id}.json") as f:
    brand_data = json.load(f)
```

### By Model
```python
# Find all entries for a specific model
def find_model_entries(brand_data, model_name):
    return [
        entry for entry in brand_data["entries"]
        if model_name.upper() in [m.upper() for m in entry["models"]]
    ]
```

### Fuzzy Search
```python
# Search across all models (case-insensitive, partial match)
def search_model(brand_data, query):
    query = query.upper()
    results = []
    for entry in brand_data["entries"]:
        if any(query in model.upper() for model in entry["models"]):
            results.append(entry)
    return results
```

---

## 🌐 URL Construction

### RTSP URL
```python
def build_rtsp_url(entry, ip, username, password):
    return f"rtsp://{username}:{password}@{ip}:{entry['port']}/{entry['url']}"

# Example:
# rtsp://admin:12345@192.168.1.100:554/live3.sdp
```

### HTTP URL
```python
def build_http_url(entry, ip, username, password):
    protocol = entry["protocol"]  # "http" or "https"
    return f"{protocol}://{username}:{password}@{ip}:{entry['port']}/{entry['url']}"

# Example:
# http://admin:12345@192.168.1.100:80/video.cgi?resolution=VGA
```

### With Template Variables
```python
def build_url(entry, ip, username, password, **kwargs):
    url_path = entry["url"]

    # Replace template variables
    replacements = {
        "username": username,
        "password": password,
        "ip": ip,
        "port": str(entry["port"]),
        **kwargs  # Additional variables (channel, width, height, etc.)
    }

    for key, value in replacements.items():
        url_path = url_path.replace(f"{{{key}}}", value)

    # Build full URL
    if entry["protocol"] == "rtsp":
        return f"rtsp://{username}:{password}@{ip}:{entry['port']}/{url_path}"
    else:
        return f"{entry['protocol']}://{username}:{password}@{ip}:{entry['port']}/{url_path}"
```

---

## ✅ Validation Rules

### Entry Validation
```python
def validate_entry(entry):
    # Required fields
    assert "models" in entry and isinstance(entry["models"], list)
    assert len(entry["models"]) > 0
    assert "type" in entry and entry["type"] in ["FFMPEG", "MJPEG", "JPEG", "VLC", "H264"]
    assert "protocol" in entry and entry["protocol"] in ["rtsp", "http", "https"]
    assert "port" in entry and isinstance(entry["port"], int)
    assert "url" in entry and isinstance(entry["url"], str)

    # Port ranges
    assert 1 <= entry["port"] <= 65535

    # Common ports check
    if entry["protocol"] == "rtsp":
        assert entry["port"] in [554, 8554, 7447]  # Common RTSP ports
    elif entry["protocol"] == "http":
        assert entry["port"] in [80, 8080, 8000, 8081]  # Common HTTP ports
```

---

## 📝 Naming Conventions

### Brand IDs
- **Format**: lowercase, kebab-case
- **Examples**: `d-link`, `hikvision`, `tp-link`
- **Invalid**: `D-Link`, `D_Link`, `dlink`

### Model Names
- **Format**: UPPERCASE with hyphens (as manufacturer specifies)
- **Examples**: `DCS-930L`, `DS-2CD2142FWD-I`, `IPC-HFW1230S`
- **Keep original**: Don't normalize or change manufacturer names

### Protocol Values
- `rtsp` - RTSP protocol
- `http` - HTTP protocol
- `https` - HTTPS protocol
- **Invalid**: `RTSP`, `Http`, `tcp`

### Type Values
- `FFMPEG` - H.264/H.265 streams (RTSP or HTTP)
- `MJPEG` - Motion JPEG streams
- `JPEG` - Still image snapshots
- `VLC` - VLC-specific streams

---

## 🔄 Versioning and Updates

### Version Format
```json
{
  "brand": "D-Link",
  "brand_id": "d-link",
  "database_version": "1.2.0",
  "last_updated": "2025-10-17T14:30:00Z",
  "entries": [...]
}
```

### Update Policy
- **Patch** (1.0.x): Add new models to existing entries
- **Minor** (1.x.0): Add new URL patterns/entries
- **Major** (x.0.0): Breaking changes to structure

---

## 📚 Examples

### Complete Brand File Example

**foscam.json:**
```json
{
  "brand": "Foscam",
  "brand_id": "foscam",
  "last_updated": "2025-10-17",
  "source": "ispyconnect.com",
  "website": "https://www.foscam.com",
  "entries": [
    {
      "models": ["FI9821P", "FI9826P", "FI9821W"],
      "type": "FFMPEG",
      "protocol": "rtsp",
      "port": 554,
      "url": "videoMain",
      "notes": "Main stream HD"
    },
    {
      "models": ["FI9821P", "FI9826P"],
      "type": "FFMPEG",
      "protocol": "rtsp",
      "port": 554,
      "url": "videoSub",
      "notes": "Sub stream SD"
    },
    {
      "models": ["FI9821P", "FI9826P", "FI9821W", "C1"],
      "type": "MJPEG",
      "protocol": "http",
      "port": 88,
      "url": "cgi-bin/CGIStream.cgi?cmd=GetMJStream&usr={username}&pwd={password}",
      "notes": "MJPEG fallback"
    },
    {
      "models": ["FI9821P", "C1", "C2"],
      "type": "JPEG",
      "protocol": "http",
      "port": 88,
      "url": "cgi-bin/CGIProxy.fcgi?cmd=snapPicture2&usr={username}&pwd={password}",
      "notes": "Snapshot"
    }
  ]
}
```

---

## 🛠️ Tools and Scripts

### Parser Script (Python)
```python
# scripts/parse_ispyconnect.py
import requests
from bs4 import BeautifulSoup
import json

def parse_brand_page(brand_id):
    url = f"https://www.ispyconnect.com/camera/{brand_id}"
    response = requests.get(url)
    soup = BeautifulSoup(response.text, 'html.parser')

    table = soup.find('table', class_='table-striped')
    entries = []

    for row in table.find_all('tr')[1:]:  # Skip header
        cols = row.find_all('td')
        if len(cols) < 4:
            continue

        models_text = cols[0].get_text()
        models = [m.strip() for m in models_text.split(',')]

        entry = {
            "models": models,
            "type": cols[1].get_text(strip=True),
            "protocol": cols[2].get_text(strip=True).replace('://', ''),
            "port": int(row.get('data-port', 0)),
            "url": cols[3].get_text(strip=True)
        }

        entries.append(entry)

    return {
        "brand": brand_id.title(),
        "brand_id": brand_id,
        "last_updated": "2025-10-17",
        "source": "ispyconnect.com",
        "entries": entries
    }
```

### Validator Script
```python
# scripts/validate_database.py
import json
import os

def validate_brand_file(filepath):
    with open(filepath) as f:
        data = json.load(f)

    # Check required fields
    assert "brand" in data
    assert "brand_id" in data
    assert "entries" in data

    # Validate each entry
    for i, entry in enumerate(data["entries"]):
        assert "models" in entry, f"Entry {i} missing models"
        assert "type" in entry, f"Entry {i} missing type"
        assert "protocol" in entry, f"Entry {i} missing protocol"
        assert "port" in entry, f"Entry {i} missing port"
        assert "url" in entry, f"Entry {i} missing url"

    print(f"✅ {filepath} is valid")

# Run validation
for file in os.listdir('data/brands/'):
    if file.endswith('.json') and file != 'index.json':
        validate_brand_file(f'data/brands/{file}')
```

---

## 📄 License and Attribution

- **Source**: ispyconnect.com camera database
- **Usage**: Free for IoT2mqtt project
- **Attribution**: Must credit ispyconnect.com as data source
- **Updates**: Community-contributed updates welcome

---

## 🤝 Contributing

To add or update camera models:

1. Follow the JSON format specification
2. Validate using `scripts/validate_database.py`
3. Test URLs with real cameras when possible
4. Submit pull request with changes

---

## 📞 Support

For questions about the database format:
- GitHub Issues: https://github.com/your-repo/issues
- Documentation: https://docs.your-project.com

---

**End of Specification**

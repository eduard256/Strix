# Configuration Generators Documentation

This document describes how Strix generates configurations for go2rtc and Frigate with support for main and sub streams.

## Go2RTC Generator (`webui/web/js/config-generators/go2rtc/index.js`)

### Purpose
Generates YAML configurations for go2rtc based on discovered camera streams. Supports both single stream and dual-stream (main + sub) configurations.

### Stream Naming Convention

**Format:** `<ip_with_underscores>_<suffix>`

**Examples:**
- Main: `192.168.1.100` → `192_168_1_100_main`
- Sub: `192.168.1.100` → `192_168_1_100_sub`
- Main: `10.0.20.112` → `10_0_20_112_main`
- Sub: `10.0.20.112` → `10_0_20_112_sub`

### Single Stream Configuration

When only a main stream is selected:

```yaml
streams:
  '192_168_1_100_main':
    - rtsp://admin:password@192.168.1.100/stream1
```

### Dual Stream Configuration (Main + Sub)

When both main and sub streams are selected:

```yaml
streams:
  '192_168_1_100_main':
    - rtsp://admin:password@192.168.1.100/live/main

  '192_168_1_100_sub':
    - rtsp://admin:password@192.168.1.100/live/sub
```

### Logic by Stream Type

#### 1. **JPEG Snapshots** (Special Case)
Static JPEG images require conversion to video stream using FFmpeg.

**Generated Config:**
```yaml
streams:
  '10_0_20_112_main':
    - exec:ffmpeg -loglevel quiet -f image2 -loop 1 -framerate 10 -i http://admin:pass@10.0.20.112/snapshot.jpg -c:v libx264 -preset ultrafast -tune zerolatency -g 20 -f rtsp {output}
```

**Parameters:**
- `-f image2 -loop 1`: Loop single image
- `-framerate 10`: 10 fps output
- `-c:v libx264`: H264 encoding
- `-preset ultrafast -tune zerolatency`: Low latency
- `-g 20`: GOP size for keyframes
- `-f rtsp {output}`: Output to RTSP (go2rtc internal)

#### 2. **All Other Formats** (Direct Pass-through)
For RTSP, MJPEG, HLS, HTTP-FLV, HTTP-TS, RTMP - use direct URL.
go2rtc has native support for these formats.

**Supported Formats:**
- **RTSP** (`rtsp://`) - Direct support
- **RTMP** (`rtmp://`) - Direct support
- **MJPEG** (`http://...mjpeg`) - Direct support
- **HLS** (`http://...m3u8`) - Direct support
- **HTTP-FLV** (`http://...flv`) - Direct support
- **HTTP-TS** (`http://...ts`) - Direct support

#### 3. **ONVIF Device Endpoints**
ONVIF URLs are converted to `onvif://` format:

```yaml
streams:
  '192_168_1_100_main':
    - onvif://admin:password@192.168.1.100:80
```

## Frigate Generator (`webui/web/js/config-generators/frigate/index.js`)

### Purpose
Generates unified Frigate + Go2RTC YAML configurations with intelligent stream routing for optimal performance.

### Key Features

- **Motion-based recording**: Records only when motion is detected
- **Object detection**: Tracks person, car, cat, dog
- **Smart stream routing**:
  - If sub stream exists → detect on sub (low CPU), record on main (quality)
  - If no sub stream → detect and record on main

### Benefits of Dual-Stream Setup

✅ **Lower CPU usage**: Detection runs on lower resolution sub stream
✅ **Better quality**: Recording uses high resolution main stream
✅ **Single connection per camera**: Go2RTC multiplexes streams
✅ **Optimal performance**: Each task uses appropriate stream quality

### Single Stream Configuration (Main Only)

When only main stream is selected, it handles both detection and recording:

```yaml
mqtt:
  enabled: false

# Global Recording Settings
record:
  enabled: true
  retain:
    days: 7
    mode: motion  # Record only on motion detection

# Go2RTC Configuration (Frigate built-in)
go2rtc:
  streams:
    '192_168_1_100_main':
      - rtsp://admin:password@192.168.1.100/stream1

# Frigate Camera Configuration
cameras:
  camera_192_168_1_100:
    ffmpeg:
      inputs:
        - path: rtsp://127.0.0.1:8554/192_168_1_100_main
          input_args: preset-rtsp-restream
          roles:
            - detect
            - record
    objects:
      track:
        - person
        - car
        - cat
        - dog
    record:
      enabled: true

version: 0.16-0
```

### Dual Stream Configuration (Main + Sub)

When both streams are selected, detection uses sub stream and recording uses main stream:

```yaml
mqtt:
  enabled: false

# Global Recording Settings
record:
  enabled: true
  retain:
    days: 7
    mode: motion  # Record only on motion detection

# Go2RTC Configuration (Frigate built-in)
go2rtc:
  streams:
    '192_168_1_100_main':
      - rtsp://admin:password@192.168.1.100/live/main

    '192_168_1_100_sub':
      - rtsp://admin:password@192.168.1.100/live/sub

# Frigate Camera Configuration
cameras:
  camera_192_168_1_100:
    ffmpeg:
      inputs:
        - path: rtsp://127.0.0.1:8554/192_168_1_100_sub
          input_args: preset-rtsp-restream
          roles:
            - detect
        - path: rtsp://127.0.0.1:8554/192_168_1_100_main
          input_args: preset-rtsp-restream
          roles:
            - record
    live:
      streams:
        Main Stream: 192_168_1_100_main    # HD для просмотра
        Sub Stream: 192_168_1_100_sub      # Низкое разрешение (опционально)
    objects:
      track:
        - person
        - car
        - cat
        - dog
    record:
      enabled: true

version: 0.16-0
```

### Why Sub Stream for Detection?

✅ **CPU Efficiency**: Processing lower resolution (typically 352x288 or 640x480) instead of HD/4K
✅ **Faster Inference**: ML model runs faster on smaller resolution
✅ **Sufficient Accuracy**: Object detection doesn't need Full HD or 4K
✅ **Quality Recording**: Main stream at full resolution (HD/4K) saved to disk
✅ **Auto-detection**: Frigate automatically detects stream resolution

### Camera Naming Convention

**Format:** `camera_<ip_with_underscores>`

**Examples:**
- `192.168.1.100` → `camera_192_168_1_100`
- `10.0.20.112` → `camera_10_0_20_112`
- `camera.local` → `camera_camera_local`

### Object Detection

The generator includes basic object detection for common use cases:

- **person** - Human detection
- **car** - Vehicle detection
- **cat** - Cat detection
- **dog** - Dog detection

To add more objects, edit the generated config and add items from [Frigate's object list](https://docs.frigate.video/configuration/objects/).

### Recording Mode

**Mode: `motion`** (Default)
- Records only when motion is detected
- Saves disk space
- May miss the start of an event

**To enable 24/7 recording**, change to:
```yaml
record:
  enabled: true
  retain:
    days: 7
    mode: all  # Continuous recording
```

## Workflow

```
Stream Discovery
      ↓
User selects Main Stream
      ↓
Config generated (main only)
      ↓
┌─────────────────────┐
│ User clicks         │
│ "Add Sub Stream"    │
└──────────┬──────────┘
           ↓
User selects Sub Stream from existing results
           ↓
Config regenerated (main + sub with optimized routing)
```

## Key Principles

1. **No additional scanning**: Sub stream is selected from already discovered streams
2. **Intelligent routing**: Sub for detect, main for record (when both available)
3. **Simplicity first**: Use direct URLs whenever possible
4. **Native support**: Leverage go2rtc's built-in format support
5. **Special cases only**: Only use exec:ffmpeg for JPEG snapshots
6. **Motion-based recording**: Save disk space by default

## Benefits of This Approach

✅ **Better performance**: Optimal stream selection for each task
✅ **Lower CPU usage**: Detection on lower resolution when sub stream available
✅ **Quality recordings**: Full resolution saved to disk
✅ **User flexibility**: Optional sub stream - not required
✅ **No re-scanning**: Reuses already discovered streams
✅ **Disk space efficiency**: Motion-based recording by default

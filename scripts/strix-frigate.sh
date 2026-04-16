#!/usr/bin/env bash
# =============================================================================
# Strix -- strix-frigate.sh (worker)
#
# Deploys Strix + Frigate together via Docker Compose.
# Generates docker-compose.yml dynamically (devices depend on hardware),
# creates .env, pulls images, starts containers, runs healthchecks.
#
# Protocol:
#   - Every action is reported as a single-line JSON to stdout.
#   - Types: check, ok, miss, install, error, done
#   - Last line is always: {"type":"done","ok":true,...} or {"type":"done","ok":false,"error":"..."}
#   - Exit code: 0 = success, 1 = failure.
#
# Parameters (all optional):
#   --port PORT              Strix listen port (default: 4567)
#   --tag TAG                Strix image tag (default: latest)
#   --log-level LEVEL        Log level: debug, info, warn, error, trace
#   --go2rtc-url URL         External go2rtc URL
#   --shm-size SIZE          Frigate shm_size (default: 512mb)
#   --frigate-tag TAG        Frigate image tag (default: stable)
#   --dir DIR                Working directory (default: /opt/strix)
#
# Usage:
#   bash scripts/strix-frigate.sh
#   bash scripts/strix-frigate.sh --port 4567 --frigate-tag stable-tensorrt
# =============================================================================

set -uo pipefail

# ---------------------------------------------------------------------------
# Defaults
# ---------------------------------------------------------------------------
STRIX_PORT="4567"
STRIX_TAG="latest"
STRIX_LOG_LEVEL=""
STRIX_GO2RTC_URL=""
FRIGATE_SHM="512mb"
FRIGATE_TAG="stable"
STRIX_DIR="/opt/strix"
STRIX_IMAGE="eduard256/strix"
FRIGATE_IMAGE="ghcr.io/blakeblackshear/frigate"

# Detected devices (populated by detect_devices)
DEVICES=()
DEVICE_NAMES=()

# ---------------------------------------------------------------------------
# Parse CLI arguments
# ---------------------------------------------------------------------------
while [[ $# -gt 0 ]]; do
    case "$1" in
        --port)        STRIX_PORT="$2";     shift 2 ;;
        --tag)         STRIX_TAG="$2";      shift 2 ;;
        --log-level)   STRIX_LOG_LEVEL="$2"; shift 2 ;;
        --go2rtc-url)  STRIX_GO2RTC_URL="$2"; shift 2 ;;
        --shm-size)    FRIGATE_SHM="$2";    shift 2 ;;
        --frigate-tag) FRIGATE_TAG="$2";    shift 2 ;;
        --dir)         STRIX_DIR="$2";      shift 2 ;;
        *)             shift ;;
    esac
done

# ---------------------------------------------------------------------------
# JSON helpers
# ---------------------------------------------------------------------------
emit() {
    local type="$1"
    local msg="$2"
    local data="${3:-}"

    msg="${msg//\\/\\\\}"
    msg="${msg//\"/\\\"}"

    if [[ -n "$data" ]]; then
        printf '{"type":"%s","msg":"%s","data":%s}\n' "$type" "$msg" "$data"
    else
        printf '{"type":"%s","msg":"%s"}\n' "$type" "$msg"
    fi
}

emit_done_ok() {
    # Accepts raw JSON data string
    local data="$1"
    printf '{"type":"done","ok":true,"data":%s}\n' "$data"
    exit 0
}

emit_done_fail() {
    local error="$1"
    error="${error//\\/\\\\}"
    error="${error//\"/\\\"}"
    printf '{"type":"done","ok":false,"error":"%s"}\n' "$error"
    exit 1
}

# ---------------------------------------------------------------------------
# Detect LAN IP
# ---------------------------------------------------------------------------
detect_lan_ip() {
    local ip=""
    ip=$(ip route get 1.1.1.1 2>/dev/null | grep -oP 'src \K\S+' | head -1)
    [[ -z "$ip" ]] && ip=$(hostname -I 2>/dev/null | awk '{print $1}')
    [[ -z "$ip" ]] && ip="localhost"
    echo "$ip"
}

# ---------------------------------------------------------------------------
# 1. Working directory
# ---------------------------------------------------------------------------
setup_dir() {
    emit "check" "Checking working directory ${STRIX_DIR}"

    if [[ -d "$STRIX_DIR" ]]; then
        emit "ok" "Directory exists: ${STRIX_DIR}"
    else
        emit "install" "Creating directory ${STRIX_DIR}"
        if mkdir -p "$STRIX_DIR" 2>/dev/null; then
            emit "ok" "Directory created: ${STRIX_DIR}"
        else
            emit "error" "Failed to create directory ${STRIX_DIR}"
            emit_done_fail "Cannot create ${STRIX_DIR}"
        fi
    fi

    # Frigate subdirectories
    emit "check" "Checking Frigate directories"

    mkdir -p "${STRIX_DIR}/frigate/config" 2>/dev/null
    mkdir -p "${STRIX_DIR}/frigate/storage" 2>/dev/null

    if [[ -d "${STRIX_DIR}/frigate/config" ]] && [[ -d "${STRIX_DIR}/frigate/storage" ]]; then
        emit "ok" "Frigate directories ready"
    else
        emit "error" "Failed to create Frigate directories"
        emit_done_fail "Cannot create Frigate directories"
    fi
}

# ---------------------------------------------------------------------------
# 2. Detect hardware devices
# ---------------------------------------------------------------------------
detect_devices() {
    emit "check" "Detecting hardware accelerators"

    local found=0

    # USB Coral
    emit "check" "Checking for USB Coral"
    if command -v lsusb &>/dev/null && lsusb 2>/dev/null | grep -qE "1a6e:089a|18d1:9302"; then
        DEVICES+=("/dev/bus/usb:/dev/bus/usb")
        DEVICE_NAMES+=("usb_coral")
        emit "ok" "USB Coral detected" "{\"device\":\"usb_coral\",\"path\":\"/dev/bus/usb\"}"
        found=$((found + 1))
    else
        emit "miss" "USB Coral not found"
    fi

    # PCIe Coral
    emit "check" "Checking for PCIe Coral"
    if [[ -e /dev/apex_0 ]]; then
        DEVICES+=("/dev/apex_0:/dev/apex_0")
        DEVICE_NAMES+=("pcie_coral")
        emit "ok" "PCIe Coral detected" "{\"device\":\"pcie_coral\",\"path\":\"/dev/apex_0\"}"
        found=$((found + 1))
    else
        emit "miss" "PCIe Coral not found"
    fi

    # Intel / AMD GPU
    emit "check" "Checking for Intel/AMD GPU"
    if [[ -e /dev/dri/renderD128 ]]; then
        DEVICES+=("/dev/dri:/dev/dri")
        DEVICE_NAMES+=("gpu")
        emit "ok" "GPU detected (Intel/AMD)" "{\"device\":\"gpu\",\"path\":\"/dev/dri\"}"
        found=$((found + 1))
    else
        emit "miss" "Intel/AMD GPU not found"
    fi

    # Intel NPU
    emit "check" "Checking for Intel NPU"
    if [[ -e /dev/accel ]]; then
        DEVICES+=("/dev/accel:/dev/accel")
        DEVICE_NAMES+=("intel_npu")
        emit "ok" "Intel NPU detected" "{\"device\":\"intel_npu\",\"path\":\"/dev/accel\"}"
        found=$((found + 1))
    else
        emit "miss" "Intel NPU not found"
    fi

    # Raspberry Pi 4 video
    emit "check" "Checking for Raspberry Pi video device"
    if [[ -e /dev/video11 ]]; then
        DEVICES+=("/dev/video11:/dev/video11")
        DEVICE_NAMES+=("rpi_video")
        emit "ok" "Raspberry Pi video device detected" "{\"device\":\"rpi_video\",\"path\":\"/dev/video11\"}"
        found=$((found + 1))
    else
        emit "miss" "Raspberry Pi video device not found"
    fi

    if [[ "$found" -eq 0 ]]; then
        emit "ok" "No hardware accelerators found, using CPU only"
    else
        emit "ok" "${found} hardware accelerator(s) detected"
    fi
}

# ---------------------------------------------------------------------------
# 3. Generate docker-compose.yml
# ---------------------------------------------------------------------------
generate_compose() {
    emit "check" "Generating docker-compose.yml"

    # Build devices section
    local devices_block=""
    if [[ ${#DEVICES[@]} -gt 0 ]]; then
        devices_block="    devices:"
        for dev in "${DEVICES[@]}"; do
            devices_block="${devices_block}
      - ${dev}"
        done
    fi

    # Build compose file
    cat > "${STRIX_DIR}/docker-compose.yml" <<EOF
# Strix + Frigate
# Generated by strix-frigate.sh

services:
  strix:
    container_name: strix
    image: ${STRIX_IMAGE}:\${STRIX_TAG:-latest}
    network_mode: host
    restart: unless-stopped
    env_file: .env
    depends_on:
      frigate:
        condition: service_started

  frigate:
    container_name: frigate
    image: ${FRIGATE_IMAGE}:${FRIGATE_TAG}
    privileged: true
    network_mode: host
    restart: unless-stopped
    stop_grace_period: 30s
    shm_size: "${FRIGATE_SHM}"
${devices_block}
    volumes:
      - /etc/localtime:/etc/localtime:ro
      - ./frigate/config:/config
      - ./frigate/storage:/media/frigate
      - type: tmpfs
        target: /tmp/cache
        tmpfs:
          size: 1000000000
    environment:
      FRIGATE_RTSP_PASSWORD: "password"
EOF

    emit "ok" "docker-compose.yml generated" "{\"frigate_tag\":\"${FRIGATE_TAG}\",\"shm_size\":\"${FRIGATE_SHM}\"}"
}

# ---------------------------------------------------------------------------
# 4. Generate .env
# ---------------------------------------------------------------------------
generate_env() {
    emit "check" "Generating .env configuration"

    cat > "${STRIX_DIR}/.env" <<EOF
# Strix configuration -- generated by strix-frigate.sh
STRIX_TAG=${STRIX_TAG}
STRIX_LISTEN=:${STRIX_PORT}
STRIX_FRIGATE_URL=http://localhost:5000
EOF

    emit "ok" "Frigate URL: http://localhost:5000 (internal API)"

    if [[ -n "$STRIX_GO2RTC_URL" ]]; then
        echo "STRIX_GO2RTC_URL=${STRIX_GO2RTC_URL}" >> "${STRIX_DIR}/.env"
        emit "ok" "go2rtc URL: ${STRIX_GO2RTC_URL}"
    fi

    if [[ -n "$STRIX_LOG_LEVEL" ]]; then
        echo "STRIX_LOG_LEVEL=${STRIX_LOG_LEVEL}" >> "${STRIX_DIR}/.env"
        emit "ok" "Log level: ${STRIX_LOG_LEVEL}"
    fi

    emit "ok" ".env generated (port ${STRIX_PORT})"
}

# ---------------------------------------------------------------------------
# 5. Pull images
# ---------------------------------------------------------------------------
pull_images() {
    emit "check" "Pulling Frigate image ${FRIGATE_IMAGE}:${FRIGATE_TAG} (this may take a while)"

    if docker pull "${FRIGATE_IMAGE}:${FRIGATE_TAG}" &>/dev/null; then
        emit "ok" "Frigate image pulled: ${FRIGATE_TAG}"
    else
        emit "error" "Failed to pull Frigate image ${FRIGATE_IMAGE}:${FRIGATE_TAG}"
        emit_done_fail "Frigate image pull failed"
    fi

    emit "check" "Pulling Strix image ${STRIX_IMAGE}:${STRIX_TAG}"

    if docker pull "${STRIX_IMAGE}:${STRIX_TAG}" &>/dev/null; then
        emit "ok" "Strix image pulled: ${STRIX_TAG}"
    else
        emit "error" "Failed to pull Strix image ${STRIX_IMAGE}:${STRIX_TAG}"
        emit_done_fail "Strix image pull failed"
    fi
}

# ---------------------------------------------------------------------------
# 6. Start containers
# ---------------------------------------------------------------------------
start_containers() {
    local running_frigate=false
    local running_strix=false

    docker ps --format '{{.Names}}' 2>/dev/null | grep -q '^frigate$' && running_frigate=true
    docker ps --format '{{.Names}}' 2>/dev/null | grep -q '^strix$' && running_strix=true

    if [[ "$running_frigate" == true ]] || [[ "$running_strix" == true ]]; then
        emit "check" "Existing containers found, recreating"
        if docker compose -f "${STRIX_DIR}/docker-compose.yml" up -d --force-recreate &>/dev/null; then
            emit "ok" "Containers recreated"
        else
            emit "error" "Failed to recreate containers"
            emit_done_fail "Container recreate failed"
        fi
    else
        emit "install" "Starting Frigate and Strix containers"
        if docker compose -f "${STRIX_DIR}/docker-compose.yml" up -d &>/dev/null; then
            emit "ok" "Containers started"
        else
            emit "error" "Failed to start containers"
            emit_done_fail "Container start failed"
        fi
    fi
}

# ---------------------------------------------------------------------------
# 7. Healthchecks
# ---------------------------------------------------------------------------
healthcheck_frigate() {
    emit "check" "Waiting for Frigate to respond on port 5000"

    local retries=30
    for (( i = 1; i <= retries; i++ )); do
        if curl -sf --connect-timeout 2 --max-time 3 "http://localhost:5000/api/config" &>/dev/null; then
            emit "ok" "Frigate is running on port 5000"
            return 0
        fi
        sleep 2
    done

    emit "error" "Frigate healthcheck failed after ${retries} attempts"
    emit_done_fail "Frigate healthcheck failed"
}

healthcheck_strix() {
    emit "check" "Waiting for Strix to respond on port ${STRIX_PORT}"

    local retries=15
    for (( i = 1; i <= retries; i++ )); do
        if curl -sf --connect-timeout 2 --max-time 3 "http://localhost:${STRIX_PORT}/api/health" &>/dev/null; then
            local version
            version=$(curl -sf --max-time 3 "http://localhost:${STRIX_PORT}/api" 2>/dev/null | grep -oP '"version"\s*:\s*"\K[^"]+' || echo "unknown")
            emit "ok" "Strix v${version} is running on port ${STRIX_PORT}" "{\"version\":\"${version}\"}"
            return 0
        fi
        sleep 1
    done

    emit "error" "Strix healthcheck failed after ${retries} attempts"
    emit_done_fail "Strix healthcheck failed"
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
main() {
    # 1. Working directory
    setup_dir

    # 2. Detect hardware
    detect_devices

    # 3. Generate compose (with detected devices)
    generate_compose

    # 4. Generate .env
    generate_env

    # 5. Pull images
    pull_images

    # 6. Start containers
    start_containers

    # 7. Healthchecks
    healthcheck_frigate
    healthcheck_strix

    # 8. Done -- all URLs
    local lan_ip
    lan_ip=$(detect_lan_ip)

    local strix_version
    strix_version=$(curl -sf --max-time 3 "http://localhost:${STRIX_PORT}/api" 2>/dev/null | grep -oP '"version"\s*:\s*"\K[^"]+' || echo "unknown")

    # Build device names JSON array
    local devices_json="["
    local first=true
    for name in "${DEVICE_NAMES[@]}"; do
        [[ "$first" == true ]] && first=false || devices_json="${devices_json},"
        devices_json="${devices_json}\"${name}\""
    done
    devices_json="${devices_json}]"

    emit_done_ok "{\"ip\":\"${lan_ip}\",\"strix_url\":\"http://${lan_ip}:${STRIX_PORT}\",\"strix_version\":\"${strix_version}\",\"frigate_url\":\"http://${lan_ip}:8971\",\"frigate_internal\":\"http://${lan_ip}:5000\",\"go2rtc_url\":\"http://${lan_ip}:1984\",\"frigate_tag\":\"${FRIGATE_TAG}\",\"port\":\"${STRIX_PORT}\",\"devices\":${devices_json}}"
}

main

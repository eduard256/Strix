#!/usr/bin/env bash
# =============================================================================
# Strix -- strix.sh (worker)
#
# Deploys Strix container via Docker Compose.
# Downloads docker-compose.yml from GitHub (if not already present),
# generates .env from parameters, pulls image, starts container, healthchecks.
#
# Protocol:
#   - Every action is reported as a single-line JSON to stdout.
#   - Types: check, ok, miss, install, error, done
#   - Field "msg" is always human-readable.
#   - Field "data" is optional, carries machine-readable details.
#   - Last line is always: {"type":"done","ok":true,...} or {"type":"done","ok":false,"error":"..."}
#   - Exit code: 0 = success, 1 = failure.
#
# Parameters (all optional):
#   --port PORT              Strix listen port (default: 4567)
#   --frigate-url URL        Frigate URL, e.g. http://192.168.1.50:5000
#   --go2rtc-url URL         go2rtc URL, e.g. http://192.168.1.50:1984
#   --log-level LEVEL        Log level: debug, info, warn, error, trace (default: info)
#   --tag TAG                Docker image tag (default: latest)
#   --dir DIR                Working directory (default: /opt/strix)
#
# Usage:
#   bash scripts/strix.sh
#   bash scripts/strix.sh --port 4567 --frigate-url http://192.168.1.50:5000
# =============================================================================

set -uo pipefail

# ---------------------------------------------------------------------------
# Defaults
# ---------------------------------------------------------------------------
STRIX_PORT="4567"
STRIX_FRIGATE_URL=""
STRIX_GO2RTC_URL=""
STRIX_LOG_LEVEL=""
STRIX_TAG="latest"
STRIX_DIR="/opt/strix"
COMPOSE_URL="https://raw.githubusercontent.com/eduard256/Strix/main/docker-compose.yml"
IMAGE="eduard256/strix"

# ---------------------------------------------------------------------------
# Parse CLI arguments
# ---------------------------------------------------------------------------
while [[ $# -gt 0 ]]; do
    case "$1" in
        --port)        STRIX_PORT="$2";        shift 2 ;;
        --frigate-url) STRIX_FRIGATE_URL="$2"; shift 2 ;;
        --go2rtc-url)  STRIX_GO2RTC_URL="$2";  shift 2 ;;
        --log-level)   STRIX_LOG_LEVEL="$2";   shift 2 ;;
        --tag)         STRIX_TAG="$2";         shift 2 ;;
        --dir)         STRIX_DIR="$2";         shift 2 ;;
        *)             shift ;;
    esac
done

# ---------------------------------------------------------------------------
# JSON helpers (same protocol as prepare.sh)
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

emit_done() {
    local ok="$1"
    shift

    if [[ "$ok" == "true" ]]; then
        # Remaining args are key:value pairs for data
        local data="{"
        local first=true
        while [[ $# -ge 2 ]]; do
            local key="$1" val="$2"; shift 2
            val="${val//\\/\\\\}"
            val="${val//\"/\\\"}"
            [[ "$first" == true ]] && first=false || data="${data},"
            data="${data}\"${key}\":\"${val}\""
        done
        data="${data}}"
        printf '{"type":"done","ok":true,"data":%s}\n' "$data"
        exit 0
    else
        local error="${1:-unknown}"
        error="${error//\\/\\\\}"
        error="${error//\"/\\\"}"
        printf '{"type":"done","ok":false,"error":"%s"}\n' "$error"
        exit 1
    fi
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
# Working directory
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
            emit_done "false" "Cannot create ${STRIX_DIR}"
        fi
    fi
}

# ---------------------------------------------------------------------------
# Download docker-compose.yml
# ---------------------------------------------------------------------------
download_compose() {
    emit "check" "Checking docker-compose.yml"

    if [[ -f "${STRIX_DIR}/docker-compose.yml" ]]; then
        emit "ok" "docker-compose.yml already exists"
        return
    fi

    emit "install" "Downloading docker-compose.yml from GitHub"

    if curl -fsSL "$COMPOSE_URL" -o "${STRIX_DIR}/docker-compose.yml" 2>/dev/null; then
        emit "ok" "docker-compose.yml downloaded"
    else
        emit "error" "Failed to download docker-compose.yml"
        emit_done "false" "docker-compose.yml download failed"
    fi
}

# ---------------------------------------------------------------------------
# Generate .env
# ---------------------------------------------------------------------------
generate_env() {
    emit "check" "Generating .env configuration"

    cat > "${STRIX_DIR}/.env" <<EOF
# Strix configuration -- generated by strix.sh
STRIX_LISTEN=:${STRIX_PORT}
EOF

    if [[ -n "$STRIX_FRIGATE_URL" ]]; then
        echo "STRIX_FRIGATE_URL=${STRIX_FRIGATE_URL}" >> "${STRIX_DIR}/.env"
        emit "ok" "Frigate URL: ${STRIX_FRIGATE_URL}" "{\"frigate_url\":\"${STRIX_FRIGATE_URL}\"}"
    fi

    if [[ -n "$STRIX_GO2RTC_URL" ]]; then
        echo "STRIX_GO2RTC_URL=${STRIX_GO2RTC_URL}" >> "${STRIX_DIR}/.env"
        emit "ok" "go2rtc URL: ${STRIX_GO2RTC_URL}" "{\"go2rtc_url\":\"${STRIX_GO2RTC_URL}\"}"
    fi

    if [[ -n "$STRIX_LOG_LEVEL" ]]; then
        echo "STRIX_LOG_LEVEL=${STRIX_LOG_LEVEL}" >> "${STRIX_DIR}/.env"
        emit "ok" "Log level: ${STRIX_LOG_LEVEL}"
    fi

    emit "ok" ".env generated (port ${STRIX_PORT})" "{\"port\":\"${STRIX_PORT}\"}"
}

# ---------------------------------------------------------------------------
# Pull image
# ---------------------------------------------------------------------------
pull_image() {
    emit "check" "Pulling image ${IMAGE}:${STRIX_TAG}"

    if docker compose -f "${STRIX_DIR}/docker-compose.yml" --env-file "${STRIX_DIR}/.env" pull &>/dev/null; then
        emit "ok" "Image pulled: ${IMAGE}:${STRIX_TAG}"
    else
        emit "error" "Failed to pull image ${IMAGE}:${STRIX_TAG}"
        emit_done "false" "Image pull failed"
    fi
}

# ---------------------------------------------------------------------------
# Start container
# ---------------------------------------------------------------------------
start_container() {
    # Check if strix container is already running
    if docker ps --format '{{.Names}}' 2>/dev/null | grep -q '^strix$'; then
        emit "check" "Strix container is running, recreating"
        if docker compose -f "${STRIX_DIR}/docker-compose.yml" --env-file "${STRIX_DIR}/.env" up -d --force-recreate &>/dev/null; then
            emit "ok" "Container recreated"
        else
            emit "error" "Failed to recreate container"
            emit_done "false" "Container recreate failed"
        fi
    else
        emit "install" "Starting Strix container"
        if docker compose -f "${STRIX_DIR}/docker-compose.yml" --env-file "${STRIX_DIR}/.env" up -d &>/dev/null; then
            emit "ok" "Container started"
        else
            emit "error" "Failed to start container"
            emit_done "false" "Container start failed"
        fi
    fi
}

# ---------------------------------------------------------------------------
# Healthcheck
# ---------------------------------------------------------------------------
healthcheck() {
    emit "check" "Waiting for Strix to respond"

    local retries=15
    local i
    for (( i = 1; i <= retries; i++ )); do
        if curl -sf --connect-timeout 2 --max-time 3 "http://localhost:${STRIX_PORT}/api/health" &>/dev/null; then
            local version
            version=$(curl -sf --max-time 3 "http://localhost:${STRIX_PORT}/api" 2>/dev/null | grep -oP '"version"\s*:\s*"\K[^"]+' || echo "unknown")
            emit "ok" "Strix v${version} is running on port ${STRIX_PORT}" "{\"version\":\"${version}\",\"port\":\"${STRIX_PORT}\"}"
            return 0
        fi
        sleep 1
    done

    emit "error" "Healthcheck failed after ${retries} attempts"
    emit_done "false" "Healthcheck failed"
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
main() {
    # 1. Working directory
    setup_dir

    # 2. Download compose file (if not present)
    download_compose

    # 3. Generate .env from parameters
    generate_env

    # 4. Pull image
    pull_image

    # 5. Start / recreate container
    start_container

    # 6. Healthcheck
    healthcheck

    # 7. Done -- include URL for navigator
    local lan_ip
    lan_ip=$(detect_lan_ip)
    local url="http://${lan_ip}:${STRIX_PORT}"

    emit_done "true" "url" "$url" "version" "$(curl -sf --max-time 3 "http://localhost:${STRIX_PORT}/api" 2>/dev/null | grep -oP '"version"\s*:\s*"\K[^"]+' || echo "unknown")" "port" "$STRIX_PORT" "ip" "$lan_ip"
}

main

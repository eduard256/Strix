#!/usr/bin/env bash
# =============================================================================
# Strix -- detect.sh (worker)
#
# Detects system environment: OS type, Docker, Compose, Frigate, go2rtc.
# Fast, silent, returns JSON events to stdout.
#
# Protocol:
#   - Every action is reported as a single-line JSON to stdout.
#   - Types: check, ok, miss, error, done
#   - Exit code: 0 always (detection never "fails", it just reports what it finds)
#
# Usage:
#   bash scripts/detect.sh
# =============================================================================

set -uo pipefail

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

# ---------------------------------------------------------------------------
# 1. System type
# ---------------------------------------------------------------------------
detect_system() {
    emit "check" "Detecting system"

    if command -v pveversion &>/dev/null; then
        local pve_ver
        pve_ver=$(pveversion 2>/dev/null | grep -oP 'pve-manager/\K[0-9]+\.[0-9]+' || echo "unknown")
        emit "ok" "Proxmox VE ${pve_ver}" "{\"type\":\"proxmox\",\"pve_version\":\"${pve_ver}\"}"

    elif [[ "$(uname -s 2>/dev/null)" == "Darwin" ]]; then
        local mac_ver
        mac_ver=$(sw_vers -productVersion 2>/dev/null || echo "unknown")
        local arch
        arch=$(uname -m 2>/dev/null || echo "unknown")
        emit "ok" "macOS ${mac_ver} (${arch})" "{\"type\":\"macos\",\"version\":\"${mac_ver}\",\"arch\":\"${arch}\"}"

    else
        local os_name="Linux"
        local os_id="unknown"
        local os_ver="unknown"
        local arch
        arch=$(uname -m 2>/dev/null || echo "unknown")

        if [[ -f /etc/os-release ]]; then
            . /etc/os-release
            os_name="${PRETTY_NAME:-Linux}"
            os_id="${ID:-unknown}"
            os_ver="${VERSION_ID:-unknown}"
        fi

        emit "ok" "${os_name} (${arch})" "{\"type\":\"linux\",\"id\":\"${os_id}\",\"version\":\"${os_ver}\",\"arch\":\"${arch}\"}"
    fi
}

# ---------------------------------------------------------------------------
# 2. Docker
# ---------------------------------------------------------------------------
detect_docker() {
    emit "check" "Checking Docker"

    if command -v docker &>/dev/null; then
        local ver
        ver=$(docker --version 2>/dev/null | grep -oP '\d+\.\d+\.\d+' | head -1 || echo "unknown")
        emit "ok" "Docker ${ver}" "{\"version\":\"${ver}\"}"
    else
        emit "miss" "Docker not installed"
    fi
}

# ---------------------------------------------------------------------------
# 3. Docker Compose
# ---------------------------------------------------------------------------
detect_compose() {
    emit "check" "Checking Docker Compose"

    if docker compose version &>/dev/null 2>&1; then
        local ver
        ver=$(docker compose version --short 2>/dev/null || echo "unknown")
        emit "ok" "Compose ${ver}" "{\"version\":\"${ver}\",\"type\":\"plugin\"}"
    elif command -v docker-compose &>/dev/null; then
        local ver
        ver=$(docker-compose --version 2>/dev/null | grep -oP '\d+\.\d+\.\d+' | head -1 || echo "unknown")
        emit "ok" "Compose ${ver}" "{\"version\":\"${ver}\",\"type\":\"standalone\"}"
    else
        emit "miss" "Docker Compose not installed"
    fi
}

# ---------------------------------------------------------------------------
# 4. Frigate
# ---------------------------------------------------------------------------
detect_frigate() {
    emit "check" "Checking Frigate"

    if command -v curl &>/dev/null; then
        if curl -sf --connect-timeout 2 --max-time 3 "http://localhost:5000/api/config" &>/dev/null; then
            emit "ok" "Frigate on port 5000" "{\"url\":\"http://localhost:5000\",\"port\":5000}"
            return
        fi
        if curl -sf --connect-timeout 2 --max-time 3 "http://localhost:8971/api/config" &>/dev/null; then
            emit "ok" "Frigate on port 8971" "{\"url\":\"http://localhost:8971\",\"port\":8971}"
            return
        fi
    fi

    emit "miss" "Frigate not found"
}

# ---------------------------------------------------------------------------
# 5. go2rtc
# ---------------------------------------------------------------------------
detect_go2rtc() {
    emit "check" "Checking go2rtc"

    if command -v curl &>/dev/null; then
        if curl -sf --connect-timeout 2 --max-time 3 "http://localhost:1984/api" &>/dev/null; then
            emit "ok" "go2rtc on port 1984" "{\"url\":\"http://localhost:1984\",\"port\":1984}"
            return
        fi
        if curl -sf --connect-timeout 2 --max-time 3 "http://localhost:11984/api" &>/dev/null; then
            emit "ok" "go2rtc on port 11984" "{\"url\":\"http://localhost:11984\",\"port\":11984}"
            return
        fi
    fi

    emit "miss" "go2rtc not found"
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
main() {
    detect_system
    detect_docker
    detect_compose
    detect_frigate
    detect_go2rtc
    printf '{"type":"done","ok":true}\n'
}

main

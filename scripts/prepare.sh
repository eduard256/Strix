#!/usr/bin/env bash
# =============================================================================
# Strix -- prepare.sh (worker)
#
# Silent backend worker that prepares the system for Strix deployment.
# Detects OS, installs Docker and Docker Compose if missing.
#
# Protocol:
#   - Every action is reported as a single-line JSON to stdout.
#   - Types: check, ok, miss, install, error, done
#   - Field "msg" is always human-readable.
#   - Field "data" is optional, carries machine-readable details.
#   - Last line is always: {"type":"done","ok":true} or {"type":"done","ok":false,"error":"..."}
#   - All internal command output goes to /dev/null or stderr (never stdout).
#   - Exit code: 0 = success, 1 = failure.
#
# Usage:
#   bash scripts/prepare.sh
#   result=$(bash scripts/prepare.sh)
# =============================================================================

set -uo pipefail

# ---------------------------------------------------------------------------
# JSON helpers (no jq dependency)
# ---------------------------------------------------------------------------

# Emit a JSON event line to stdout.
# Usage: emit "type" "msg" '{"key":"val"}'
emit() {
    local type="$1"
    local msg="$2"
    local data="${3:-}"

    # Escape double quotes in msg
    msg="${msg//\\/\\\\}"
    msg="${msg//\"/\\\"}"

    if [[ -n "$data" ]]; then
        printf '{"type":"%s","msg":"%s","data":%s}\n' "$type" "$msg" "$data"
    else
        printf '{"type":"%s","msg":"%s"}\n' "$type" "$msg"
    fi
}

# Emit final done event and exit.
emit_done() {
    local ok="$1"
    local error="${2:-}"

    if [[ "$ok" == "true" ]]; then
        printf '{"type":"done","ok":true}\n'
        exit 0
    else
        error="${error//\\/\\\\}"
        error="${error//\"/\\\"}"
        printf '{"type":"done","ok":false,"error":"%s"}\n' "$error"
        exit 1
    fi
}

# ---------------------------------------------------------------------------
# OS detection
# ---------------------------------------------------------------------------
detect_os() {
    emit "check" "Detecting operating system"

    local kernel
    kernel=$(uname -s 2>/dev/null || echo "unknown")

    case "$kernel" in
        Linux)
            local os_id="unknown"
            local os_ver="unknown"
            local os_name="Unknown Linux"

            if [[ -f /etc/os-release ]]; then
                # shellcheck disable=SC1091
                . /etc/os-release
                os_id="${ID:-unknown}"
                os_ver="${VERSION_ID:-unknown}"
                os_name="${PRETTY_NAME:-${ID} ${VERSION_ID}}"
            fi

            local arch
            arch=$(uname -m 2>/dev/null || echo "unknown")
            local arch_label="$arch"
            case "$arch" in
                x86_64)  arch_label="amd64" ;;
                aarch64) arch_label="arm64" ;;
                armv7l)  arch_label="armv7" ;;
            esac

            OS_TYPE="linux"
            OS_ID="$os_id"
            OS_VER="$os_ver"
            OS_NAME="$os_name"
            OS_ARCH="$arch_label"

            emit "ok" "${os_name} (${arch_label})" \
                "{\"os\":\"linux\",\"id\":\"${os_id}\",\"ver\":\"${os_ver}\",\"arch\":\"${arch_label}\"}"
            ;;

        Darwin)
            local mac_ver
            mac_ver=$(sw_vers -productVersion 2>/dev/null || echo "unknown")

            local arch
            arch=$(uname -m 2>/dev/null || echo "unknown")
            local arch_label="$arch"
            case "$arch" in
                x86_64) arch_label="amd64" ;;
                arm64)  arch_label="arm64" ;;
            esac

            OS_TYPE="mac"
            OS_ID="macos"
            OS_VER="$mac_ver"
            OS_NAME="macOS ${mac_ver}"
            OS_ARCH="$arch_label"

            emit "ok" "macOS ${mac_ver} (${arch_label})" \
                "{\"os\":\"mac\",\"id\":\"macos\",\"ver\":\"${mac_ver}\",\"arch\":\"${arch_label}\"}"
            ;;

        *)
            emit "error" "Unsupported OS: ${kernel}" \
                "{\"kernel\":\"${kernel}\"}"
            emit_done "false" "Unsupported operating system: ${kernel}"
            ;;
    esac
}

# ---------------------------------------------------------------------------
# Root check (Linux only)
# ---------------------------------------------------------------------------
check_root() {
    if [[ "$OS_TYPE" == "mac" ]]; then
        return
    fi

    emit "check" "Checking root privileges"

    if [[ "$(id -u)" -eq 0 ]]; then
        emit "ok" "Running as root"
    else
        emit "error" "Root privileges required. Run with sudo."
        emit_done "false" "Not running as root"
    fi
}

# ---------------------------------------------------------------------------
# curl (required for Docker install and compose download)
# ---------------------------------------------------------------------------
ensure_curl() {
    emit "check" "Checking curl"

    if command -v curl &>/dev/null; then
        emit "ok" "curl available"
        return 0
    fi

    emit "miss" "curl not found"
    emit "install" "Installing curl"

    local pkg_mgr="unknown"
    if command -v apt-get &>/dev/null; then
        pkg_mgr="apt"
        emit "check" "Updating apt package lists"
        if ! apt-get update -qq &>/dev/null; then
            emit "error" "apt-get update failed"
            emit_done "false" "Failed to update package lists"
        fi
        emit "ok" "Package lists updated"
        emit "install" "Installing curl via apt"
        apt-get install -y -qq curl &>/dev/null
    elif command -v yum &>/dev/null; then
        pkg_mgr="yum"
        emit "install" "Installing curl via yum"
        yum install -y -q curl &>/dev/null
    elif command -v dnf &>/dev/null; then
        pkg_mgr="dnf"
        emit "install" "Installing curl via dnf"
        dnf install -y -q curl &>/dev/null
    elif command -v apk &>/dev/null; then
        pkg_mgr="apk"
        emit "install" "Installing curl via apk"
        apk add --no-cache curl &>/dev/null
    elif command -v pacman &>/dev/null; then
        pkg_mgr="pacman"
        emit "install" "Installing curl via pacman"
        pacman -Sy --noconfirm curl &>/dev/null
    elif command -v zypper &>/dev/null; then
        pkg_mgr="zypper"
        emit "install" "Installing curl via zypper"
        zypper install -y curl &>/dev/null
    else
        emit "error" "No supported package manager found" "{\"tried\":\"apt,yum,dnf,apk,pacman,zypper\"}"
        emit_done "false" "Cannot install curl: no supported package manager"
    fi

    if command -v curl &>/dev/null; then
        emit "ok" "curl installed via ${pkg_mgr}"
        return 0
    fi

    emit "error" "curl installation failed via ${pkg_mgr}"
    emit_done "false" "curl installation failed"
}

# ---------------------------------------------------------------------------
# Docker
# ---------------------------------------------------------------------------
check_docker() {
    emit "check" "Checking Docker"

    if command -v docker &>/dev/null; then
        local ver
        ver=$(docker --version 2>/dev/null | grep -oP '\d+\.\d+\.\d+' | head -1 || echo "unknown")
        emit "ok" "Docker ${ver}" "{\"version\":\"${ver}\"}"
        return 0
    fi

    emit "miss" "Docker not found"
    return 1
}

install_docker_linux() {
    emit "install" "Downloading Docker install script from get.docker.com"

    local tmp_script="/tmp/get-docker.sh"
    if ! curl -fsSL https://get.docker.com -o "$tmp_script" 2>/dev/null; then
        emit "error" "Failed to download get.docker.com"
        emit_done "false" "Docker download failed"
    fi

    emit "ok" "Docker install script downloaded"
    emit "install" "Running Docker install script (this may take a minute)"

    if sh "$tmp_script" &>/dev/null; then
        rm -f "$tmp_script"
        emit "ok" "Docker install script completed"
    else
        rm -f "$tmp_script"
        emit "error" "Docker install script failed"
        emit_done "false" "Docker installation failed"
    fi

    # Enable and start via systemd
    if command -v systemctl &>/dev/null; then
        emit "check" "Enabling Docker service"
        systemctl enable docker &>/dev/null || true
        systemctl start docker &>/dev/null || true

        if systemctl is-active docker &>/dev/null; then
            emit "ok" "Docker service started"
        else
            emit "error" "Docker service failed to start"
            emit_done "false" "Docker service failed to start"
        fi
    fi

    # Verify docker binary works
    if command -v docker &>/dev/null; then
        local ver
        ver=$(docker --version 2>/dev/null | grep -oP '\d+\.\d+\.\d+' | head -1 || echo "unknown")
        emit "ok" "Docker ${ver} installed" "{\"version\":\"${ver}\"}"
        return 0
    fi

    emit "error" "Docker binary not found after install"
    emit_done "false" "Docker installation failed"
}

install_docker_mac() {
    emit "check" "Checking Docker Desktop for Mac"

    # Docker Desktop should already be installed on Mac.
    # We can't silently install it -- it requires GUI interaction.
    emit "error" "Docker not found. Install Docker Desktop from https://docker.com/products/docker-desktop"
    emit_done "false" "Docker Desktop not installed on Mac"
}

# ---------------------------------------------------------------------------
# Docker Compose
# ---------------------------------------------------------------------------
check_compose() {
    emit "check" "Checking Docker Compose"

    # Plugin (v2): docker compose
    if docker compose version &>/dev/null; then
        local ver
        ver=$(docker compose version --short 2>/dev/null || echo "unknown")
        COMPOSE_CMD="docker compose"
        emit "ok" "Docker Compose ${ver} (plugin)" "{\"version\":\"${ver}\",\"type\":\"plugin\"}"
        return 0
    fi

    # Standalone: docker-compose
    if command -v docker-compose &>/dev/null; then
        local ver
        ver=$(docker-compose --version 2>/dev/null | grep -oP '\d+\.\d+\.\d+' | head -1 || echo "unknown")
        COMPOSE_CMD="docker-compose"
        emit "ok" "Docker Compose ${ver} (standalone)" "{\"version\":\"${ver}\",\"type\":\"standalone\"}"
        return 0
    fi

    emit "miss" "Docker Compose not found"
    return 1
}

install_compose_linux() {
    emit "install" "Installing Docker Compose plugin"

    local installed=false

    # Try package manager first
    if command -v apt-get &>/dev/null; then
        apt-get update -qq &>/dev/null && apt-get install -y -qq docker-compose-plugin &>/dev/null && installed=true
    elif command -v yum &>/dev/null; then
        yum install -y -q docker-compose-plugin &>/dev/null && installed=true
    elif command -v dnf &>/dev/null; then
        dnf install -y -q docker-compose-plugin &>/dev/null && installed=true
    fi

    # Fallback: download binary
    if [[ "$installed" == false ]]; then
        emit "install" "Downloading Docker Compose binary"

        local compose_ver="v2.29.1"
        local compose_arch
        case "$OS_ARCH" in
            amd64) compose_arch="x86_64" ;;
            arm64) compose_arch="aarch64" ;;
            *)     compose_arch="$(uname -m)" ;;
        esac

        mkdir -p /usr/local/lib/docker/cli-plugins &>/dev/null
        if curl -fsSL "https://github.com/docker/compose/releases/download/${compose_ver}/docker-compose-linux-${compose_arch}" \
            -o /usr/local/lib/docker/cli-plugins/docker-compose &>/dev/null; then
            chmod +x /usr/local/lib/docker/cli-plugins/docker-compose
            installed=true
        fi
    fi

    # Verify
    if [[ "$installed" == true ]] && docker compose version &>/dev/null; then
        local ver
        ver=$(docker compose version --short 2>/dev/null || echo "unknown")
        COMPOSE_CMD="docker compose"
        emit "ok" "Docker Compose ${ver} installed" "{\"version\":\"${ver}\"}"
        return 0
    fi

    emit "error" "Docker Compose installation failed"
    emit_done "false" "Docker Compose installation failed"
}

install_compose_mac() {
    # On Mac, Docker Compose comes with Docker Desktop
    emit "error" "Docker Compose not found. It should be included with Docker Desktop."
    emit_done "false" "Docker Compose missing on Mac"
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
main() {
    # 1. Detect OS
    detect_os

    # 2. Root check
    check_root

    # 3. curl (needed for Docker install, always present on Mac)
    if [[ "$OS_TYPE" == "linux" ]]; then
        ensure_curl
    fi

    # 4. Docker
    if ! check_docker; then
        case "$OS_TYPE" in
            linux) install_docker_linux ;;
            mac)   install_docker_mac ;;
        esac
    fi

    # 5. Docker Compose
    if ! check_compose; then
        case "$OS_TYPE" in
            linux) install_compose_linux ;;
            mac)   install_compose_mac ;;
        esac
    fi

    # 6. All good
    emit_done "true"
}

main

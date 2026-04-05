#!/usr/bin/env bash
# =============================================================================
# Strix Installer
# Universal installer for any Linux distribution.
# Installs Docker (via get.docker.com), Docker Compose, detects Frigate/go2rtc,
# and deploys Strix via docker compose.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/eduard256/Strix/main/install.sh | sudo bash
#   sudo bash install.sh [OPTIONS]
#
# Options:
#   --no-logo          Suppress ASCII logo (useful when called from another script)
#   --no-color         Disable colored output
#   --yes, -y          Non-interactive mode, accept all defaults
#   --version TAG      Set image tag without prompt (latest/dev/1.0.9)
#   --verbose, -v      Show detailed output from docker commands
#
# Exit codes:
#   0 = success
#   1 = docker installation failed
#   2 = docker compose not available
#   3 = image pull failed
#   4 = container failed to start
#   5 = healthcheck failed
# =============================================================================

set -euo pipefail

# ---------------------------------------------------------------------------
# Globals
# ---------------------------------------------------------------------------
STRIX_DIR="/opt/strix"
STRIX_PORT="4567"
IMAGE="eduard256/strix"
LOG_FILE="${STRIX_DIR}/install.log"
DIALOG_TIMEOUT=10

# Flags (overridden by CLI args)
SHOW_LOGO=true
USE_COLOR=true
INTERACTIVE=true
VERBOSE=false
TAG=""

# Result variables (printed at the end for parent scripts)
INSTALL_MODE=""        # install | update
FRIGATE_STATUS="none"  # found | set | none
GO2RTC_STATUS="none"   # found | set | none
FINAL_VERSION=""

# ---------------------------------------------------------------------------
# Parse CLI arguments
# ---------------------------------------------------------------------------
while [[ $# -gt 0 ]]; do
    case "$1" in
        --no-logo)   SHOW_LOGO=false; shift ;;
        --no-color)  USE_COLOR=false;  shift ;;
        --yes|-y)    INTERACTIVE=false; shift ;;
        --verbose|-v) VERBOSE=true;    shift ;;
        --version)   TAG="$2";         shift 2 ;;
        *)           shift ;;
    esac
done

# ---------------------------------------------------------------------------
# Color setup
# ---------------------------------------------------------------------------
setup_colors() {
    if [[ "$USE_COLOR" == false ]] || [[ "${NO_COLOR:-}" != "" ]] || [[ ! -t 1 ]]; then
        USE_COLOR=false
        C_RESET="" C_BOLD="" C_DIM=""
        C_RED="" C_GREEN="" C_YELLOW="" C_CYAN="" C_MAGENTA="" C_WHITE=""
        return
    fi

    local colors
    colors=$(tput colors 2>/dev/null || echo 0)
    if [[ "$colors" -lt 8 ]]; then
        USE_COLOR=false
        C_RESET="" C_BOLD="" C_DIM=""
        C_RED="" C_GREEN="" C_YELLOW="" C_CYAN="" C_MAGENTA="" C_WHITE=""
        return
    fi

    C_RESET="\033[0m"
    C_BOLD="\033[1m"
    C_DIM="\033[2m"
    C_RED="\033[31m"
    C_GREEN="\033[32m"
    C_YELLOW="\033[33m"
    C_CYAN="\033[36m"
    C_MAGENTA="\033[35m"
    C_WHITE="\033[97m"
}

# ---------------------------------------------------------------------------
# Terminal width helpers
# ---------------------------------------------------------------------------
term_width() {
    local w
    w=$(tput cols 2>/dev/null || echo 80)
    [[ "$w" -lt 40 ]] && w=40
    echo "$w"
}

# Print text centered in terminal
center() {
    local text="$1"
    local w
    w=$(term_width)
    # Strip ANSI codes for length calculation
    local stripped
    stripped=$(echo -e "$text" | sed 's/\x1b\[[0-9;]*m//g')
    local len=${#stripped}
    local pad=$(( (w - len) / 2 ))
    [[ "$pad" -lt 0 ]] && pad=0
    printf "%*s%b\n" "$pad" "" "$text"
}

# ---------------------------------------------------------------------------
# Logging
# ---------------------------------------------------------------------------
mkdir -p "$STRIX_DIR" 2>/dev/null || true

log_raw() {
    local ts
    ts=$(date '+%H:%M:%S')
    echo -e "$ts  $1" >> "$LOG_FILE" 2>/dev/null || true
}

log() {
    log_raw "$1"
    if [[ "$VERBOSE" == true ]]; then
        local ts
        ts=$(date '+%H:%M:%S')
        echo -e "  ${C_DIM}${ts}  $1${C_RESET}"
    fi
}

# Status line helpers
status_ok()   { echo -e "  ${C_GREEN}${C_BOLD}[OK]${C_RESET}  $1"; log "[OK] $1"; }
status_warn() { echo -e "  ${C_YELLOW}${C_BOLD}[!!]${C_RESET}  $1"; log "[!!] $1"; }
status_fail() { echo -e "  ${C_RED}${C_BOLD}[XX]${C_RESET}  $1"; log "[XX] $1"; }
status_info() { echo -e "  ${C_CYAN}${C_BOLD}[..]${C_RESET}  $1"; log "[..] $1"; }
status_skip() { echo -e "  ${C_DIM}[--]${C_RESET}  $1"; log "[--] $1"; }

# ---------------------------------------------------------------------------
# ASCII art
# ---------------------------------------------------------------------------
show_logo() {
    [[ "$SHOW_LOGO" == false ]] && return

    echo ""

    # Block STRIX title
    center "${C_MAGENTA}${C_BOLD}███████╗████████╗██████╗ ██╗██╗  ██╗${C_RESET}"
    center "${C_MAGENTA}${C_BOLD}██╔════╝╚══██╔══╝██╔══██╗██║╚██╗██╔╝${C_RESET}"
    center "${C_MAGENTA}${C_BOLD}███████╗   ██║   ██████╔╝██║ ╚███╔╝${C_RESET}"
    center "${C_MAGENTA}${C_BOLD}╚════██║   ██║   ██╔══██╗██║ ██╔██╗${C_RESET}"
    center "${C_MAGENTA}${C_BOLD}███████║   ██║   ██║  ██║██║██╔╝ ██╗${C_RESET}"
    center "${C_MAGENTA}${C_BOLD}╚══════╝   ╚═╝   ╚═╝  ╚═╝╚═╝╚═╝  ╚═╝${C_RESET}"

    echo ""

    # Owl
    center "${C_CYAN}     __________-------____                 ____-------__________${C_RESET}"
    center "${C_CYAN}    \\------____-------___--__---------__--___-------____------/${C_RESET}"
    center "${C_CYAN}     \\//////// / / / / / \\   _-------_   / \\ \\ \\ \\ \\ \\\\\\\\\\\\\\\\/${C_RESET}"
    center "${C_CYAN}       \\////-/-/------/_/_| /___   ___\\ |_\\_\\------\\-\\-\\\\\\\\/${C_RESET}"
    center "${C_CYAN}         --//// / /  /  //|| ${C_YELLOW}(O)${C_CYAN}\\ /${C_YELLOW}(O)${C_CYAN} ||\\\\  \\  \\ \\ \\\\\\\\--${C_RESET}"
    center "${C_CYAN}              ---__/  // /| \\_  /V\\  _/ |\\ \\\\  \\__---${C_RESET}"
    center "${C_CYAN}                   -//  / /\\_ ------- _/\\ \\  \\\\-${C_RESET}"
    center "${C_CYAN}                     \\_/_/ /\\---------/\\ \\_\\_/${C_RESET}"
    center "${C_CYAN}                         ----\\   |   /----${C_RESET}"
    center "${C_CYAN}                              | -|- |${C_RESET}"
    center "${C_CYAN}                             /   |   \\${C_RESET}"
    center "${C_CYAN}                             ----  ----${C_RESET}"

    echo ""
    center "${C_WHITE}${C_BOLD}Smart IP Camera Stream Finder${C_RESET}"
    center "${C_DIM}────────────────────────────────────────${C_RESET}"
    echo ""
}

show_owl_small() {
    echo -e "  ${C_CYAN}       ,___,${C_RESET}"
    echo -e "  ${C_CYAN}      /(6 6)\\_${C_RESET}"
    echo -e "  ${C_CYAN}     /\\\` ' \`'\\\\_${C_RESET}"
    echo -e "  ${C_CYAN}     \\\\\\\\_''''|\\\\\\\\${C_RESET}"
    echo -e "  ${C_CYAN}      )\\\\\\\\\\\\''//||/${C_RESET}"
    echo -e "  ${C_CYAN} ._,--/////\"\"------${C_RESET}"
}

# ---------------------------------------------------------------------------
# Dialog boxes
# ---------------------------------------------------------------------------

# Draw a box around content.
# Usage: draw_box "Title" "line1" "line2" ...
draw_box() {
    local title="$1"; shift
    local w
    w=$(term_width)
    local box_w=$(( w - 4 ))
    [[ "$box_w" -gt 60 ]] && box_w=60

    local top_line=""
    local bot_line=""
    local i

    # Build horizontal lines
    for (( i = 0; i < box_w - 2; i++ )); do
        top_line="${top_line}─"
        bot_line="${bot_line}─"
    done

    # Center the box
    local pad=$(( (w - box_w) / 2 ))
    [[ "$pad" -lt 0 ]] && pad=0
    local sp
    sp=$(printf "%*s" "$pad" "")

    echo ""
    echo -e "${sp}${C_CYAN}┌─ ${C_WHITE}${C_BOLD}${title}${C_RESET}${C_CYAN} ${top_line:$(( ${#title} + 3 ))}┐${C_RESET}"
    echo -e "${sp}${C_CYAN}│$(printf "%*s" $(( box_w - 2 )) "")│${C_RESET}"

    for line in "$@"; do
        local stripped
        stripped=$(echo -e "$line" | sed 's/\x1b\[[0-9;]*m//g')
        local line_len=${#stripped}
        local right_pad=$(( box_w - 2 - line_len ))
        [[ "$right_pad" -lt 0 ]] && right_pad=0
        echo -e "${sp}${C_CYAN}│${C_RESET} ${line}$(printf "%*s" "$right_pad" "")${C_CYAN}│${C_RESET}"
    done

    echo -e "${sp}${C_CYAN}│$(printf "%*s" $(( box_w - 2 )) "")│${C_RESET}"
    echo -e "${sp}${C_CYAN}└${bot_line}┘${C_RESET}"
    echo ""
}

# Prompt with timeout. Returns user input or default.
# Usage: timed_prompt "prompt text" "default" timeout_seconds
timed_prompt() {
    local prompt_text="$1"
    local default="$2"
    local timeout="$3"
    local result=""

    if [[ "$INTERACTIVE" == false ]]; then
        echo "$default"
        return
    fi

    # Show countdown hint
    echo -ne "  ${C_YELLOW}${prompt_text}${C_RESET} ${C_DIM}(${timeout}s -> ${default})${C_RESET}: "

    if read -r -t "$timeout" result 2>/dev/null; then
        if [[ -z "$result" ]]; then
            echo "$default"
        else
            echo "$result"
        fi
    else
        echo ""  # newline after timeout
        echo "$default"
    fi
}

# ---------------------------------------------------------------------------
# System detection
# ---------------------------------------------------------------------------
detect_system() {
    log "Detecting OS from /etc/os-release"

    if [[ -f /etc/os-release ]]; then
        # shellcheck disable=SC1091
        . /etc/os-release
        OS_ID="${ID:-unknown}"
        OS_VERSION="${VERSION_ID:-unknown}"
        OS_NAME="${PRETTY_NAME:-${ID} ${VERSION_ID}}"
    else
        OS_ID="unknown"
        OS_VERSION="unknown"
        OS_NAME="Unknown Linux"
    fi

    ARCH=$(uname -m)
    case "$ARCH" in
        x86_64)  ARCH_LABEL="amd64" ;;
        aarch64) ARCH_LABEL="arm64" ;;
        armv7l)  ARCH_LABEL="armv7" ;;
        *)       ARCH_LABEL="$ARCH" ;;
    esac

    log "Detected: ${OS_ID} ${OS_VERSION} (${ARCH_LABEL})"
    status_ok "System: ${C_WHITE}${C_BOLD}${OS_NAME}${C_RESET} (${ARCH_LABEL})"
}

# ---------------------------------------------------------------------------
# Ensure root
# ---------------------------------------------------------------------------
ensure_root() {
    if [[ "$(id -u)" -ne 0 ]]; then
        status_info "Root privileges required, re-running with sudo..."
        exec sudo "$0" "$@"
    fi
}

# ---------------------------------------------------------------------------
# Install curl if missing
# ---------------------------------------------------------------------------
ensure_curl() {
    if command -v curl &>/dev/null; then
        log "curl found"
        return
    fi

    status_info "Installing curl..."
    log "curl not found, installing"

    case "$OS_ID" in
        ubuntu|debian|raspbian|linuxmint|pop)
            apt-get update -qq && apt-get install -y -qq curl ;;
        centos|rhel|rocky|almalinux|ol)
            yum install -y -q curl ;;
        fedora)
            dnf install -y -q curl ;;
        alpine)
            apk add --no-cache curl ;;
        arch|manjaro)
            pacman -Sy --noconfirm curl ;;
        opensuse*|sles)
            zypper install -y curl ;;
        *)
            status_fail "Cannot install curl: unknown package manager for ${OS_ID}"
            status_fail "Please install curl manually and re-run the script"
            exit 1 ;;
    esac

    if command -v curl &>/dev/null; then
        status_ok "curl installed"
    else
        status_fail "Failed to install curl"
        exit 1
    fi
}

# ---------------------------------------------------------------------------
# Docker installation
# ---------------------------------------------------------------------------
check_docker() {
    if command -v docker &>/dev/null; then
        local ver
        ver=$(docker --version 2>/dev/null | grep -oP '\d+\.\d+\.\d+' | head -1 || echo "unknown")
        status_ok "Docker ${C_WHITE}${C_BOLD}${ver}${C_RESET}"
        log "Docker found: ${ver}"
        return 0
    fi
    return 1
}

install_docker() {
    status_info "Installing Docker via ${C_WHITE}get.docker.com${C_RESET}..."
    log "Downloading and running get.docker.com"

    if [[ "$VERBOSE" == true ]]; then
        curl -fsSL https://get.docker.com | sh
    else
        curl -fsSL https://get.docker.com | sh &>/dev/null
    fi

    # Enable and start docker
    if command -v systemctl &>/dev/null; then
        systemctl enable docker &>/dev/null || true
        systemctl start docker &>/dev/null || true
    fi

    if check_docker; then
        return 0
    else
        status_fail "Docker installation failed"
        log "Docker installation failed"
        exit 1
    fi
}

# ---------------------------------------------------------------------------
# Docker Compose check
# ---------------------------------------------------------------------------
check_compose() {
    # Check plugin first (docker compose v2)
    if docker compose version &>/dev/null; then
        local ver
        ver=$(docker compose version --short 2>/dev/null || echo "unknown")
        status_ok "Docker Compose ${C_WHITE}${C_BOLD}${ver}${C_RESET}"
        log "Docker Compose found: ${ver}"
        COMPOSE_CMD="docker compose"
        return 0
    fi

    # Check standalone docker-compose
    if command -v docker-compose &>/dev/null; then
        local ver
        ver=$(docker-compose --version 2>/dev/null | grep -oP '\d+\.\d+\.\d+' | head -1 || echo "unknown")
        status_ok "Docker Compose ${C_WHITE}${C_BOLD}${ver}${C_RESET} (standalone)"
        log "Docker Compose standalone found: ${ver}"
        COMPOSE_CMD="docker-compose"
        return 0
    fi

    return 1
}

install_compose() {
    status_info "Installing Docker Compose plugin..."
    log "Installing Docker Compose plugin"

    # Docker Compose V2 is typically bundled with Docker now.
    # Try installing the plugin package.
    case "$OS_ID" in
        ubuntu|debian|raspbian|linuxmint|pop)
            apt-get update -qq && apt-get install -y -qq docker-compose-plugin 2>/dev/null ;;
        centos|rhel|rocky|almalinux|ol|fedora)
            yum install -y -q docker-compose-plugin 2>/dev/null || dnf install -y -q docker-compose-plugin 2>/dev/null ;;
        *)
            # Fallback: download binary
            local compose_ver="v2.29.1"
            local compose_arch
            case "$ARCH_LABEL" in
                amd64) compose_arch="x86_64" ;;
                arm64) compose_arch="aarch64" ;;
                *)     compose_arch="$ARCH" ;;
            esac
            mkdir -p /usr/local/lib/docker/cli-plugins
            curl -fsSL "https://github.com/docker/compose/releases/download/${compose_ver}/docker-compose-linux-${compose_arch}" \
                -o /usr/local/lib/docker/cli-plugins/docker-compose
            chmod +x /usr/local/lib/docker/cli-plugins/docker-compose
            ;;
    esac

    if check_compose; then
        return 0
    else
        status_fail "Docker Compose installation failed"
        log "Docker Compose installation failed"
        exit 2
    fi
}

# ---------------------------------------------------------------------------
# Check if Strix is already installed
# ---------------------------------------------------------------------------
check_existing() {
    if [[ -f "${STRIX_DIR}/docker-compose.yml" ]]; then
        if $COMPOSE_CMD -f "${STRIX_DIR}/docker-compose.yml" ps --format '{{.State}}' 2>/dev/null | grep -qi "running"; then
            INSTALL_MODE="update"
            status_info "Strix is already running -- ${C_WHITE}${C_BOLD}update mode${C_RESET}"
            log "Existing Strix installation found, switching to update mode"
            return 0
        fi
        # compose file exists but not running
        INSTALL_MODE="update"
        status_info "Strix config found but not running -- ${C_WHITE}${C_BOLD}update mode${C_RESET}"
        log "Existing Strix config found (not running), switching to update mode"
        return 0
    fi

    INSTALL_MODE="install"
    log "No existing Strix installation found"
    return 1
}

# ---------------------------------------------------------------------------
# Version selection dialog
# ---------------------------------------------------------------------------
select_version() {
    # If already set via --version flag
    if [[ -n "$TAG" ]]; then
        FINAL_VERSION="$TAG"
        status_ok "Version: ${C_WHITE}${C_BOLD}${TAG}${C_RESET} (from --version flag)"
        log "Version set via flag: ${TAG}"
        return
    fi

    if [[ "$INTERACTIVE" == false ]]; then
        FINAL_VERSION="latest"
        status_ok "Version: ${C_WHITE}${C_BOLD}latest${C_RESET} (non-interactive default)"
        log "Version defaulted to latest (non-interactive)"
        return
    fi

    draw_box "Select Version" \
        "  ${C_GREEN}${C_BOLD}[1]${C_RESET}  latest              ${C_DIM}(recommended)${C_RESET}" \
        "  ${C_YELLOW}${C_BOLD}[2]${C_RESET}  dev                 ${C_DIM}(development)${C_RESET}" \
        "  ${C_CYAN}${C_BOLD}[3]${C_RESET}  custom tag           ${C_DIM}(e.g. 1.0.9)${C_RESET}"

    local choice
    choice=$(timed_prompt "Choice [1/2/3]" "1" "$DIALOG_TIMEOUT")

    case "$choice" in
        1|"") FINAL_VERSION="latest" ;;
        2)    FINAL_VERSION="dev" ;;
        3)
            echo -ne "  ${C_YELLOW}Enter tag: ${C_RESET}"
            local custom_tag
            if read -r -t "$DIALOG_TIMEOUT" custom_tag 2>/dev/null && [[ -n "$custom_tag" ]]; then
                FINAL_VERSION="$custom_tag"
            else
                FINAL_VERSION="latest"
                echo ""
            fi
            ;;
        *)    FINAL_VERSION="latest" ;;
    esac

    status_ok "Version: ${C_WHITE}${C_BOLD}${FINAL_VERSION}${C_RESET}"
    log "Version selected: ${FINAL_VERSION}"
}

# ---------------------------------------------------------------------------
# Service detection (Frigate / go2rtc)
# ---------------------------------------------------------------------------
probe_frigate() {
    log "Probing Frigate at localhost:5000"

    if curl -sf --connect-timeout 2 --max-time 3 "http://localhost:5000/api/config" &>/dev/null; then
        FRIGATE_STATUS="found"
        FRIGATE_URL="http://localhost:5000"
        status_ok "Frigate: ${C_WHITE}${C_BOLD}localhost:5000${C_RESET}"
        log "Frigate found at localhost:5000"
        return
    fi

    log "Frigate not found locally"

    if [[ "$INTERACTIVE" == false ]]; then
        FRIGATE_STATUS="none"
        FRIGATE_URL=""
        status_skip "Frigate: not found (skipped)"
        return
    fi

    echo ""
    show_owl_small
    draw_box "Frigate Not Found" \
        "  Frigate was not detected on this machine." \
        "  Enter Frigate URL or leave empty to skip." \
        "" \
        "  ${C_DIM}Example: http://192.168.1.100:5000${C_RESET}"

    local input
    input=$(timed_prompt "Frigate URL" "" "$DIALOG_TIMEOUT")

    if [[ -n "$input" ]]; then
        FRIGATE_STATUS="set"
        FRIGATE_URL="$input"
        status_ok "Frigate: ${C_WHITE}${C_BOLD}${input}${C_RESET} (manual)"
        log "Frigate URL set manually: ${input}"
    else
        FRIGATE_STATUS="none"
        FRIGATE_URL=""
        status_skip "Frigate: not configured"
        log "Frigate skipped"
    fi
}

probe_go2rtc() {
    log "Probing go2rtc at localhost:1984 and localhost:11984"

    if curl -sf --connect-timeout 2 --max-time 3 "http://localhost:1984/api" &>/dev/null; then
        GO2RTC_STATUS="found"
        GO2RTC_URL="http://localhost:1984"
        status_ok "go2rtc: ${C_WHITE}${C_BOLD}localhost:1984${C_RESET}"
        log "go2rtc found at localhost:1984"
        return
    fi

    if curl -sf --connect-timeout 2 --max-time 3 "http://localhost:11984/api" &>/dev/null; then
        GO2RTC_STATUS="found"
        GO2RTC_URL="http://localhost:11984"
        status_ok "go2rtc: ${C_WHITE}${C_BOLD}localhost:11984${C_RESET}"
        log "go2rtc found at localhost:11984"
        return
    fi

    log "go2rtc not found locally"

    if [[ "$INTERACTIVE" == false ]]; then
        GO2RTC_STATUS="none"
        GO2RTC_URL=""
        status_skip "go2rtc: not found (skipped)"
        return
    fi

    echo ""
    show_owl_small
    draw_box "go2rtc Not Found" \
        "  go2rtc was not detected on this machine." \
        "  Enter go2rtc URL or leave empty to skip." \
        "" \
        "  ${C_DIM}Example: http://192.168.1.100:1984${C_RESET}"

    local input
    input=$(timed_prompt "go2rtc URL" "" "$DIALOG_TIMEOUT")

    if [[ -n "$input" ]]; then
        GO2RTC_STATUS="set"
        GO2RTC_URL="$input"
        status_ok "go2rtc: ${C_WHITE}${C_BOLD}${input}${C_RESET} (manual)"
        log "go2rtc URL set manually: ${input}"
    else
        GO2RTC_STATUS="none"
        GO2RTC_URL=""
        status_skip "go2rtc: not configured"
        log "go2rtc skipped"
    fi
}

# ---------------------------------------------------------------------------
# Generate config files
# ---------------------------------------------------------------------------
generate_env() {
    log "Generating ${STRIX_DIR}/.env"

    cat > "${STRIX_DIR}/.env" <<EOF
# Strix configuration -- generated by install.sh
STRIX_TAG=${FINAL_VERSION}
STRIX_PORT=${STRIX_PORT}
EOF

    if [[ -n "${FRIGATE_URL:-}" ]]; then
        echo "STRIX_FRIGATE_URL=${FRIGATE_URL}" >> "${STRIX_DIR}/.env"
    fi

    if [[ -n "${GO2RTC_URL:-}" ]]; then
        echo "STRIX_GO2RTC_URL=${GO2RTC_URL}" >> "${STRIX_DIR}/.env"
    fi

    log "Generated .env: TAG=${FINAL_VERSION}, FRIGATE=${FRIGATE_URL:-}, GO2RTC=${GO2RTC_URL:-}"
}

generate_compose() {
    log "Generating ${STRIX_DIR}/docker-compose.yml"

    local env_section=""
    if [[ -n "${FRIGATE_URL:-}" ]] || [[ -n "${GO2RTC_URL:-}" ]]; then
        env_section="    environment:"
        [[ -n "${FRIGATE_URL:-}" ]] && env_section="${env_section}
      - STRIX_FRIGATE_URL=\${STRIX_FRIGATE_URL}"
        [[ -n "${GO2RTC_URL:-}" ]] && env_section="${env_section}
      - STRIX_GO2RTC_URL=\${STRIX_GO2RTC_URL}"
    fi

    cat > "${STRIX_DIR}/docker-compose.yml" <<EOF
# Strix -- Smart IP Camera Stream Finder
# Generated by install.sh -- do not edit manually, re-run installer to update.

services:
  strix:
    image: ${IMAGE}:\${STRIX_TAG:-latest}
    container_name: strix
    restart: unless-stopped
    network_mode: host
${env_section}
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:\${STRIX_PORT:-4567}/api/health"]
      interval: 30s
      timeout: 3s
      retries: 3
EOF

    log "Generated docker-compose.yml"
}

# ---------------------------------------------------------------------------
# Deploy
# ---------------------------------------------------------------------------
pull_image() {
    status_info "Pulling ${C_WHITE}${C_BOLD}${IMAGE}:${FINAL_VERSION}${C_RESET}..."
    log "Pulling image ${IMAGE}:${FINAL_VERSION}"

    if [[ "$VERBOSE" == true ]]; then
        $COMPOSE_CMD -f "${STRIX_DIR}/docker-compose.yml" --env-file "${STRIX_DIR}/.env" pull
    else
        $COMPOSE_CMD -f "${STRIX_DIR}/docker-compose.yml" --env-file "${STRIX_DIR}/.env" pull 2>&1 | tail -1
    fi

    if [[ $? -eq 0 ]]; then
        status_ok "Image pulled: ${C_WHITE}${C_BOLD}${IMAGE}:${FINAL_VERSION}${C_RESET}"
        log "Image pulled successfully"
    else
        status_fail "Failed to pull image ${IMAGE}:${FINAL_VERSION}"
        log "Image pull failed"
        exit 3
    fi
}

start_container() {
    log "Starting container"

    if [[ "$INSTALL_MODE" == "update" ]]; then
        status_info "Recreating container..."
        $COMPOSE_CMD -f "${STRIX_DIR}/docker-compose.yml" --env-file "${STRIX_DIR}/.env" up -d --force-recreate 2>&1 | \
            if [[ "$VERBOSE" == true ]]; then cat; else tail -1; fi
    else
        status_info "Starting container..."
        $COMPOSE_CMD -f "${STRIX_DIR}/docker-compose.yml" --env-file "${STRIX_DIR}/.env" up -d 2>&1 | \
            if [[ "$VERBOSE" == true ]]; then cat; else tail -1; fi
    fi

    if [[ $? -ne 0 ]]; then
        status_fail "Container failed to start"
        log "Container failed to start"
        exit 4
    fi

    log "Container started"
}

healthcheck() {
    status_info "Running healthcheck..."
    log "Waiting for healthcheck on localhost:${STRIX_PORT}"

    local retries=10
    local i
    for (( i = 1; i <= retries; i++ )); do
        if curl -sf --connect-timeout 2 --max-time 3 "http://localhost:${STRIX_PORT}/api/health" &>/dev/null; then
            local version
            version=$(curl -sf --max-time 3 "http://localhost:${STRIX_PORT}/api" 2>/dev/null | grep -oP '"version"\s*:\s*"\K[^"]+' || echo "unknown")
            status_ok "Strix is running ${C_WHITE}${C_BOLD}v${version}${C_RESET} on port ${C_WHITE}${C_BOLD}${STRIX_PORT}${C_RESET}"
            log "Healthcheck passed, version: ${version}"
            return 0
        fi
        sleep 1
    done

    status_fail "Healthcheck failed after ${retries} attempts"
    log "Healthcheck failed"
    exit 5
}

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
show_summary() {
    echo ""
    local w
    w=$(term_width)
    local line=""
    local box_w=$(( w - 4 ))
    [[ "$box_w" -gt 60 ]] && box_w=60
    for (( i = 0; i < box_w - 2; i++ )); do line="${line}─"; done
    local pad=$(( (w - box_w) / 2 ))
    [[ "$pad" -lt 0 ]] && pad=0
    local sp
    sp=$(printf "%*s" "$pad" "")

    echo -e "${sp}${C_GREEN}┌─ ${C_WHITE}${C_BOLD}Complete${C_RESET} ${C_GREEN}${line:11}┐${C_RESET}"
    echo -e "${sp}${C_GREEN}│$(printf "%*s" $(( box_w - 2 )) "")│${C_RESET}"
    echo -e "${sp}${C_GREEN}│${C_RESET}  Mode:     ${C_WHITE}${C_BOLD}${INSTALL_MODE}${C_RESET}$(printf "%*s" $(( box_w - 16 - ${#INSTALL_MODE} )) "")${C_GREEN}│${C_RESET}"
    echo -e "${sp}${C_GREEN}│${C_RESET}  Version:  ${C_WHITE}${C_BOLD}${FINAL_VERSION}${C_RESET}$(printf "%*s" $(( box_w - 16 - ${#FINAL_VERSION} )) "")${C_GREEN}│${C_RESET}"
    echo -e "${sp}${C_GREEN}│${C_RESET}  Port:     ${C_WHITE}${C_BOLD}${STRIX_PORT}${C_RESET}$(printf "%*s" $(( box_w - 16 - ${#STRIX_PORT} )) "")${C_GREEN}│${C_RESET}"
    echo -e "${sp}${C_GREEN}│${C_RESET}  Frigate:  ${C_WHITE}${FRIGATE_STATUS}${C_RESET}$(printf "%*s" $(( box_w - 16 - ${#FRIGATE_STATUS} )) "")${C_GREEN}│${C_RESET}"
    echo -e "${sp}${C_GREEN}│${C_RESET}  go2rtc:   ${C_WHITE}${GO2RTC_STATUS}${C_RESET}$(printf "%*s" $(( box_w - 16 - ${#GO2RTC_STATUS} )) "")${C_GREEN}│${C_RESET}"
    echo -e "${sp}${C_GREEN}│${C_RESET}  Config:   ${C_DIM}${STRIX_DIR}/${C_RESET}$(printf "%*s" $(( box_w - 16 - ${#STRIX_DIR} - 1 )) "")${C_GREEN}│${C_RESET}"
    echo -e "${sp}${C_GREEN}│${C_RESET}  Log:      ${C_DIM}${LOG_FILE}${C_RESET}$(printf "%*s" $(( box_w - 16 - ${#LOG_FILE} )) "")${C_GREEN}│${C_RESET}"
    echo -e "${sp}${C_GREEN}│$(printf "%*s" $(( box_w - 2 )) "")│${C_RESET}"
    echo -e "${sp}${C_GREEN}│${C_RESET}  ${C_CYAN}Open: ${C_WHITE}${C_BOLD}http://localhost:${STRIX_PORT}${C_RESET}$(printf "%*s" $(( box_w - 30 - ${#STRIX_PORT} )) "")${C_GREEN}│${C_RESET}"
    echo -e "${sp}${C_GREEN}│$(printf "%*s" $(( box_w - 2 )) "")│${C_RESET}"
    echo -e "${sp}${C_GREEN}└${line}┘${C_RESET}"
    echo ""

    # Machine-readable output for parent scripts (always last line)
    echo "STRIX_RESULT=OK MODE=${INSTALL_MODE} VERSION=${FINAL_VERSION} FRIGATE=${FRIGATE_STATUS} GO2RTC=${GO2RTC_STATUS}"
}

# ---------------------------------------------------------------------------
# Verbose log tail (shown on error)
# ---------------------------------------------------------------------------
show_error_log() {
    if [[ -f "$LOG_FILE" ]]; then
        echo ""
        echo -e "  ${C_RED}${C_BOLD}── Last log entries ──${C_RESET}"
        tail -20 "$LOG_FILE" | while IFS= read -r line; do
            echo -e "  ${C_RED}${line}${C_RESET}"
        done
        echo -e "  ${C_RED}${C_BOLD}──────────────────────${C_RESET}"
        echo ""
        echo -e "  Full log: ${C_WHITE}${LOG_FILE}${C_RESET}"
    fi
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
main() {
    # Trap errors to show log
    trap 'show_error_log' ERR

    # Initialize
    setup_colors
    ensure_root "$@"

    # Show logo
    show_logo

    # Detect system
    detect_system

    # Ensure curl is available
    ensure_curl

    # Docker
    if ! check_docker; then
        install_docker
    fi

    # Docker Compose
    if ! check_compose; then
        install_compose
    fi

    # Version selection
    select_version

    # Check existing installation
    check_existing || true

    # Detect services (only for fresh install)
    if [[ "$INSTALL_MODE" == "install" ]]; then
        probe_frigate
        probe_go2rtc
    else
        # For updates, load existing env
        if [[ -f "${STRIX_DIR}/.env" ]]; then
            # shellcheck disable=SC1091
            source "${STRIX_DIR}/.env" 2>/dev/null || true
            FRIGATE_URL="${STRIX_FRIGATE_URL:-}"
            GO2RTC_URL="${STRIX_GO2RTC_URL:-}"
            [[ -n "$FRIGATE_URL" ]] && FRIGATE_STATUS="set"
            [[ -n "$GO2RTC_URL" ]] && GO2RTC_STATUS="set"
        fi
    fi

    echo ""
    echo -e "  ${C_DIM}────────────────────────────────────────${C_RESET}"
    echo ""

    # Generate configs
    generate_env
    generate_compose

    # Deploy
    pull_image
    start_container
    healthcheck

    # Done
    show_summary
}

main "$@"

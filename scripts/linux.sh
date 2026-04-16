#!/usr/bin/env bash
# =============================================================================
# Strix -- linux.sh (navigator for plain Linux / macOS)
# =============================================================================

set -o pipefail

BACKTITLE="Strix Installer | Linux mode"
WT_H=16
WT_W=60

command -v whiptail &>/dev/null || { echo "whiptail required (install: apt install whiptail | dnf install newt)"; exit 1; }

# Parameters
INSTALL_MODE=""
FRIGATE_URL=""
GO2RTC_URL=""
STRIX_PORT="4567"
LOG_LEVEL=""
STRIX_TAG="latest"

# ---------------------------------------------------------------------------
# Simple flow
# ---------------------------------------------------------------------------
simple_flow() {
    local step=1
    while true; do
        case $step in
        1) # Mode
            INSTALL_MODE=$(whiptail --backtitle "$BACKTITLE" --title " Install Mode " \
                --menu "" $WT_H $WT_W 3 \
                "1" "Strix only" \
                "2" "Strix + Frigate" \
                "3" "Advanced setup" \
                3>&1 1>&2 2>&3) || { clear; exit 0; }

            case "$INSTALL_MODE" in
                1) INSTALL_MODE="strix"; step=2 ;;
                2) INSTALL_MODE="strix-frigate"; step=3 ;;
                3) advanced_flow; return ;;
            esac
            ;;

        2) # Frigate URL (strix only)
            FRIGATE_URL=$(whiptail --backtitle "$BACKTITLE" --title " Frigate " \
                --inputbox "Frigate URL (empty to skip):\n\nExample: http://192.168.1.100:5000" \
                $WT_H $WT_W "${FRIGATE_URL:-http://}" \
                3>&1 1>&2 2>&3) || { step=1; continue; }

            [[ "$FRIGATE_URL" == "http://" || "$FRIGATE_URL" == "https://" ]] && FRIGATE_URL=""
            step=3
            ;;

        3) # Confirm
            local s="Mode:     ${INSTALL_MODE}\nPort:     ${STRIX_PORT}\n"
            [[ -n "$FRIGATE_URL" ]] && s+="Frigate:  ${FRIGATE_URL}\n"

            whiptail --backtitle "$BACKTITLE" --title " Confirm " \
                --yesno "$s" $WT_H $WT_W || { step=1; continue; }
            break
            ;;
        esac
    done
}

# ---------------------------------------------------------------------------
# Advanced flow
# ---------------------------------------------------------------------------
advanced_flow() {
    local step=1
    INSTALL_MODE="${INSTALL_MODE:-strix}"

    while true; do
        case $step in
        1) # Mode
            local choice
            choice=$(whiptail --backtitle "$BACKTITLE" --title " Mode " \
                --menu "" $WT_H $WT_W 2 \
                "strix" "Strix only" \
                "strix-frigate" "Strix + Frigate" \
                --default-item "$INSTALL_MODE" \
                3>&1 1>&2 2>&3) || { clear; exit 0; }
            INSTALL_MODE="$choice"; step=2 ;;

        2) # Port
            local val
            val=$(whiptail --backtitle "$BACKTITLE" --title " Port " \
                --inputbox "Strix port:" 9 $WT_W "$STRIX_PORT" \
                3>&1 1>&2 2>&3) || { step=1; continue; }
            STRIX_PORT="${val:-4567}"; step=3 ;;

        3) # Frigate
            val=$(whiptail --backtitle "$BACKTITLE" --title " Frigate " \
                --inputbox "Frigate URL (empty to skip):" 9 $WT_W "${FRIGATE_URL:-http://}" \
                3>&1 1>&2 2>&3) || { step=1; continue; }
            [[ "$val" == "http://" || "$val" == "https://" ]] && val=""
            FRIGATE_URL="$val"; step=4 ;;

        4) # go2rtc
            val=$(whiptail --backtitle "$BACKTITLE" --title " go2rtc " \
                --inputbox "go2rtc URL (empty to skip):" 9 $WT_W "${GO2RTC_URL:-http://}" \
                3>&1 1>&2 2>&3) || { step=1; continue; }
            [[ "$val" == "http://" || "$val" == "https://" ]] && val=""
            GO2RTC_URL="$val"; step=5 ;;

        5) # Log level
            val=$(whiptail --backtitle "$BACKTITLE" --title " Log Level " \
                --menu "" 14 $WT_W 5 \
                ""      "default (info)" \
                "debug" "debug" \
                "info"  "info" \
                "warn"  "warn" \
                "error" "error" \
                3>&1 1>&2 2>&3) || { step=1; continue; }
            LOG_LEVEL="$val"; step=6 ;;

        6) # Tag
            val=$(whiptail --backtitle "$BACKTITLE" --title " Image Tag " \
                --inputbox "Strix image tag:" 9 $WT_W "$STRIX_TAG" \
                3>&1 1>&2 2>&3) || { step=1; continue; }
            STRIX_TAG="${val:-latest}"; step=7 ;;

        7) # Confirm
            local s="Mode:     ${INSTALL_MODE}\nPort:     ${STRIX_PORT}\nTag:      ${STRIX_TAG}\n"
            [[ -n "$FRIGATE_URL" ]] && s+="Frigate:  ${FRIGATE_URL}\n"
            [[ -n "$GO2RTC_URL" ]]  && s+="go2rtc:   ${GO2RTC_URL}\n"
            [[ -n "$LOG_LEVEL" ]]   && s+="Log:      ${LOG_LEVEL}\n"

            whiptail --backtitle "$BACKTITLE" --title " Confirm " \
                --yesno "$s" $WT_H $WT_W || { step=1; continue; }
            break
            ;;
        esac
    done
}

# ---------------------------------------------------------------------------
# Colors
# ---------------------------------------------------------------------------
C_RESET="\033[0m"
C_BOLD="\033[1m"
C_DIM="\033[2m"
C_GREEN="\033[32m"
C_RED="\033[31m"
C_YELLOW="\033[33m"
C_CYAN="\033[36m"
C_WHITE="\033[97m"
C_MAGENTA="\033[35m"

# ---------------------------------------------------------------------------
# Worker runner: streams JSON events as status lines
# ---------------------------------------------------------------------------
SCRIPTS_BASE="https://raw.githubusercontent.com/eduard256/Strix/main/scripts"

download_worker() {
    local name="$1"
    local dest="/tmp/strix-${name}"
    curl -fsSL "${SCRIPTS_BASE}/${name}" -o "$dest" 2>/dev/null
    echo "$dest"
}

print_events() {
    while IFS= read -r line; do
        type=""; msg=""
        type=$(echo "$line" | grep -oP '"type"\s*:\s*"\K[^"]+' | head -1)
        msg=$(echo "$line" | grep -oP '"msg"\s*:\s*"\K[^"]+' | head -1)
        case "$type" in
            check)   echo -e "  ${C_CYAN}[..]${C_RESET}  ${msg}" ;;
            ok)      echo -e "  ${C_GREEN}[OK]${C_RESET}  ${msg}" ;;
            miss)    echo -e "  ${C_YELLOW}[--]${C_RESET}  ${msg}" ;;
            install) echo -e "  ${C_CYAN}[>>]${C_RESET}  ${msg}" ;;
            error)   echo -e "  ${C_RED}[XX]${C_RESET}  ${msg}" ;;
        esac
    done
}

json_field() {
    echo "$1" | grep -oP "\"$2\"\s*:\s*\"\K[^\"]*" | head -1
}

# ---------------------------------------------------------------------------
# LAN IP detection
# ---------------------------------------------------------------------------
detect_lan_ip() {
    local ip=""
    ip=$(ip route get 1.1.1.1 2>/dev/null | grep -oP 'src \K\S+' | head -1)
    [[ -z "$ip" ]] && ip=$(hostname -I 2>/dev/null | awk '{print $1}')
    [[ -z "$ip" ]] && ip=$(ifconfig 2>/dev/null | grep -oP 'inet \K[0-9.]+' | grep -v '127.0.0.1' | head -1)
    [[ -z "$ip" ]] && ip="localhost"
    echo "$ip"
}

# ---------------------------------------------------------------------------
# Final URLs
# ---------------------------------------------------------------------------
show_urls() {
    local ip="$1"
    local port="$2"
    local mode="$3"

    echo ""
    echo -e "  ${C_GREEN}${C_BOLD}====================================${C_RESET}"
    echo -e "  ${C_GREEN}${C_BOLD}  Installation Complete${C_RESET}"
    echo -e "  ${C_GREEN}${C_BOLD}====================================${C_RESET}"
    echo ""
    echo -e "  ${C_WHITE}${C_BOLD}Strix:${C_RESET}           ${C_CYAN}http://${ip}:${port}${C_RESET}"

    if [[ "$mode" == "strix-frigate" ]]; then
        echo -e "  ${C_WHITE}${C_BOLD}Frigate:${C_RESET}         ${C_CYAN}http://${ip}:8971${C_RESET}"
        echo -e "  ${C_WHITE}${C_BOLD}Frigate API:${C_RESET}     ${C_CYAN}http://${ip}:5000${C_RESET}"
        echo -e "  ${C_WHITE}${C_BOLD}go2rtc:${C_RESET}          ${C_CYAN}http://${ip}:1984${C_RESET}"
    fi

    echo ""
    echo -e "  ${C_DIM}Press Enter to exit${C_RESET}"
    read -r
}

# ---------------------------------------------------------------------------
# Check root / docker -- bail early if not sudo and docker missing
# ---------------------------------------------------------------------------
check_sudo_required() {
    if [[ "$(id -u)" -eq 0 ]]; then
        return 0  # already root
    fi

    if command -v docker &>/dev/null; then
        return 0  # docker present, maybe root not strictly needed
    fi

    clear
    echo ""
    echo -e "  ${C_RED}${C_BOLD}Root privileges required${C_RESET}"
    echo ""
    echo -e "  Docker is not installed. Installing it needs root."
    echo -e "  Please re-run the installer with ${C_BOLD}sudo${C_RESET}:"
    echo ""
    echo -e "  ${C_CYAN}${C_BOLD}curl -fsSL https://raw.githubusercontent.com/eduard256/Strix/main/scripts/install.sh | sudo bash${C_RESET}"
    echo ""
    exit 1
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
simple_flow

clear
echo ""
echo -e "  ${C_MAGENTA}${C_BOLD}STRIX INSTALLER${C_RESET}  ${C_DIM}(Linux)${C_RESET}"
echo -e "  ${C_DIM}Mode: ${INSTALL_MODE} | Port: ${STRIX_PORT}${C_RESET}"
echo ""

check_sudo_required

# Step 1: Check Docker / install via prepare.sh
if ! command -v docker &>/dev/null || ! docker compose version &>/dev/null; then
    echo -e "  ${C_MAGENTA}${C_BOLD}--- Installing Docker ---${C_RESET}"
    echo ""

    prepare_script=$(download_worker "prepare.sh")
    bash "$prepare_script" 2>/dev/null | print_events
    rm -f "$prepare_script"
    echo ""
fi

# Step 2: Deploy
if [[ "$INSTALL_MODE" == "strix-frigate" ]]; then
    echo -e "  ${C_MAGENTA}${C_BOLD}--- Deploying Strix + Frigate ---${C_RESET}"
    echo ""

    deploy_script=$(download_worker "strix-frigate.sh")
    deploy_args="--port $STRIX_PORT --tag $STRIX_TAG"
    [[ -n "$GO2RTC_URL" ]] && deploy_args="$deploy_args --go2rtc-url $GO2RTC_URL"
    [[ -n "$LOG_LEVEL" ]]  && deploy_args="$deploy_args --log-level $LOG_LEVEL"

    deploy_output=$(bash "$deploy_script" $deploy_args 2>/dev/null)
    deploy_done=$(echo "$deploy_output" | grep '"type":"done"')
    echo "$deploy_output" | print_events
    rm -f "$deploy_script"

else
    echo -e "  ${C_MAGENTA}${C_BOLD}--- Deploying Strix ---${C_RESET}"
    echo ""

    deploy_script=$(download_worker "strix.sh")
    deploy_args="--port $STRIX_PORT --tag $STRIX_TAG"
    [[ -n "$FRIGATE_URL" ]] && deploy_args="$deploy_args --frigate-url $FRIGATE_URL"
    [[ -n "$GO2RTC_URL" ]]  && deploy_args="$deploy_args --go2rtc-url $GO2RTC_URL"
    [[ -n "$LOG_LEVEL" ]]   && deploy_args="$deploy_args --log-level $LOG_LEVEL"

    deploy_output=$(bash "$deploy_script" $deploy_args 2>/dev/null)
    deploy_done=$(echo "$deploy_output" | grep '"type":"done"')
    echo "$deploy_output" | print_events
    rm -f "$deploy_script"
fi

# Final URLs
deploy_ok=$(echo "$deploy_done" | grep -oP '"ok"\s*:\s*\K[a-z]+' | head -1)
if [[ "$deploy_ok" == "true" ]]; then
    lan_ip=$(detect_lan_ip)
    show_urls "$lan_ip" "$STRIX_PORT" "$INSTALL_MODE"
else
    echo ""
    echo -e "  ${C_RED}${C_BOLD}Deployment failed.${C_RESET}"
    echo ""
fi

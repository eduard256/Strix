#!/usr/bin/env bash
# =============================================================================
# Strix -- proxmox.sh (navigator for Proxmox)
# =============================================================================

set -o pipefail

BACKTITLE="Strix Installer | Proxmox mode"
WT_H=16
WT_W=60

command -v whiptail &>/dev/null || { echo "whiptail required"; exit 1; }

# Dark theme for whiptail
export NEWT_COLORS='
root=,black
window=,black
border=white,black
textbox=white,black
button=black,white
actbutton=white,magenta
compactbutton=white,black
listbox=white,black
actlistbox=white,magenta
title=magenta,black
roottext=white,black
emptyscale=,black
fullscale=,magenta
helpline=white,black
'

# Parameters
INSTALL_MODE=""
FRIGATE_URL=""
GO2RTC_URL=""
STRIX_PORT="4567"
LXC_HOSTNAME="strix"
LXC_MEMORY="2048"
LXC_CORES="2"
LXC_DISK="32"
LXC_SWAP="512"
LXC_IP="dhcp"
LXC_GATEWAY=""
LXC_BRIDGE=""
LXC_STORAGE=""

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
            s+="\nLXC: auto (${LXC_HOSTNAME}, ${LXC_MEMORY}MB, ${LXC_CORES}cpu, ${LXC_DISK}GB)"

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

        5) # Hostname
            val=$(whiptail --backtitle "$BACKTITLE" --title " LXC Hostname " \
                --inputbox "Hostname:" 9 $WT_W "$LXC_HOSTNAME" \
                3>&1 1>&2 2>&3) || { step=1; continue; }
            LXC_HOSTNAME="${val:-strix}"; step=6 ;;

        6) # RAM
            val=$(whiptail --backtitle "$BACKTITLE" --title " LXC RAM " \
                --inputbox "RAM (MB):" 9 $WT_W "$LXC_MEMORY" \
                3>&1 1>&2 2>&3) || { step=1; continue; }
            LXC_MEMORY="${val:-2048}"; step=7 ;;

        7) # CPU
            val=$(whiptail --backtitle "$BACKTITLE" --title " LXC CPU " \
                --inputbox "CPU cores:" 9 $WT_W "$LXC_CORES" \
                3>&1 1>&2 2>&3) || { step=1; continue; }
            LXC_CORES="${val:-2}"; step=8 ;;

        8) # Disk
            val=$(whiptail --backtitle "$BACKTITLE" --title " LXC Disk " \
                --inputbox "Disk (GB):" 9 $WT_W "$LXC_DISK" \
                3>&1 1>&2 2>&3) || { step=1; continue; }
            LXC_DISK="${val:-32}"; step=9 ;;

        9) # Swap
            val=$(whiptail --backtitle "$BACKTITLE" --title " LXC Swap " \
                --inputbox "Swap (MB):" 9 $WT_W "$LXC_SWAP" \
                3>&1 1>&2 2>&3) || { step=1; continue; }
            LXC_SWAP="${val:-512}"; step=10 ;;

        10) # IP
            val=$(whiptail --backtitle "$BACKTITLE" --title " LXC Network " \
                --inputbox "IP (dhcp or CIDR e.g. 10.0.20.110/24):" 9 $WT_W "$LXC_IP" \
                3>&1 1>&2 2>&3) || { step=1; continue; }
            LXC_IP="${val:-dhcp}"
            [[ "$LXC_IP" != "dhcp" ]] && step=11 || step=12
            ;;

        11) # Gateway
            val=$(whiptail --backtitle "$BACKTITLE" --title " Gateway " \
                --inputbox "Gateway:" 9 $WT_W "$LXC_GATEWAY" \
                3>&1 1>&2 2>&3) || { step=1; continue; }
            LXC_GATEWAY="$val"; step=12 ;;

        12) # Bridge
            val=$(whiptail --backtitle "$BACKTITLE" --title " Bridge " \
                --inputbox "Network bridge (empty=auto):" 9 $WT_W "$LXC_BRIDGE" \
                3>&1 1>&2 2>&3) || { step=1; continue; }
            LXC_BRIDGE="$val"; step=13 ;;

        13) # Storage
            val=$(whiptail --backtitle "$BACKTITLE" --title " Storage " \
                --inputbox "Storage (empty=auto):" 9 $WT_W "$LXC_STORAGE" \
                3>&1 1>&2 2>&3) || { step=1; continue; }
            LXC_STORAGE="$val"; step=14 ;;

        14) # Confirm
            local s="Mode:     ${INSTALL_MODE}\nPort:     ${STRIX_PORT}\n"
            [[ -n "$FRIGATE_URL" ]] && s+="Frigate:  ${FRIGATE_URL}\n"
            [[ -n "$GO2RTC_URL" ]]  && s+="go2rtc:   ${GO2RTC_URL}\n"
            s+="\nLXC:\n"
            s+="  ${LXC_HOSTNAME} | ${LXC_MEMORY}MB | ${LXC_CORES}cpu | ${LXC_DISK}GB\n"
            s+="  IP: ${LXC_IP}"
            [[ -n "$LXC_GATEWAY" ]] && s+=" gw ${LXC_GATEWAY}"
            s+="\n"
            [[ -n "$LXC_BRIDGE" ]]  && s+="  Bridge: ${LXC_BRIDGE}\n" || s+="  Bridge: auto\n"
            [[ -n "$LXC_STORAGE" ]] && s+="  Storage: ${LXC_STORAGE}\n" || s+="  Storage: auto\n"

            whiptail --backtitle "$BACKTITLE" --title " Confirm " \
                --yesno "$s" 18 $WT_W || { step=1; continue; }
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
# Worker runner: streams JSON events and prints status lines
# ---------------------------------------------------------------------------
SCRIPTS_BASE="https://raw.githubusercontent.com/eduard256/Strix/main/scripts"

# Download a worker script to /tmp
download_worker() {
    local name="$1"
    local dest="/tmp/strix-${name}"
    curl -fsSL "${SCRIPTS_BASE}/${name}" -o "$dest" 2>/dev/null
    echo "$dest"
}

# Run a worker and display its JSON events as status lines
run_worker() {
    local script="$1"
    shift
    local label="$1"
    shift

    echo ""
    echo -e "  ${C_MAGENTA}${C_BOLD}--- ${label} ---${C_RESET}"
    echo ""

    bash "$script" "$@" 2>/dev/null | while IFS= read -r line; do
        type=""; msg=""
        type=$(echo "$line" | grep -oP '"type"\s*:\s*"\K[^"]+' | head -1)
        msg=$(echo "$line" | grep -oP '"msg"\s*:\s*"\K[^"]+' | head -1)

        case "$type" in
            check)   echo -e "  ${C_CYAN}[..]${C_RESET}  ${msg}" ;;
            ok)      echo -e "  ${C_GREEN}[OK]${C_RESET}  ${msg}" ;;
            miss)    echo -e "  ${C_YELLOW}[--]${C_RESET}  ${msg}" ;;
            install) echo -e "  ${C_CYAN}[>>]${C_RESET}  ${msg}" ;;
            error)   echo -e "  ${C_RED}[XX]${C_RESET}  ${msg}" ;;
            done)    ;; # handled after loop
        esac
    done

    echo ""
}

# Extract a field from JSON done line
json_field() {
    echo "$1" | grep -oP "\"$2\"\s*:\s*\"\K[^\"]*" | head -1
}

# ---------------------------------------------------------------------------
# Show final URLs
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
# Main
# ---------------------------------------------------------------------------
simple_flow

clear
echo ""
echo -e "  ${C_MAGENTA}${C_BOLD}STRIX INSTALLER${C_RESET}"
echo -e "  ${C_DIM}Mode: ${INSTALL_MODE} | Port: ${STRIX_PORT}${C_RESET}"
echo ""

# Step 1: Create LXC container
echo -e "  ${C_MAGENTA}${C_BOLD}--- Creating LXC Container ---${C_RESET}"
echo ""

lxc_script=$(download_worker "proxmox-lxc-create.sh")

lxc_args=""
[[ -n "$LXC_HOSTNAME" ]] && lxc_args="$lxc_args --hostname $LXC_HOSTNAME"
[[ -n "$LXC_MEMORY" ]]   && lxc_args="$lxc_args --memory $LXC_MEMORY"
[[ -n "$LXC_CORES" ]]    && lxc_args="$lxc_args --cores $LXC_CORES"
[[ -n "$LXC_DISK" ]]     && lxc_args="$lxc_args --disk $LXC_DISK"
[[ -n "$LXC_SWAP" ]]     && lxc_args="$lxc_args --swap $LXC_SWAP"
[[ -n "$LXC_BRIDGE" ]]   && lxc_args="$lxc_args --bridge $LXC_BRIDGE"
[[ -n "$LXC_STORAGE" ]]  && lxc_args="$lxc_args --storage $LXC_STORAGE"
[[ "$LXC_IP" != "dhcp" && -n "$LXC_IP" ]] && lxc_args="$lxc_args --ip $LXC_IP"
[[ -n "$LXC_GATEWAY" ]]  && lxc_args="$lxc_args --gateway $LXC_GATEWAY"

# Run LXC creation and capture full output
lxc_output=$(bash "$lxc_script" $lxc_args 2>/dev/null)
lxc_done=$(echo "$lxc_output" | grep '"type":"done"')

# Display LXC creation events
echo "$lxc_output" | while IFS= read -r line; do
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

# Check if LXC creation succeeded
lxc_ok=$(echo "$lxc_done" | grep -oP '"ok"\s*:\s*\K[a-z]+' | head -1)
if [[ "$lxc_ok" != "true" ]]; then
    echo ""
    echo -e "  ${C_RED}${C_BOLD}LXC creation failed. Aborting.${C_RESET}"
    rm -f "$lxc_script"
    exit 1
fi

# Extract LXC data
CT_ID=$(json_field "$lxc_done" "id")
CT_IP=$(json_field "$lxc_done" "ip")
CT_PASS=$(json_field "$lxc_done" "password")

echo ""
echo -e "  ${C_GREEN}${C_BOLD}LXC ${CT_ID} ready${C_RESET} -- IP: ${C_WHITE}${CT_IP}${C_RESET}"

# Step 2: Run prepare.sh inside LXC (install Docker)
echo ""
echo -e "  ${C_MAGENTA}${C_BOLD}--- Installing Docker ---${C_RESET}"
echo ""

prepare_script=$(download_worker "prepare.sh")
pct push "$CT_ID" "$prepare_script" /tmp/prepare.sh &>/dev/null

pct exec "$CT_ID" -- bash /tmp/prepare.sh 2>/dev/null | while IFS= read -r line; do
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

# Step 3: Deploy Strix (or Strix + Frigate)
if [[ "$INSTALL_MODE" == "strix-frigate" ]]; then
    echo ""
    echo -e "  ${C_MAGENTA}${C_BOLD}--- Deploying Strix + Frigate ---${C_RESET}"
    echo ""

    deploy_script=$(download_worker "strix-frigate.sh")
    pct push "$CT_ID" "$deploy_script" /tmp/deploy.sh &>/dev/null

    deploy_args="--port $STRIX_PORT"
    [[ -n "$GO2RTC_URL" ]] && deploy_args="$deploy_args --go2rtc-url $GO2RTC_URL"

    deploy_output=$(pct exec "$CT_ID" -- bash /tmp/deploy.sh $deploy_args 2>/dev/null)
    deploy_done=$(echo "$deploy_output" | grep '"type":"done"')

    echo "$deploy_output" | while IFS= read -r line; do
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

else
    echo ""
    echo -e "  ${C_MAGENTA}${C_BOLD}--- Deploying Strix ---${C_RESET}"
    echo ""

    deploy_script=$(download_worker "strix.sh")
    pct push "$CT_ID" "$deploy_script" /tmp/deploy.sh &>/dev/null

    deploy_args="--port $STRIX_PORT"
    [[ -n "$FRIGATE_URL" ]] && deploy_args="$deploy_args --frigate-url $FRIGATE_URL"
    [[ -n "$GO2RTC_URL" ]]  && deploy_args="$deploy_args --go2rtc-url $GO2RTC_URL"

    deploy_output=$(pct exec "$CT_ID" -- bash /tmp/deploy.sh $deploy_args 2>/dev/null)
    deploy_done=$(echo "$deploy_output" | grep '"type":"done"')

    echo "$deploy_output" | while IFS= read -r line; do
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
fi

# Show final URLs
deploy_ok=$(echo "$deploy_done" | grep -oP '"ok"\s*:\s*\K[a-z]+' | head -1)
if [[ "$deploy_ok" == "true" ]]; then
    show_urls "$CT_IP" "$STRIX_PORT" "$INSTALL_MODE"
else
    echo ""
    echo -e "  ${C_RED}${C_BOLD}Deployment failed.${C_RESET}"
    echo -e "  ${C_DIM}LXC ${CT_ID} (${CT_IP}) is still running. Check logs inside.${C_RESET}"
    echo ""
fi

# Cleanup
rm -f "$lxc_script" "$prepare_script" "$deploy_script" 2>/dev/null

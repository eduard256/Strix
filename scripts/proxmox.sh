#!/usr/bin/env bash
# =============================================================================
# Strix -- proxmox.sh (navigator for Proxmox)
# =============================================================================

set -o pipefail

BACKTITLE="Strix Installer"
WT_H=16
WT_W=60

command -v whiptail &>/dev/null || { echo "whiptail required"; exit 1; }

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
# Main
# ---------------------------------------------------------------------------
simple_flow

clear
echo "Mode:      $INSTALL_MODE"
echo "Port:      $STRIX_PORT"
[[ -n "$FRIGATE_URL" ]] && echo "Frigate:   $FRIGATE_URL"
[[ -n "$GO2RTC_URL" ]]  && echo "go2rtc:    $GO2RTC_URL"
echo "LXC:       ${LXC_HOSTNAME} ${LXC_MEMORY}MB ${LXC_CORES}cpu ${LXC_DISK}GB"
echo "Network:   ${LXC_IP} ${LXC_GATEWAY:+gw $LXC_GATEWAY}"
echo "Bridge:    ${LXC_BRIDGE:-auto}"
echo "Storage:   ${LXC_STORAGE:-auto}"
echo ""
echo "(Workers would run here)"

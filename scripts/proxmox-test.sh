#!/usr/bin/env bash
SCRIPT_URL="https://raw.githubusercontent.com/eduard256/Strix/main/scripts/proxmox.sh"

whiptail --title " Strix " --yesno "Run Strix installer?" 8 40 || { clear; exit 0; }

tmpfile=$(mktemp /tmp/strix-proxmox.XXXXXX)
curl -fsSL "$SCRIPT_URL" -o "$tmpfile" 2>/dev/null || { echo "Download failed"; exit 1; }
bash "$tmpfile"
rm -f "$tmpfile"

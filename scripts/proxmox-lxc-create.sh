#!/usr/bin/env bash
# =============================================================================
# Strix -- proxmox-lxc-create.sh (worker)
#
# Creates an unprivileged Ubuntu LXC container on Proxmox with Docker support.
# Runs ON the Proxmox host. Uses only official CLI tools (pct, pveam, pvesm).
# Does NOT install anything inside the container -- just creates and starts it.
#
# Protocol:
#   - Every action is reported as a single-line JSON to stdout.
#   - Types: check, ok, miss, install, error, done
#   - Last line: {"type":"done","ok":true,"data":{...}} or {"type":"done","ok":false,"error":"..."}
#   - Exit code: 0 = success, 1 = failure.
#
# Parameters (all optional):
#   --id ID                  Container ID (default: auto, next free)
#   --hostname NAME          Hostname (default: strix)
#   --memory MB              RAM in MB (default: 2048)
#   --swap MB                Swap in MB (default: 512)
#   --disk GB                Disk size in GB (default: 32)
#   --cores N                CPU cores (default: 2)
#   --storage NAME           Storage for container disk (default: auto)
#   --bridge NAME            Network bridge (default: auto, first vmbr*)
#   --ip CIDR                IP address, e.g. 10.0.99.110/24 (default: dhcp)
#   --gateway IP             Gateway (required if --ip is static)
#   --password PASS          Root password (default: auto-generated)
#
# Usage:
#   bash scripts/proxmox-lxc-create.sh
#   bash scripts/proxmox-lxc-create.sh --hostname strix --memory 4096 --cores 4
# =============================================================================

set -uo pipefail

# ---------------------------------------------------------------------------
# Defaults
# ---------------------------------------------------------------------------
CT_ID=""
CT_HOSTNAME="strix"
CT_MEMORY="2048"
CT_SWAP="512"
CT_DISK="32"
CT_CORES="2"
CT_STORAGE=""
CT_BRIDGE=""
CT_IP="dhcp"
CT_GATEWAY=""
CT_PASSWORD=""

TEMPLATE_STORAGE=""
TEMPLATE=""

# ---------------------------------------------------------------------------
# Parse CLI arguments
# ---------------------------------------------------------------------------
while [[ $# -gt 0 ]]; do
    case "$1" in
        --id)        CT_ID="$2";        shift 2 ;;
        --hostname)  CT_HOSTNAME="$2";  shift 2 ;;
        --memory)    CT_MEMORY="$2";    shift 2 ;;
        --swap)      CT_SWAP="$2";      shift 2 ;;
        --disk)      CT_DISK="$2";      shift 2 ;;
        --cores)     CT_CORES="$2";     shift 2 ;;
        --storage)   CT_STORAGE="$2";   shift 2 ;;
        --bridge)    CT_BRIDGE="$2";    shift 2 ;;
        --ip)        CT_IP="$2";        shift 2 ;;
        --gateway)   CT_GATEWAY="$2";   shift 2 ;;
        --password)  CT_PASSWORD="$2";  shift 2 ;;
        *)           shift ;;
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

# Cleanup on failure: destroy container if it was partially created
cleanup_on_fail() {
    local id="$1"
    local msg="$2"
    if pct status "$id" &>/dev/null; then
        pct stop "$id" &>/dev/null || true
        pct destroy "$id" --purge &>/dev/null || true
        emit "ok" "Rolled back: container ${id} destroyed"
    fi
    emit "error" "$msg"
    emit_done_fail "$msg"
}

# ---------------------------------------------------------------------------
# 1. Verify Proxmox environment
# ---------------------------------------------------------------------------
check_proxmox() {
    emit "check" "Verifying Proxmox environment"

    if ! command -v pct &>/dev/null; then
        emit "error" "pct not found -- this script must run on a Proxmox host"
        emit_done_fail "Not a Proxmox host"
    fi

    if ! command -v pveam &>/dev/null; then
        emit "error" "pveam not found -- this script must run on a Proxmox host"
        emit_done_fail "Not a Proxmox host"
    fi

    local pve_ver
    pve_ver=$(pveversion 2>/dev/null | grep -oP 'pve-manager/\K[0-9]+\.[0-9]+' || echo "unknown")
    emit "ok" "Proxmox VE ${pve_ver}" "{\"pve_version\":\"${pve_ver}\"}"
}

# ---------------------------------------------------------------------------
# 2. Auto-detect container ID
# ---------------------------------------------------------------------------
resolve_ct_id() {
    emit "check" "Resolving container ID"

    if [[ -n "$CT_ID" ]]; then
        # Verify it's free
        if pct status "$CT_ID" &>/dev/null || qm status "$CT_ID" &>/dev/null; then
            emit "error" "Container/VM ID ${CT_ID} is already in use"
            emit_done_fail "CT ID ${CT_ID} already in use"
        fi
        emit "ok" "Using specified ID: ${CT_ID}"
    else
        CT_ID=$(pvesh get /cluster/nextid 2>/dev/null || echo "")
        if [[ -z "$CT_ID" ]]; then
            emit "error" "Failed to get next free container ID"
            emit_done_fail "Cannot get next free CT ID"
        fi
        # Double-check it's actually free
        if pct status "$CT_ID" &>/dev/null || qm status "$CT_ID" &>/dev/null; then
            CT_ID=$((CT_ID + 1))
        fi
        emit "ok" "Auto-assigned ID: ${CT_ID}" "{\"id\":\"${CT_ID}\"}"
    fi
}

# ---------------------------------------------------------------------------
# 3. Auto-detect storage
# ---------------------------------------------------------------------------
resolve_storage() {
    # Container storage (rootdir)
    emit "check" "Resolving container storage"

    if [[ -n "$CT_STORAGE" ]]; then
        if ! pvesm status 2>/dev/null | awk 'NR>1{print $1}' | grep -qx "$CT_STORAGE"; then
            emit "error" "Storage '${CT_STORAGE}' not found"
            emit_done_fail "Storage ${CT_STORAGE} not found"
        fi
        emit "ok" "Using specified storage: ${CT_STORAGE}"
    else
        # Find first storage that supports rootdir content
        CT_STORAGE=$(pvesm status -content rootdir 2>/dev/null | awk 'NR>1 && $2=="active"{print $1; exit}')
        if [[ -z "$CT_STORAGE" ]]; then
            # Fallback: try local-lvm, then local
            if pvesm status 2>/dev/null | awk 'NR>1{print $1}' | grep -qx "local-lvm"; then
                CT_STORAGE="local-lvm"
            else
                CT_STORAGE="local"
            fi
        fi
        emit "ok" "Auto-detected storage: ${CT_STORAGE}" "{\"storage\":\"${CT_STORAGE}\"}"
    fi

    # Template storage (vztmpl)
    emit "check" "Resolving template storage"
    TEMPLATE_STORAGE=$(pvesm status -content vztmpl 2>/dev/null | awk 'NR>1 && $2=="active"{print $1; exit}')
    if [[ -z "$TEMPLATE_STORAGE" ]]; then
        TEMPLATE_STORAGE="local"
    fi
    emit "ok" "Template storage: ${TEMPLATE_STORAGE}" "{\"template_storage\":\"${TEMPLATE_STORAGE}\"}"
}

# ---------------------------------------------------------------------------
# 4. Check free space
# ---------------------------------------------------------------------------
check_free_space() {
    emit "check" "Checking free space on ${CT_STORAGE}"

    local avail_kb
    avail_kb=$(pvesm status 2>/dev/null | awk -v s="$CT_STORAGE" '$1==s{print $6}')

    if [[ -n "$avail_kb" ]]; then
        local avail_gb=$((avail_kb / 1024 / 1024))
        local required_gb=$CT_DISK

        if [[ "$avail_gb" -lt "$required_gb" ]]; then
            emit "error" "Not enough space: ${avail_gb}GB available, ${required_gb}GB required"
            emit_done_fail "Not enough disk space on ${CT_STORAGE}"
        fi
        emit "ok" "${avail_gb}GB available, ${required_gb}GB required"
    else
        emit "ok" "Could not determine free space, proceeding"
    fi
}

# ---------------------------------------------------------------------------
# 5. Auto-detect network bridge
# ---------------------------------------------------------------------------
resolve_bridge() {
    emit "check" "Resolving network bridge"

    if [[ -n "$CT_BRIDGE" ]]; then
        emit "ok" "Using specified bridge: ${CT_BRIDGE}"
        return
    fi

    # Find first vmbr* interface
    CT_BRIDGE=$(ip link show 2>/dev/null | grep -oP 'vmbr\d+' | head -1)

    if [[ -z "$CT_BRIDGE" ]]; then
        CT_BRIDGE="vmbr0"
        emit "ok" "Defaulting to bridge: vmbr0"
    else
        emit "ok" "Auto-detected bridge: ${CT_BRIDGE}" "{\"bridge\":\"${CT_BRIDGE}\"}"
    fi
}

# ---------------------------------------------------------------------------
# 6. Generate password
# ---------------------------------------------------------------------------
resolve_password() {
    if [[ -n "$CT_PASSWORD" ]]; then
        return
    fi

    emit "check" "Generating root password"
    CT_PASSWORD=$(openssl rand -base64 12 2>/dev/null | tr -d '/+=' | head -c 16)
    if [[ -z "$CT_PASSWORD" ]]; then
        # Fallback if openssl not available
        CT_PASSWORD=$(head -c 32 /dev/urandom | base64 | tr -d '/+=' | head -c 16)
    fi
    emit "ok" "Root password generated"
}

# ---------------------------------------------------------------------------
# 7. Download Ubuntu template
# ---------------------------------------------------------------------------
download_template() {
    emit "check" "Searching for Ubuntu template"

    # Check if already downloaded locally
    TEMPLATE=$(pveam list "$TEMPLATE_STORAGE" 2>/dev/null \
        | awk '$1 ~ /ubuntu-24\.04.*-standard_/ {print $1}' \
        | sed 's|.*/||' \
        | sort -V \
        | tail -1)

    if [[ -n "$TEMPLATE" ]]; then
        emit "ok" "Template found locally: ${TEMPLATE}"
        return
    fi

    # Not local, try online
    emit "miss" "No local Ubuntu 24.04 template"
    emit "install" "Updating template catalog"

    if command -v timeout &>/dev/null; then
        timeout 30 pveam update &>/dev/null || true
    else
        pveam update &>/dev/null || true
    fi

    # Search for Ubuntu 24.04
    TEMPLATE=$(pveam available --section system 2>/dev/null \
        | awk '$2 ~ /ubuntu-24\.04.*-standard_/ {print $2}' \
        | sort -V \
        | tail -1)

    # Fallback to 22.04
    if [[ -z "$TEMPLATE" ]]; then
        emit "miss" "Ubuntu 24.04 not available, trying 22.04"
        TEMPLATE=$(pveam available --section system 2>/dev/null \
            | awk '$2 ~ /ubuntu-22\.04.*-standard_/ {print $2}' \
            | sort -V \
            | tail -1)
    fi

    if [[ -z "$TEMPLATE" ]]; then
        emit "error" "No Ubuntu template found"
        emit_done_fail "No Ubuntu template available"
    fi

    emit "install" "Downloading template: ${TEMPLATE}"

    local attempt
    for attempt in 1 2 3; do
        if pveam download "$TEMPLATE_STORAGE" "$TEMPLATE" &>/dev/null; then
            emit "ok" "Template downloaded: ${TEMPLATE}"
            return
        fi
        if [[ "$attempt" -lt 3 ]]; then
            emit "check" "Download failed, retrying (${attempt}/3)"
            sleep $((attempt * 5))
        fi
    done

    emit "error" "Template download failed after 3 attempts"
    emit_done_fail "Template download failed"
}

# ---------------------------------------------------------------------------
# 8. Ensure subuid/subgid (required for unprivileged containers)
# ---------------------------------------------------------------------------
fix_subuid_subgid() {
    emit "check" "Checking subuid/subgid mappings"

    local changed=false

    if ! grep -q "root:100000:65536" /etc/subuid 2>/dev/null; then
        echo "root:100000:65536" >> /etc/subuid
        changed=true
    fi

    if ! grep -q "root:100000:65536" /etc/subgid 2>/dev/null; then
        echo "root:100000:65536" >> /etc/subgid
        changed=true
    fi

    if [[ "$changed" == true ]]; then
        emit "ok" "subuid/subgid mappings added"
    else
        emit "ok" "subuid/subgid mappings present"
    fi
}

# ---------------------------------------------------------------------------
# 9. Create container
# ---------------------------------------------------------------------------
create_container() {
    emit "install" "Creating LXC container ${CT_ID}"

    # Build network string
    local net_string="name=eth0,bridge=${CT_BRIDGE}"
    if [[ "$CT_IP" == "dhcp" ]]; then
        net_string="${net_string},ip=dhcp,ip6=dhcp"
    else
        net_string="${net_string},ip=${CT_IP}"
        [[ -n "$CT_GATEWAY" ]] && net_string="${net_string},gw=${CT_GATEWAY}"
    fi

    local pct_cmd=(
        pct create "$CT_ID" "${TEMPLATE_STORAGE}:vztmpl/${TEMPLATE}"
        -hostname "$CT_HOSTNAME"
        -cores "$CT_CORES"
        -memory "$CT_MEMORY"
        -swap "$CT_SWAP"
        -rootfs "${CT_STORAGE}:${CT_DISK}"
        -net0 "$net_string"
        -features "nesting=1,keyctl=1"
        -unprivileged 1
        -onboot 1
        -password "$CT_PASSWORD"
    )

    if "${pct_cmd[@]}" &>/dev/null; then
        emit "ok" "Container ${CT_ID} created"
    else
        # Retry once -- could be race condition on ID
        if pct status "$CT_ID" &>/dev/null; then
            emit "error" "Container ID ${CT_ID} was claimed by another process"
            CT_ID=$((CT_ID + 1))
            pct_cmd[2]="$CT_ID"
            if "${pct_cmd[@]}" &>/dev/null; then
                emit "ok" "Container ${CT_ID} created (reassigned ID)"
            else
                emit "error" "Container creation failed"
                emit_done_fail "pct create failed"
            fi
        else
            emit "error" "Container creation failed"
            emit_done_fail "pct create failed"
        fi
    fi
}

# ---------------------------------------------------------------------------
# 10. Start container
# ---------------------------------------------------------------------------
start_container() {
    emit "install" "Starting container ${CT_ID}"

    if pct start "$CT_ID" &>/dev/null; then
        emit "ok" "Container ${CT_ID} started"
    else
        cleanup_on_fail "$CT_ID" "Failed to start container ${CT_ID}"
    fi
}

# ---------------------------------------------------------------------------
# 11. Setup autologin for Proxmox console
# ---------------------------------------------------------------------------
setup_autologin() {
    emit "check" "Configuring console autologin"

    # Wait a moment for systemd to initialize inside the container
    sleep 2

    pct exec "$CT_ID" -- bash -c '
        mkdir -p /etc/systemd/system/container-getty@1.service.d
        cat > /etc/systemd/system/container-getty@1.service.d/override.conf <<AUTOLOGIN
[Service]
ExecStart=
ExecStart=-/sbin/agetty --autologin root --noclear --keep-baud tty%I 115200,38400,9600 \$TERM
AUTOLOGIN
        systemctl daemon-reload
        systemctl restart container-getty@1.service
    ' &>/dev/null

    if [[ $? -eq 0 ]]; then
        emit "ok" "Console autologin enabled"
    else
        # Non-fatal -- container works fine without it
        emit "ok" "Console autologin skipped (non-critical)"
    fi
}

# ---------------------------------------------------------------------------
# 12. Select fastest apt mirror and update
# ---------------------------------------------------------------------------
setup_apt_mirror() {
    emit "check" "Selecting fastest apt mirror"

    # Wait for network inside container first
    local net_ready=false
    for (( i = 1; i <= 15; i++ )); do
        if pct exec "$CT_ID" -- ping -c 1 -W 2 archive.ubuntu.com &>/dev/null; then
            net_ready=true
            break
        fi
        sleep 1
    done

    if [[ "$net_ready" == false ]]; then
        emit "ok" "Network not ready, skipping mirror selection"
        return
    fi

    # Ping mirrors in parallel, pick fastest
    local best_mirror="archive.ubuntu.com"
    local best_time=9999

    local mirrors=(
        "archive.ubuntu.com"
        "mirror.yandex.ru"
        "de.archive.ubuntu.com"
        "nl.archive.ubuntu.com"
        "us.archive.ubuntu.com"
        "mirror.linux-ia64.org"
    )

    local tmpdir
    tmpdir=$(pct exec "$CT_ID" -- mktemp -d 2>/dev/null || echo "/tmp/mirror-test")

    # Launch all pings in parallel inside the container
    pct exec "$CT_ID" -- bash -c "
        mkdir -p ${tmpdir}
        for m in ${mirrors[*]}; do
            (ping -c 1 -W 2 \$m 2>/dev/null | grep -oP 'time=\K[0-9.]+' > ${tmpdir}/\$m || echo 9999 > ${tmpdir}/\$m) &
        done
        wait
    " &>/dev/null

    # Read results
    for m in "${mirrors[@]}"; do
        local ms
        ms=$(pct exec "$CT_ID" -- cat "${tmpdir}/${m}" 2>/dev/null | head -1)
        ms="${ms:-9999}"

        # Compare as integers (strip decimal)
        local ms_int="${ms%%.*}"
        ms_int="${ms_int:-9999}"

        if [[ "$ms_int" -lt "$best_time" ]]; then
            best_time="$ms_int"
            best_mirror="$m"
        fi
    done

    # Cleanup
    pct exec "$CT_ID" -- rm -rf "$tmpdir" &>/dev/null

    emit "ok" "Fastest mirror: ${best_mirror} (${best_time}ms)" "{\"mirror\":\"${best_mirror}\",\"latency_ms\":${best_time}}"

    # Apply mirror if different from default
    if [[ "$best_mirror" != "archive.ubuntu.com" ]]; then
        emit "install" "Configuring apt mirror: ${best_mirror}"
        pct exec "$CT_ID" -- bash -c "
            sed -i 's|http://archive.ubuntu.com|http://${best_mirror}|g' /etc/apt/sources.list
        " &>/dev/null
        emit "ok" "Apt mirror set to ${best_mirror}"
    fi

    # Run apt update
    emit "install" "Updating package lists"
    if pct exec "$CT_ID" -- bash -c "apt-get update -qq" &>/dev/null; then
        emit "ok" "Package lists updated"
    else
        emit "ok" "Package lists update had warnings (non-critical)"
    fi
}

# ---------------------------------------------------------------------------
# 13. Wait for network and get IP
# ---------------------------------------------------------------------------
wait_for_network() {
    emit "check" "Waiting for network"

    local ip=""
    local retries=30

    for (( i = 1; i <= retries; i++ )); do
        ip=$(pct exec "$CT_ID" -- ip -4 -o addr show dev eth0 2>/dev/null \
            | awk '{print $4}' \
            | cut -d/ -f1 \
            | head -1)

        if [[ -n "$ip" && "$ip" != "127.0.0.1" ]]; then
            emit "ok" "Container IP: ${ip}" "{\"ip\":\"${ip}\"}"
            CT_ACTUAL_IP="$ip"
            return
        fi
        sleep 1
    done

    # Fallback: no IP but container is running
    CT_ACTUAL_IP="unknown"
    emit "ok" "Container running but IP not detected (check network manually)"
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
main() {
    # 1. Verify we're on Proxmox
    check_proxmox

    # 2. Container ID
    resolve_ct_id

    # 3. Storage
    resolve_storage

    # 4. Free space
    check_free_space

    # 5. Network bridge
    resolve_bridge

    # 6. Password
    resolve_password

    # 7. Template
    download_template

    # 8. subuid/subgid
    fix_subuid_subgid

    # 9. Create
    create_container

    # 10. Start
    start_container

    # 11. Autologin
    setup_autologin

    # 12. Apt mirror + update
    setup_apt_mirror

    # 13. Network
    wait_for_network

    # 14. Done
    emit_done_ok "{\"id\":\"${CT_ID}\",\"hostname\":\"${CT_HOSTNAME}\",\"ip\":\"${CT_ACTUAL_IP}\",\"password\":\"${CT_PASSWORD}\",\"memory\":\"${CT_MEMORY}\",\"swap\":\"${CT_SWAP}\",\"disk\":\"${CT_DISK}\",\"cores\":\"${CT_CORES}\",\"storage\":\"${CT_STORAGE}\",\"bridge\":\"${CT_BRIDGE}\",\"template\":\"${TEMPLATE}\"}"
}

main

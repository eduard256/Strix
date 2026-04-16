#!/usr/bin/env bash
# =============================================================================
# Strix -- install.sh (navigator / frontend)
#
# Main entry point for Strix installation.
# Shows animated owl + STRIX logo while running background checks.
# Detects system type (Proxmox / Linux / macOS), Docker, Frigate, go2rtc.
# Then guides the user through installation by calling worker scripts.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/eduard256/Strix/main/scripts/install.sh | bash
#   bash scripts/install.sh
# =============================================================================

# ---------------------------------------------------------------------------
# Owl definitions (5 lines each: line1, line2, line3, line4, name)
# ---------------------------------------------------------------------------
OWL_COUNT=5

# Wide-eyed owl
OWL_0_1="   ___"
OWL_0_2="  <O,O>"
OWL_0_3="  [\`-']"
OWL_0_4="  -\"-\"-"
OWL_0_NAME="wide-eyed owl"

# Happy owl
OWL_1_1="   ___"
OWL_1_2="  <^,^>"
OWL_1_3="  [\`-']"
OWL_1_4="  -\"-\"-"
OWL_1_NAME="happy owl"

# Winking owl
OWL_2_1="   ___"
OWL_2_2="  <*,->"
OWL_2_3="  [\`-']"
OWL_2_4="  -\"-\"-"
OWL_2_NAME="winking owl"

# Flying owl
OWL_3_1="   ___"
OWL_3_2="  <*,*>"
OWL_3_3="  =^\`-'^="
OWL_3_4="    \" \""
OWL_3_NAME="flying owl"

# Super owl
OWL_4_1="   ___"
OWL_4_2="  <*,*>"
OWL_4_3="  [\`S']"
OWL_4_4="  -\"-\"-"
OWL_4_NAME="super owl"

# ---------------------------------------------------------------------------
# Terminal helpers
# ---------------------------------------------------------------------------
term_width() {
    local w
    w=$(tput cols 2>/dev/null || echo 80)
    [[ "$w" -lt 20 ]] && w=80
    echo "$w"
}

term_height() {
    local h
    h=$(tput lines 2>/dev/null || echo 24)
    [[ "$h" -lt 10 ]] && h=24
    echo "$h"
}

# Print text centered horizontally at a specific row
# Usage: print_at_center ROW "text" [color_code]
print_at_center() {
    local row="$1"
    local text="$2"
    local color="${3:-}"
    local reset="\033[0m"

    local w
    w=$(term_width)

    # Strip ANSI for length calc
    local stripped
    stripped=$(echo -e "$text" | sed 's/\x1b\[[0-9;]*m//g')
    local len=${#stripped}
    local col=$(( (w - len) / 2 ))
    [[ "$col" -lt 0 ]] && col=0

    tput cup "$row" "$col" 2>/dev/null
    if [[ -n "$color" ]]; then
        echo -ne "${color}${text}${reset}"
    else
        echo -ne "${text}"
    fi
}

# ---------------------------------------------------------------------------
# Show a single owl centered on screen
# Usage: show_owl INDEX [brightness]
#   brightness: "bright" "dim" "verydim" "hidden"
# ---------------------------------------------------------------------------
show_owl() {
    local idx="$1"
    local brightness="${2:-bright}"

    local color=""
    case "$brightness" in
        bright)  color="\033[97m" ;;     # bright white
        dim)     color="\033[37m" ;;     # normal white
        verydim) color="\033[90m" ;;     # dark gray
        hidden)  color="\033[30m" ;;     # black (invisible)
    esac

    local name_color=""
    case "$brightness" in
        bright)  name_color="\033[36m" ;;   # cyan
        dim)     name_color="\033[2;36m" ;; # dim cyan
        verydim) name_color="\033[90m" ;;   # dark gray
        hidden)  name_color="\033[30m" ;;   # black
    esac

    # Get owl lines by index
    local l1 l2 l3 l4 name
    eval "l1=\"\$OWL_${idx}_1\""
    eval "l2=\"\$OWL_${idx}_2\""
    eval "l3=\"\$OWL_${idx}_3\""
    eval "l4=\"\$OWL_${idx}_4\""
    eval "name=\"\$OWL_${idx}_NAME\""

    # Position: top area of screen
    local start_row=2

    print_at_center "$start_row"       "$l1" "$color"
    print_at_center "$((start_row+1))" "$l2" "$color"
    print_at_center "$((start_row+2))" "$l3" "$color"
    print_at_center "$((start_row+3))" "$l4" "$color"
}

# ---------------------------------------------------------------------------
# Clear owl area (rows 2-6)
# ---------------------------------------------------------------------------
clear_owl_area() {
    local w
    w=$(term_width)
    local blank
    blank=$(printf "%*s" "$w" "")

    for row in 2 3 4 5 6; do
        tput cup "$row" 0 2>/dev/null
        echo -ne "$blank"
    done
}

# ---------------------------------------------------------------------------
# Transition: fade out current owl, fade in next
# ---------------------------------------------------------------------------
transition_owl() {
    local from_idx="$1"
    local to_idx="$2"

    # Fade out: bright -> dim -> verydim -> hidden
    show_owl "$from_idx" "dim"
    sleep 0.1
    show_owl "$from_idx" "verydim"
    sleep 0.1
    clear_owl_area

    # Fade in: hidden -> verydim -> dim -> bright
    show_owl "$to_idx" "verydim"
    sleep 0.1
    show_owl "$to_idx" "dim"
    sleep 0.1
    show_owl "$to_idx" "bright"
}

# ---------------------------------------------------------------------------
# Cycle through all owls with animation
# Usage: cycle_owls [cycles] [delay_between_seconds]
#   cycles=0 means infinite
# ---------------------------------------------------------------------------
cycle_owls() {
    local cycles="${1:-3}"
    local delay="${2:-2}"
    local infinite=false
    [[ "$cycles" -eq 0 ]] && infinite=true

    local current=0
    local i=0

    while [[ "$infinite" == true ]] || [[ "$i" -lt $((cycles * OWL_COUNT)) ]]; do
        [[ "${_OWL_RUNNING:-true}" == false ]] && return

        local next=$(( (current + 1) % OWL_COUNT ))

        if [[ "$i" -eq 0 ]]; then
            show_owl "$current" "verydim"
            sleep 0.1 || return
            show_owl "$current" "dim"
            sleep 0.1 || return
            show_owl "$current" "bright"
        else
            transition_owl "$current" "$next"
            current=$next
        fi

        sleep "$delay" || return
        i=$((i + 1))
    done
}

# ---------------------------------------------------------------------------
# Setup screen
# ---------------------------------------------------------------------------
setup_screen() {
    tput civis 2>/dev/null
    clear
}

# ---------------------------------------------------------------------------
# Restore screen
# ---------------------------------------------------------------------------
restore_screen() {
    tput cnorm 2>/dev/null
    echo -ne "\033[0m"
    clear
}

# ---------------------------------------------------------------------------
# STRIX block title (static, drawn once below owl area)
# ---------------------------------------------------------------------------
show_title() {
    local c="\033[35;1m"  # bold magenta
    local tr=7
    print_at_center "$((tr))"   '███████╗████████╗██████╗ ██╗██╗  ██╗' "$c"
    print_at_center "$((tr+1))" '██╔════╝╚══██╔══╝██╔══██╗██║╚██╗██╔╝' "$c"
    print_at_center "$((tr+2))" '███████╗   ██║   ██████╔╝██║ ╚███╔╝'  "$c"
    print_at_center "$((tr+3))" '╚════██║   ██║   ██╔══██╗██║ ██╔██╗'  "$c"
    print_at_center "$((tr+4))" '███████║   ██║   ██║  ██║██║██╔╝ ██╗' "$c"
    print_at_center "$((tr+5))" '╚══════╝   ╚═╝   ╚═╝  ╚═╝╚═╝╚═╝  ╚═╝' "$c"
}

# ---------------------------------------------------------------------------
# Detection: download and run detect.sh in background
# ---------------------------------------------------------------------------
DETECT_FILE="/tmp/strix-detect-$$.sh"
DETECT_RESULT="/tmp/strix-detect-$$.out"
DETECT_BASE="https://raw.githubusercontent.com/eduard256/Strix/main/scripts"

run_detection() {
    # Download detect.sh
    local dl_ok=false
    if command -v curl &>/dev/null; then
        curl -fsSL "${DETECT_BASE}/detect.sh" -o "$DETECT_FILE" 2>/dev/null && dl_ok=true
    elif command -v wget &>/dev/null; then
        wget -qO "$DETECT_FILE" "${DETECT_BASE}/detect.sh" 2>/dev/null && dl_ok=true
    fi

    if [[ "$dl_ok" == false ]] || [[ ! -f "$DETECT_FILE" ]]; then
        echo '{"type":"error","msg":"Failed to download detect.sh"}' > "$DETECT_RESULT"
        echo '{"type":"done","ok":false}' >> "$DETECT_RESULT"
        return 1
    fi

    # Run detect.sh, redirect stdout to file, stderr to /dev/null
    bash "$DETECT_FILE" 1>"$DETECT_RESULT" 2>/dev/null
}

# ---------------------------------------------------------------------------
# Status table: parse detect.sh JSON output and draw below STRIX title
# ---------------------------------------------------------------------------
STATUS_ROW=14

draw_status_line() {
    local row="$1"
    local label="$2"
    local value="$3"
    local status="${4:-}"

    local lbl_color="\033[37m"
    local val_color="\033[97m"
    local dot_color="\033[90m"

    case "$status" in
        ok)      val_color="\033[32m" ;;
        miss)    val_color="\033[90m" ;;
        loading) val_color="\033[33m" ;;
    esac

    local w
    w=$(term_width)
    local box_w=44
    local col=$(( (w - box_w) / 2 ))
    [[ "$col" -lt 0 ]] && col=0

    tput cup "$row" 0 2>/dev/null
    printf "%*s" "$w" ""

    tput cup "$row" "$col" 2>/dev/null

    local dots_len=$(( box_w - ${#label} - ${#value} - 4 ))
    [[ "$dots_len" -lt 1 ]] && dots_len=1
    local dots=""
    local d
    for (( d = 0; d < dots_len; d++ )); do dots="${dots}."; done

    echo -ne "${lbl_color}  ${label} ${dot_color}${dots} ${val_color}${value}\033[0m"
}

draw_loading_status() {
    draw_status_line "$STATUS_ROW"       "System"  "detecting..." "loading"
    draw_status_line "$((STATUS_ROW+1))" "Docker"  "..." "loading"
    draw_status_line "$((STATUS_ROW+2))" "Compose" "..." "loading"
    draw_status_line "$((STATUS_ROW+3))" "Frigate" "..." "loading"
    draw_status_line "$((STATUS_ROW+4))" "go2rtc"  "..." "loading"
}

# Parse a JSON field from a line: extract "msg" value (first match only)
json_msg() {
    echo "$1" | grep -oP '"msg"\s*:\s*"\K[^"]+' | head -1 || echo ""
}

# Parse a JSON field: extract "type" value (first match only)
json_type() {
    echo "$1" | grep -oP '"type"\s*:\s*"\K[^"]+' | head -1 || echo ""
}

draw_detect_results() {
    [[ -f "$DETECT_RESULT" ]] || return 1
    grep -q '"type":"done"' "$DETECT_RESULT" 2>/dev/null || return 1

    # Parse each "check" section: the line after "check" is either "ok" or "miss"
    local section=""
    local sys_msg="unknown" sys_status="miss"
    local docker_msg="not installed" docker_status="miss"
    local compose_msg="not installed" compose_status="miss"
    local frigate_msg="not found" frigate_status="miss"
    local go2rtc_msg="not found" go2rtc_status="miss"

    while IFS= read -r line; do
        local t m
        t=$(json_type "$line")
        m=$(json_msg "$line")

        if [[ "$t" == "check" ]]; then
            case "$m" in
                *system*)    section="system" ;;
                *Docker\ C*|*Compose*) section="compose" ;;
                *Docker*)    section="docker" ;;
                *Frigate*)   section="frigate" ;;
                *go2rtc*)    section="go2rtc" ;;
            esac
            continue
        fi

        case "$section" in
            system)
                [[ "$t" == "ok" ]]   && { sys_msg="$m"; sys_status="ok"; }
                [[ "$t" == "miss" ]] && { sys_msg="unknown"; sys_status="miss"; }
                section="" ;;
            docker)
                [[ "$t" == "ok" ]]   && { docker_msg="$m"; docker_status="ok"; }
                [[ "$t" == "miss" ]] && { docker_msg="not installed"; docker_status="miss"; }
                section="" ;;
            compose)
                [[ "$t" == "ok" ]]   && { compose_msg="$m"; compose_status="ok"; }
                [[ "$t" == "miss" ]] && { compose_msg="not installed"; compose_status="miss"; }
                section="" ;;
            frigate)
                [[ "$t" == "ok" ]]   && { frigate_msg="$m"; frigate_status="ok"; }
                [[ "$t" == "miss" ]] && { frigate_msg="not found"; frigate_status="miss"; }
                section="" ;;
            go2rtc)
                [[ "$t" == "ok" ]]   && { go2rtc_msg="$m"; go2rtc_status="ok"; }
                [[ "$t" == "miss" ]] && { go2rtc_msg="not found"; go2rtc_status="miss"; }
                section="" ;;
        esac
    done < "$DETECT_RESULT"

    draw_status_line "$STATUS_ROW"       "System"  "$sys_msg"     "$sys_status"
    draw_status_line "$((STATUS_ROW+1))" "Docker"  "$docker_msg"  "$docker_status"
    draw_status_line "$((STATUS_ROW+2))" "Compose" "$compose_msg" "$compose_status"
    draw_status_line "$((STATUS_ROW+3))" "Frigate" "$frigate_msg" "$frigate_status"
    draw_status_line "$((STATUS_ROW+4))" "go2rtc"  "$go2rtc_msg"  "$go2rtc_status"

    return 0
}

# ---------------------------------------------------------------------------
# Main loop: owl animation + status table refresh
# ---------------------------------------------------------------------------
main_loop() {
    local current=0
    local i=0

    while [[ "${_OWL_RUNNING:-true}" == true ]]; do
        local next=$(( (current + 1) % OWL_COUNT ))

        if [[ "$i" -eq 0 ]]; then
            show_owl "$current" "verydim"
            sleep 0.1 || return
            show_owl "$current" "dim"
            sleep 0.1 || return
            show_owl "$current" "bright"
        else
            transition_owl "$current" "$next"
            current=$next
        fi

        sleep 2 || return
        i=$((i + 1))
    done
}

# ---------------------------------------------------------------------------
# Launch navigator based on detected system
# ---------------------------------------------------------------------------
launch_navigator() {
    if [[ ! -f "$DETECT_RESULT" ]]; then
        main_loop
        restore_screen
        return
    fi

    local sys_type
    sys_type=$(grep -oP '"type"\s*:\s*"\K(proxmox|linux|macos)' "$DETECT_RESULT" | head -1)

    case "$sys_type" in
        proxmox)
            sleep 3
            restore_screen

            local proxmox_script="/tmp/strix-proxmox-$$.sh"
            curl -fsSL "${DETECT_BASE}/proxmox.sh" -o "$proxmox_script" 2>/dev/null
            if [[ -f "$proxmox_script" ]]; then
                bash "$proxmox_script"
                rm -f "$proxmox_script"
            fi
            ;;
        linux)
            sleep 3
            restore_screen

            local linux_script="/tmp/strix-linux-$$.sh"
            curl -fsSL "${DETECT_BASE}/linux.sh" -o "$linux_script" 2>/dev/null
            if [[ -f "$linux_script" ]]; then
                bash "$linux_script"
                rm -f "$linux_script"
            fi
            ;;
        macos)
            print_at_center "$((STATUS_ROW + 6))" "macOS installer coming soon" "\033[33m"
            main_loop
            restore_screen
            ;;
        *)
            main_loop
            restore_screen
            ;;
    esac
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    _OWL_RUNNING=true

    _owl_cleanup() {
        _OWL_RUNNING=false
        rm -f "$DETECT_FILE" "$DETECT_RESULT" 2>/dev/null
        restore_screen
        exit 0
    }

    trap _owl_cleanup INT TERM
    trap 'rm -f "$DETECT_FILE" "$DETECT_RESULT" 2>/dev/null; restore_screen' EXIT

    setup_screen
    show_title

    # Show loading state while detecting
    draw_loading_status

    # Run detection (synchronous, ~6-8 sec on slow networks)
    run_detection

    # Redraw screen in case detect leaked output
    clear
    show_title

    # Draw real results
    draw_detect_results || true

    # Check system type and launch appropriate navigator
    launch_navigator

    # Cleanup
    rm -f "$DETECT_FILE" "$DETECT_RESULT" 2>/dev/null
fi

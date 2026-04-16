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
# Main
# ---------------------------------------------------------------------------
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    _OWL_RUNNING=true

    _owl_cleanup() {
        _OWL_RUNNING=false
        restore_screen
        exit 0
    }

    trap _owl_cleanup INT TERM
    trap restore_screen EXIT

    setup_screen
    show_title
    cycle_owls 0 2
    restore_screen
fi

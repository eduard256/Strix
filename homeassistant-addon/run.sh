#!/usr/bin/with-contenv bashio

# Get configuration from Home Assistant
LOG_LEVEL=$(bashio::config 'log_level')
PORT=$(bashio::config 'port')
STRICT_VALIDATION=$(bashio::config 'strict_validation')

# Print banner
bashio::log.info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
bashio::log.info "  ____  _        _      "
bashio::log.info " / ___|| |_ _ __(_)_  __"
bashio::log.info " \___ \| __| '__| \ \/ /"
bashio::log.info "  ___) | |_| |  | |>  < "
bashio::log.info " |____/ \__|_|  |_/_/\_\\"
bashio::log.info ""
bashio::log.info " Smart IP Camera Stream Discovery System"
bashio::log.info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Set environment variables
export STRIX_LOG_LEVEL="${LOG_LEVEL}"
export STRIX_LOG_FORMAT="json"
export STRIX_API_LISTEN=":${PORT}"
export STRIX_DATA_PATH="/app/data"

bashio::log.info "Starting Strix with the following configuration:"
bashio::log.info " - Log Level: ${LOG_LEVEL}"
bashio::log.info " - Port: ${PORT}"
bashio::log.info " - Strict Validation: ${STRICT_VALIDATION}"
bashio::log.info " - Data Path: ${STRIX_DATA_PATH}"

# Check if ffprobe is available
if command -v ffprobe &> /dev/null; then
    bashio::log.info "FFProbe found: $(ffprobe -version | head -n1)"
else
    bashio::log.warning "FFProbe not found, stream validation will be limited"
fi

# Start Strix
bashio::log.info "Starting Strix server..."
exec /app/strix

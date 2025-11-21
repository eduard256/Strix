#!/bin/bash
# Simple development server for Strix WebUI
# This allows you to test the UI without running the Go backend

PORT=${1:-8080}

echo "Starting development server on port $PORT"
echo "Open: http://localhost:$PORT?mock=true"
echo ""
echo "Press Ctrl+C to stop"

# Use Python's built-in HTTP server
cd "$(dirname "$0")"
python3 -m http.server $PORT

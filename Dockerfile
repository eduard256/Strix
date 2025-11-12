# Strix - Smart IP Camera Stream Discovery System
# Multi-stage Dockerfile for minimal image size

# Stage 1: Builder
FROM golang:1.24-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build \
    -ldflags="-s -w -X main.Version=docker" \
    -o strix \
    cmd/strix/main.go

# Stage 2: Runtime
FROM alpine:latest

# Install runtime dependencies
# - ffmpeg/ffprobe: Required for RTSP stream validation
# - ca-certificates: Required for HTTPS requests to cameras
# - tzdata: Required for correct timestamps
# - wget: Required for healthcheck
RUN apk add --no-cache \
    ffmpeg \
    ca-certificates \
    tzdata \
    wget \
    && rm -rf /var/cache/apk/*

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/strix .

# Copy camera database (CRITICAL - app won't work without it)
COPY --from=builder /build/data ./data

# Create directory for optional config
RUN mkdir -p /app/config

# Create non-root user for security
RUN addgroup -g 1000 strix && \
    adduser -D -u 1000 -G strix strix && \
    chown -R strix:strix /app

# Switch to non-root user
USER strix

# Expose default port
EXPOSE 4567

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:4567/api/v1/health || exit 1

# Start application
CMD ["./strix"]

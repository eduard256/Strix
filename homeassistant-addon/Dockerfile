ARG BUILD_FROM
FROM ${BUILD_FROM}

# Install runtime dependencies
RUN apk add --no-cache \
    ffmpeg \
    ca-certificates \
    tzdata \
    wget \
    && rm -rf /var/cache/apk/*

# Set working directory
WORKDIR /app

# Copy binary from build context
COPY strix /app/strix

# Copy camera database (CRITICAL)
COPY data /app/data

# Copy WebUI files
COPY webui /app/webui

# Copy run script
COPY run.sh /
RUN chmod a+x /run.sh

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
CMD ["/run.sh"]

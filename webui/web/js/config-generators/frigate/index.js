/**
 * Frigate NVR Configuration Generator
 * Generates unified Frigate + Go2RTC YAML configs
 * All cameras are routed through Frigate's built-in go2rtc for optimal performance
 */
export class FrigateGenerator {
    /**
     * Generate complete Frigate config with embedded Go2RTC
     * @param {Object} mainStream - Main stream object (used for recording)
     * @param {Object} subStream - Optional sub stream object (used for detection if provided)
     * @returns {string} YAML configuration string
     */
    static generate(mainStream, subStream = null) {
        const cameraName = this.generateCameraName(mainStream);
        const config = [];

        // MQTT Configuration
        config.push('mqtt:');
        config.push('  enabled: false');
        config.push('');

        // Global Record Configuration
        config.push('# Global Recording Settings');
        config.push('record:');
        config.push('  enabled: true');
        config.push('  retain:');
        config.push('    days: 7');
        config.push('    mode: motion  # Record only on motion detection');
        config.push('');

        // Generate Go2RTC section
        config.push('# Go2RTC Configuration (Frigate built-in)');
        config.push('go2rtc:');
        config.push('  streams:');

        // Main stream configuration
        const mainStreamName = this.generateStreamName(mainStream, 'main');
        const mainSource = this.generateGo2RTCSource(mainStream);
        config.push(`    '${mainStreamName}':`);
        config.push(`      - ${mainSource}`);

        // Sub stream configuration if provided
        if (subStream) {
            config.push('');
            const subStreamName = this.generateStreamName(subStream, 'sub');
            const subSource = this.generateGo2RTCSource(subStream);
            config.push(`    '${subStreamName}':`);
            config.push(`      - ${subSource}`);
        }

        config.push('');

        // Generate Frigate cameras section
        config.push('# Frigate Camera Configuration');
        config.push('cameras:');
        config.push(`  ${cameraName}:`);
        config.push('    ffmpeg:');
        config.push('      inputs:');

        if (subStream) {
            // If sub stream exists: use it for detection, main for recording
            const subStreamName = this.generateStreamName(subStream, 'sub');
            config.push(`        - path: rtsp://127.0.0.1:8554/${subStreamName}`);
            config.push('          input_args: preset-rtsp-restream');
            config.push('          roles:');
            config.push('            - detect');
            config.push(`        - path: rtsp://127.0.0.1:8554/${mainStreamName}`);
            config.push('          input_args: preset-rtsp-restream');
            config.push('          roles:');
            config.push('            - record');
        } else {
            // No sub stream: use main for both detection and recording
            config.push(`        - path: rtsp://127.0.0.1:8554/${mainStreamName}`);
            config.push('          input_args: preset-rtsp-restream');
            config.push('          roles:');
            config.push('            - detect');
            config.push('            - record');
        }

        // Live view configuration
        if (subStream) {
            config.push('    live:');
            config.push('      streams:');
            config.push(`        Main Stream: ${mainStreamName}    # HD для просмотра`);
            config.push(`        Sub Stream: ${this.generateStreamName(subStream, 'sub')}      # Низкое разрешение (опционально)`);
        }

        // Object detection configuration
        config.push('    objects:');
        config.push('      track:');
        config.push('        - person');
        config.push('        - car');
        config.push('        - cat');
        config.push('        - dog');

        // Recording configuration
        config.push('    record:');
        config.push('      enabled: true');

        config.push('');
        config.push('version: 0.16-0');

        return config.join('\n');
    }

    /**
     * Generate Go2RTC source configuration based on stream type
     * Returns the source string for go2rtc streams section
     */
    static generateGo2RTCSource(stream) {
        // Handle JPEG snapshots with exec:ffmpeg conversion
        // Uses full path to ffmpeg and {{output}} for Frigate template escaping
        if (stream.type === 'JPEG') {
            return [
                'exec:/usr/lib/ffmpeg/7.0/bin/ffmpeg',
                '-loglevel quiet',
                '-f image2',
                '-loop 1',
                '-framerate 10',
                `-i ${stream.url}`,
                '-c:v libx264',
                '-preset ultrafast',
                '-tune zerolatency',
                '-g 20',
                '-f rtsp {{output}}'  // Double braces for Frigate template escaping
            ].join(' ');
        }

        // Handle ONVIF - convert to onvif:// format if needed
        if (stream.type === 'ONVIF') {
            try {
                const urlObj = new URL(stream.url);
                // Extract credentials and host from HTTP URL
                const username = urlObj.username || 'admin';
                const password = urlObj.password || '';
                const host = urlObj.hostname;
                const port = urlObj.port || '80';

                // Generate onvif:// URL
                return `onvif://${username}:${password}@${host}:${port}`;
            } catch (e) {
                // If URL parsing fails, return as-is
                return stream.url;
            }
        }

        // Handle BUBBLE protocol - convert to bubble:// format for go2rtc
        if (stream.type === 'BUBBLE') {
            try {
                const urlObj = new URL(stream.url);
                const username = urlObj.username || 'admin';
                const password = urlObj.password || '';
                const host = urlObj.hostname;
                const port = urlObj.port || '80';
                const path = urlObj.pathname + urlObj.search;
                return `bubble://${username}:${password}@${host}:${port}${path}#video=copy`;
            } catch (e) {
                return stream.url;
            }
        }

        // For all other types (RTSP, MJPEG, HLS, HTTP-FLV, RTMP, etc.): use direct URL
        // Go2RTC handles these formats natively
        return stream.url;
    }

    /**
     * Generate camera name from IP address
     * Format: "camera_192_168_1_100"
     */
    static generateCameraName(stream) {
        try {
            const urlObj = new URL(stream.url);
            const ip = urlObj.hostname.replace(/\./g, '_').replace(/:/g, '_');
            return `camera_${ip}`;
        } catch (e) {
            return 'camera';
        }
    }

    /**
     * Generate stream name for Go2RTC reference
     * Format: "192_168_1_100_main" or "192_168_1_100_sub"
     */
    static generateStreamName(stream, suffix) {
        try {
            const urlObj = new URL(stream.url);
            const ip = urlObj.hostname.replace(/\./g, '_').replace(/:/g, '_');
            return `${ip}_${suffix}`;
        } catch (e) {
            return `camera_stream_${suffix}`;
        }
    }
}

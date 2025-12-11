/**
 * Frigate NVR Configuration Generator
 * Adds cameras to existing Frigate configuration
 * Based on frigate-conf-generate logic
 */
export class FrigateGenerator {
    /**
     * Main entry point - generates config (new or adds to existing)
     * @param {string} existingConfig - Existing YAML config text (or empty string)
     * @param {Object} mainStream - Main stream object
     * @param {Object} subStream - Optional sub stream object
     * @returns {string} YAML configuration string
     */
    static generate(existingConfig, mainStream, subStream = null) {
        if (!existingConfig || existingConfig.trim() === '') {
            // Create new config from scratch
            return this.createNewConfig(mainStream, subStream);
        }

        // Add to existing config
        return this.addToExistingConfig(existingConfig, mainStream, subStream);
    }

    /**
     * Add camera to existing config (text-based, preserves everything)
     */
    static addToExistingConfig(existingConfig, mainStream, subStream) {
        const lines = existingConfig.split('\n');

        // Find existing camera names and stream names to avoid duplicates
        const existingCameras = this.findExistingCameras(lines);
        const existingStreams = this.findExistingStreams(lines);

        // Generate unique camera info
        const cameraInfo = this.generateUniqueCameraInfo(mainStream, subStream, existingCameras, existingStreams);

        // Find insertion points
        const go2rtcStreamIndex = this.findGo2rtcStreamsInsertionPoint(lines);
        const camerasInsertIndex = this.findCamerasInsertionPoint(lines);

        if (go2rtcStreamIndex === -1 || camerasInsertIndex === -1) {
            throw new Error('Could not find go2rtc streams or cameras section in config');
        }

        // Generate new stream lines
        const streamLines = this.generateStreamLines(cameraInfo);

        // Generate new camera lines
        const cameraLines = this.generateCameraLines(cameraInfo);

        // Insert streams into go2rtc section
        lines.splice(go2rtcStreamIndex, 0, ...streamLines);

        // Insert camera into cameras section (adjust index after first insertion)
        const adjustedCameraIndex = camerasInsertIndex + streamLines.length;
        lines.splice(adjustedCameraIndex, 0, ...cameraLines);

        return lines.join('\n');
    }

    /**
     * Find existing camera names
     */
    static findExistingCameras(lines) {
        const cameras = new Set();
        let inCamerasSection = false;

        for (const line of lines) {
            if (line.match(/^cameras:/)) {
                inCamerasSection = true;
                continue;
            }

            if (inCamerasSection && line.match(/^[a-z]/)) {
                break; // Next top-level section
            }

            if (inCamerasSection && line.match(/^\s{2}(\w+):/)) {
                const match = line.match(/^\s{2}(\w+):/);
                cameras.add(match[1]);
            }
        }

        return cameras;
    }

    /**
     * Find existing stream names
     */
    static findExistingStreams(lines) {
        const streams = new Set();
        let inStreamsSection = false;

        for (const line of lines) {
            if (line.match(/^\s{2}streams:/)) {
                inStreamsSection = true;
                continue;
            }

            if (inStreamsSection && line.match(/^[a-z]/)) {
                break; // Next top-level section
            }

            if (inStreamsSection && line.match(/^\s{4}'?(\w+)'?:/)) {
                const match = line.match(/^\s{4}'?(\w+)'?:/);
                streams.add(match[1]);
            }
        }

        return streams;
    }

    /**
     * Find where to insert new streams in go2rtc section
     */
    static findGo2rtcStreamsInsertionPoint(lines) {
        let inStreamsSection = false;
        let lastStreamIndex = -1;

        for (let i = 0; i < lines.length; i++) {
            const line = lines[i];

            if (line.match(/^\s{2}streams:/)) {
                inStreamsSection = true;
                continue;
            }

            if (inStreamsSection) {
                // Check if this is a stream definition or its content
                if (line.match(/^\s{4,}/)) {
                    lastStreamIndex = i;
                } else if (line.match(/^[a-z#]/)) {
                    // Found next section - insert before empty line if it exists
                    if (lastStreamIndex >= 0 && lines[lastStreamIndex + 1]?.trim() === '') {
                        return lastStreamIndex + 2; // After existing empty line
                    }
                    return lastStreamIndex + 1;
                }
            }
        }

        return lastStreamIndex + 1;
    }

    /**
     * Find where to insert new camera in cameras section
     */
    static findCamerasInsertionPoint(lines) {
        let inCamerasSection = false;
        let lastCameraLineIndex = -1;

        for (let i = 0; i < lines.length; i++) {
            const line = lines[i];

            if (line.match(/^cameras:/)) {
                inCamerasSection = true;
                continue;
            }

            if (inCamerasSection) {
                // Check if we're still in a camera definition
                if (line.match(/^\s{2}\w+:/)) {
                    // New camera starting
                    lastCameraLineIndex = i;
                } else if (line.match(/^\s{2,}\S/)) {
                    // Still inside camera definition
                    lastCameraLineIndex = i;
                } else if (line.match(/^[a-z]/) && !line.match(/^cameras:/)) {
                    // Found next top-level section
                    // Skip any empty lines before it
                    let insertIndex = lastCameraLineIndex + 1;
                    while (insertIndex < lines.length && lines[insertIndex].trim() === '') {
                        insertIndex++;
                    }
                    return insertIndex;
                } else if (line.match(/^version:/)) {
                    // Insert before version, skip empty lines
                    let insertIndex = i;
                    while (insertIndex > 0 && lines[insertIndex - 1].trim() === '') {
                        insertIndex--;
                    }
                    return insertIndex;
                }
            }
        }

        // If we reach end of file, insert at end
        return lines.length;
    }

    /**
     * Generate unique camera info avoiding duplicates
     */
    static generateUniqueCameraInfo(mainStream, subStream, existingCameras, existingStreams) {
        const ip = this.extractIP(mainStream.url);
        const baseName = ip ? `camera_${ip.replace(/\./g, '_').replace(/:/g, '_')}` : 'camera';
        const streamBaseName = ip ? ip.replace(/\./g, '_').replace(/:/g, '_') : 'stream';

        // Find unique camera name
        let cameraName = baseName;
        let suffix = 0;
        while (existingCameras.has(cameraName)) {
            suffix++;
            cameraName = `${baseName}_${suffix}`;
        }

        // Find unique stream names
        let mainStreamName = `${streamBaseName}_main${suffix ? `_${suffix}` : ''}`;
        while (existingStreams.has(mainStreamName)) {
            suffix++;
            mainStreamName = `${streamBaseName}_main_${suffix}`;
        }

        let subStreamName = null;
        if (subStream) {
            subStreamName = `${streamBaseName}_sub${suffix ? `_${suffix}` : ''}`;
            while (existingStreams.has(subStreamName)) {
                suffix++;
                subStreamName = `${streamBaseName}_sub_${suffix}`;
            }
        }

        return {
            cameraName,
            mainStreamName,
            subStreamName,
            mainStream,
            subStream
        };
    }

    /**
     * Generate stream lines for go2rtc section
     */
    static generateStreamLines(cameraInfo) {
        const lines = [];

        // Add main stream
        const mainSource = this.generateGo2RTCSource(cameraInfo.mainStream);
        lines.push(`    '${cameraInfo.mainStreamName}':`);
        lines.push(`      - ${mainSource}`);

        // Add sub stream if provided
        if (cameraInfo.subStream) {
            lines.push('');
            const subSource = this.generateGo2RTCSource(cameraInfo.subStream);
            lines.push(`    '${cameraInfo.subStreamName}':`);
            lines.push(`      - ${subSource}`);
        }

        lines.push('');
        return lines;
    }

    /**
     * Build RTSP path with optional ?mp4 suffix for BUBBLE streams
     */
    static buildRtspPath(streamName, streamType) {
        const basePath = `rtsp://127.0.0.1:8554/${streamName}`;
        // Add ?mp4 parameter only for BUBBLE streams to enable recording in Frigate
        return streamType === 'BUBBLE' ? `${basePath}?mp4` : basePath;
    }

    /**
     * Generate camera lines for cameras section
     */
    static generateCameraLines(cameraInfo) {
        const lines = [];

        lines.push(`  ${cameraInfo.cameraName}:`);
        lines.push('    ffmpeg:');
        lines.push('      inputs:');

        if (cameraInfo.subStream) {
            // Use sub for detect, main for record
            const subPath = this.buildRtspPath(cameraInfo.subStreamName, cameraInfo.subStream.type);
            const mainPath = this.buildRtspPath(cameraInfo.mainStreamName, cameraInfo.mainStream.type);

            lines.push(`        - path: ${subPath}`);
            lines.push('          input_args: preset-rtsp-restream');
            lines.push('          roles:');
            lines.push('            - detect');
            lines.push(`        - path: ${mainPath}`);
            lines.push('          input_args: preset-rtsp-restream');
            lines.push('          roles:');
            lines.push('            - record');

            // Add live view configuration
            lines.push('    live:');
            lines.push('      streams:');
            lines.push(`        Main Stream: ${cameraInfo.mainStreamName}    # HD для просмотра`);
            lines.push(`        Sub Stream: ${cameraInfo.subStreamName}      # Низкое разрешение (опционально)`);
        } else {
            // Use main for both detect and record
            const mainPath = this.buildRtspPath(cameraInfo.mainStreamName, cameraInfo.mainStream.type);

            lines.push(`        - path: ${mainPath}`);
            lines.push('          input_args: preset-rtsp-restream');
            lines.push('          roles:');
            lines.push('            - detect');
            lines.push('            - record');
        }

        // Add objects configuration
        lines.push('    objects:');
        lines.push('      track:');
        lines.push('        - person');
        lines.push('        - car');
        lines.push('        - cat');
        lines.push('        - dog');

        // Add record configuration
        lines.push('    record:');
        lines.push('      enabled: true');
        lines.push('');

        return lines;
    }

    /**
     * Create new configuration from scratch
     */
    static createNewConfig(mainStream, subStream) {
        const cameraInfo = this.generateUniqueCameraInfo(mainStream, subStream, new Set(), new Set());
        const lines = [];

        // MQTT
        lines.push('mqtt:');
        lines.push('  enabled: false');
        lines.push('');

        // Record
        lines.push('# Global Recording Settings');
        lines.push('record:');
        lines.push('  enabled: true');
        lines.push('  retain:');
        lines.push('    days: 7');
        lines.push('    mode: motion  # Record only on motion detection');
        lines.push('');

        // Go2RTC
        lines.push('# Go2RTC Configuration (Frigate built-in)');
        lines.push('go2rtc:');
        lines.push('  streams:');
        lines.push(...this.generateStreamLines(cameraInfo));

        // Cameras
        lines.push('# Frigate Camera Configuration');
        lines.push('cameras:');
        lines.push(...this.generateCameraLines(cameraInfo));

        // Version
        lines.push('version: 0.16-0');

        return lines.join('\n');
    }

    /**
     * Extract IP address from URL
     */
    static extractIP(url) {
        try {
            const urlObj = new URL(url);
            return urlObj.hostname;
        } catch (e) {
            // Try to extract IP with regex
            const match = url.match(/(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})/);
            return match ? match[1] : null;
        }
    }

    /**
     * Generate Go2RTC source configuration based on stream type
     */
    static generateGo2RTCSource(stream) {
        // Handle JPEG snapshots with exec:ffmpeg conversion
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
                '-f rtsp {{output}}'
            ].join(' ');
        }

        // Handle ONVIF - convert to onvif:// format if needed
        if (stream.type === 'ONVIF') {
            try {
                const urlObj = new URL(stream.url);
                const username = urlObj.username || 'admin';
                const password = urlObj.password || '';
                const host = urlObj.hostname;
                const port = urlObj.port || '80';
                return `onvif://${username}:${password}@${host}:${port}`;
            } catch (e) {
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

        // For all other types: use direct URL
        return stream.url;
    }
}

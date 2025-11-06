/**
 * Go2RTC Configuration Generator
 * Generates proper go2rtc YAML configs based on stream type
 * Following go2rtc documentation and best practices
 */
export class Go2RTCGenerator {
    /**
     * Generate go2rtc config for streams (main + optional sub)
     * @param {Object} mainStream - Main stream object with type, protocol, and url
     * @param {Object} subStream - Optional sub stream object
     * @returns {string} YAML configuration string
     */
    static generate(mainStream, subStream = null) {
        const configs = [];
        configs.push('streams:');

        // Generate main stream config
        const mainStreamName = this.generateStreamName(mainStream, 'main');
        const mainSource = this.generateSource(mainStream);
        configs.push(`  '${mainStreamName}':`);
        configs.push(`    - ${mainSource}`);

        // Generate sub stream config if provided
        if (subStream) {
            configs.push('');
            const subStreamName = this.generateStreamName(subStream, 'sub');
            const subSource = this.generateSource(subStream);
            configs.push(`  '${subStreamName}':`);
            configs.push(`    - ${subSource}`);
        }

        return configs.join('\n');
    }

    /**
     * Generate stream name from IP address with suffix
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

    /**
     * Generate source configuration based on stream type
     */
    static generateSource(stream) {
        // Handle JPEG snapshots with special exec:ffmpeg conversion
        if (stream.type === 'JPEG') {
            return this.generateJPEGSource(stream);
        }

        // Handle ONVIF
        if (stream.type === 'ONVIF') {
            return this.generateONVIFSource(stream);
        }

        // For all other types: use direct URL
        return stream.url;
    }

    /**
     * Generate JPEG snapshot conversion using exec:ffmpeg
     * Converts static JPEG to RTSP stream with H264 encoding
     */
    static generateJPEGSource(stream) {
        return [
            'exec:ffmpeg',
            '-loglevel quiet',
            '-f image2',
            '-loop 1',
            '-framerate 10',
            `-i ${stream.url}`,
            '-c:v libx264',
            '-preset ultrafast',
            '-tune zerolatency',
            '-g 20',
            '-f rtsp {output}'
        ].join(' ');
    }

    /**
     * Generate ONVIF source
     * Converts HTTP device service endpoint to onvif:// format
     */
    static generateONVIFSource(stream) {
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
}

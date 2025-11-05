export class FrigateGenerator {
    static generate(stream) {
        // For non-RTSP streams, suggest using Go2RTC
        if (stream.type !== 'FFMPEG' || stream.protocol !== 'rtsp') {
            return `# This stream type requires Go2RTC proxy\n\n` +
                   `# This ${stream.type} stream is not natively supported by Frigate.\n` +
                   `# Please use Go2RTC to convert it to RTSP first.\n\n` +
                   `# Steps:\n` +
                   `# 1. Add this stream to your Go2RTC configuration\n` +
                   `# 2. Use the Go2RTC RTSP endpoint in Frigate\n` +
                   `# 3. Example: rtsp://localhost:8554/camera_stream_0`;
        }

        // Generate RTSP config for Frigate
        const cameraName = this.generateCameraName(stream);
        const config = [];

        config.push(`cameras:`);
        config.push(`  ${cameraName}:`);
        config.push(`    ffmpeg:`);
        config.push(`      inputs:`);
        config.push(`        - path: ${stream.url}`);
        config.push(`          roles:`);
        config.push(`            - detect`);
        config.push(`            - record`);

        if (stream.resolution) {
            config.push(`    detect:`);
            const [width, height] = stream.resolution.split('x').map(Number);
            if (width && height) {
                config.push(`      width: ${width}`);
                config.push(`      height: ${height}`);
            }
        }

        return config.join('\n');
    }

    static generateCameraName(stream) {
        try {
            const urlObj = new URL(stream.url);
            const ip = urlObj.hostname.replace(/\./g, '_').replace(/:/g, '_');
            return `camera_${ip}`;
        } catch (e) {
            return 'camera';
        }
    }
}

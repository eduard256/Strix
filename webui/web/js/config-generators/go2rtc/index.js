export class Go2RTCGenerator {
    static generate(stream) {
        const streamName = this.generateStreamName(stream);

        switch (stream.type) {
            case 'FFMPEG':
                if (stream.protocol === 'rtsp') {
                    return this.generateRTSP(streamName, stream);
                }
                break;
            case 'JPEG':
                return this.generateJPEG(streamName, stream);
            case 'MJPEG':
                return this.generateMJPEG(streamName, stream);
            case 'HTTP_VIDEO':
                return this.generateHTTPVideo(streamName, stream);
            case 'HLS':
                return this.generateHLS(streamName, stream);
            case 'ONVIF':
                return `# ONVIF Device Service\n# This is a device management endpoint, not a stream\n# URL: ${stream.url}`;
            default:
                return this.generateRTSP(streamName, stream);
        }
    }

    static generateStreamName(stream) {
        try {
            const urlObj = new URL(stream.url);
            const ip = urlObj.hostname.replace(/\./g, '_').replace(/:/g, '_');
            return `${ip}_0`;
        } catch (e) {
            return 'camera_stream_0';
        }
    }

    static generateRTSP(streamName, stream) {
        return `streams:\n  '${streamName}':\n    - ${stream.url}`;
    }

    static generateJPEG(streamName, stream) {
        const framerate = 10;
        const ffmpegCmd = [
            'exec:ffmpeg',
            '-loglevel quiet',
            '-f image2',
            '-loop 1',
            `-framerate ${framerate}`,
            `-i ${stream.url}`,
            '-c:v libx264',
            '-preset ultrafast',
            '-tune zerolatency',
            '-g 20',
            '-f rtsp {output}'
        ].join(' ');

        return `streams:\n  '${streamName}':\n    - ${ffmpegCmd}`;
    }

    static generateMJPEG(streamName, stream) {
        const ffmpegCmd = [
            'exec:ffmpeg',
            '-loglevel quiet',
            `-i ${stream.url}`,
            '-c:v copy',
            '-f rtsp {output}'
        ].join(' ');

        return `streams:\n  '${streamName}':\n    - ${ffmpegCmd}`;
    }

    static generateHTTPVideo(streamName, stream) {
        const ffmpegCmd = [
            'exec:ffmpeg',
            '-loglevel quiet',
            `-i ${stream.url}`,
            '-c:v copy',
            '-c:a copy',
            '-f rtsp {output}'
        ].join(' ');

        return `streams:\n  '${streamName}':\n    - ${ffmpegCmd}`;
    }

    static generateHLS(streamName, stream) {
        return `streams:\n  '${streamName}':\n    - ${stream.url}`;
    }
}

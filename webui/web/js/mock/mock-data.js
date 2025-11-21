// Mock data for development and testing
export const MOCK_CAMERAS = [
    { brand: "Hikvision", model: "DS-2CD2143G0-I" },
    { brand: "Hikvision", model: "DS-2CD2385G1-I" },
    { brand: "Hikvision", model: "DS-2CD2T85G1-I8" },
    { brand: "Dahua", model: "IPC-HFW5831E-Z5E" },
    { brand: "Dahua", model: "IPC-HDW5831R-ZE" },
    { brand: "Axis", model: "M3046-V" },
    { brand: "Axis", model: "P3245-LVE" },
    { brand: "Uniview", model: "IPC2324LB-ADZK-G" },
    { brand: "Reolink", model: "RLC-810A" },
    { brand: "TP-Link", model: "VIGI C540V" }
];

export const MOCK_STREAMS = [
    {
        url: "rtsp://admin:password@192.168.1.100:554/stream1",
        type: "FFMPEG",
        resolution: "1920x1080",
        codec: "h264",
        fps: 25,
        bitrate: 4096,
        audio: true
    },
    {
        url: "rtsp://admin:password@192.168.1.100:554/stream2",
        type: "FFMPEG",
        resolution: "640x360",
        codec: "h264",
        fps: 15,
        bitrate: 512,
        audio: true
    },
    {
        url: "http://admin:password@192.168.1.100:80/onvif/device_service",
        type: "ONVIF",
        resolution: "1920x1080",
        codec: "h264",
        fps: 25,
        bitrate: 4096,
        audio: false
    },
    {
        url: "rtsp://admin:password@192.168.1.100/live/main",
        type: "FFMPEG",
        resolution: "2560x1440",
        codec: "h265",
        fps: 30,
        bitrate: 6144,
        audio: true
    },
    {
        url: "rtsp://admin:password@192.168.1.100/live/sub",
        type: "FFMPEG",
        resolution: "704x576",
        codec: "h264",
        fps: 15,
        bitrate: 768,
        audio: false
    },
    {
        url: "rtsp://admin:password@192.168.1.100:554/ch01/0",
        type: "FFMPEG",
        resolution: "3840x2160",
        codec: "h265",
        fps: 25,
        bitrate: 8192,
        audio: true
    },
    {
        url: "rtsp://admin:password@192.168.1.100:554/ch01/1",
        type: "FFMPEG",
        resolution: "1280x720",
        codec: "h264",
        fps: 20,
        bitrate: 2048,
        audio: false
    },
    {
        url: "http://admin:password@192.168.1.100:8080/video.mjpeg",
        type: "MJPEG",
        resolution: "1920x1080",
        codec: "mjpeg",
        fps: 10,
        bitrate: 3072,
        audio: false
    },
    {
        url: "rtsp://admin:password@192.168.1.100/h264_stream",
        type: "FFMPEG",
        resolution: "1920x1080",
        codec: "h264",
        fps: 30,
        bitrate: 4096,
        audio: true
    },
    {
        url: "http://admin:password@192.168.1.100:8081/stream.m3u8",
        type: "HLS",
        resolution: "1920x1080",
        codec: "h264",
        fps: 25,
        bitrate: 4096,
        audio: true
    }
];

// Mock Camera Search API
export class MockCameraSearch {
    async search(query, limit = 10) {
        // Simulate network delay
        await this.delay(100);

        const results = MOCK_CAMERAS.filter(camera => {
            const searchStr = `${camera.brand} ${camera.model}`.toLowerCase();
            return searchStr.includes(query.toLowerCase());
        });

        return {
            cameras: results.slice(0, limit),
            total: results.length
        };
    }

    delay(ms) {
        return new Promise(resolve => setTimeout(resolve, ms));
    }
}

// Mock Stream Discovery API
export class MockStreamDiscovery {
    constructor() {
        this.isRunning = false;
        this.timeoutId = null;
    }

    discover(request, callbacks) {
        this.isRunning = true;
        let tested = 0;
        const totalToTest = 516;
        const foundStreams = [...MOCK_STREAMS];

        // Initial progress
        callbacks.onProgress({
            tested: 0,
            found: 0,
            remaining: totalToTest
        });

        // Simulate progressive testing
        const progressInterval = setInterval(() => {
            if (!this.isRunning) {
                clearInterval(progressInterval);
                return;
            }

            tested += Math.floor(Math.random() * 30) + 20;
            if (tested > totalToTest) tested = totalToTest;

            callbacks.onProgress({
                tested: tested,
                found: foundStreams.length,
                remaining: totalToTest - tested
            });

            if (tested >= totalToTest) {
                clearInterval(progressInterval);
            }
        }, 200);

        // Send found streams progressively
        let streamIndex = 0;
        const streamInterval = setInterval(() => {
            if (!this.isRunning) {
                clearInterval(streamInterval);
                return;
            }

            if (streamIndex < foundStreams.length) {
                callbacks.onStreamFound({
                    stream: foundStreams[streamIndex]
                });
                streamIndex++;
            } else {
                clearInterval(streamInterval);
            }
        }, 800);

        // Complete after ~7.7 seconds
        this.timeoutId = setTimeout(() => {
            if (!this.isRunning) return;

            callbacks.onComplete({
                total_found: foundStreams.length,
                duration: 7.7
            });

            this.isRunning = false;
        }, 7700);
    }

    close() {
        this.isRunning = false;
        if (this.timeoutId) {
            clearTimeout(this.timeoutId);
            this.timeoutId = null;
        }
    }
}

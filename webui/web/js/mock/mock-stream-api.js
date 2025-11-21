// Mock implementation of StreamDiscoveryAPI for development
export class MockStreamAPI {
    constructor() {
        this.mockStreams = [
            {
                url: "rtsp://192.168.1.100:554/Streaming/Channels/101",
                path: "/Streaming/Channels/101",
                type: "FFMPEG",
                resolution: "1920x1080",
                codec: "H.264",
                fps: 25,
                bitrate: 4096000,
                has_audio: true
            },
            {
                url: "http://192.168.1.100/snap.jpg",
                path: "/snap.jpg",
                type: "JPEG",
                resolution: "1920x1080",
                codec: "JPEG",
                fps: 1,
                bitrate: 0,
                has_audio: false
            },
            {
                url: "http://192.168.1.100/video.mjpg",
                path: "/video.mjpg",
                type: "MJPEG",
                resolution: "1280x720",
                codec: "MJPEG",
                fps: 10,
                bitrate: 2048000,
                has_audio: false
            },
            {
                url: "http://192.168.1.100/stream/live.m3u8",
                path: "/stream/live.m3u8",
                type: "HLS",
                resolution: "1920x1080",
                codec: "H.264",
                fps: 25,
                bitrate: 3072000,
                has_audio: true
            },
            {
                url: "http://192.168.1.100/videostream.cgi?user=admin&pwd=12345",
                path: "/videostream.cgi?user=admin&pwd=12345",
                type: "HTTP_VIDEO",
                resolution: "1280x960",
                codec: "H.264",
                fps: 20,
                bitrate: 2048000,
                has_audio: false
            },
            {
                url: "rtsp://192.168.1.100:554/Streaming/Channels/102",
                path: "/Streaming/Channels/102",
                type: "FFMPEG",
                resolution: "640x480",
                codec: "H.264",
                fps: 15,
                bitrate: 512000,
                has_audio: false
            },
            {
                url: "rtsp://192.168.1.100:554/cam/realmonitor?channel=1&subtype=0",
                path: "/cam/realmonitor?channel=1&subtype=0",
                type: "ONVIF",
                resolution: "2560x1440",
                codec: "H.265",
                fps: 30,
                bitrate: 6144000,
                has_audio: true
            },
            {
                url: "rtsp://192.168.1.100:554/h264Preview_01_main",
                path: "/h264Preview_01_main",
                type: "FFMPEG",
                resolution: "1920x1080",
                codec: "H.264",
                fps: 20,
                bitrate: 3072000,
                has_audio: true
            },
            {
                url: "rtsp://192.168.1.100:554/live/ch0",
                path: "/live/ch0",
                type: "ONVIF",
                resolution: "2688x1520",
                codec: "H.265",
                fps: 25,
                bitrate: 5120000,
                has_audio: true
            },
            {
                url: "rtsp://192.168.1.100:554/stream1",
                path: "/stream1",
                type: "FFMPEG",
                resolution: "3840x2160",
                codec: "H.265",
                fps: 30,
                bitrate: 8192000,
                has_audio: true
            }
        ];
    }

    discover(request, callbacks) {
        const totalToScan = 150;
        const streamsToFind = this.mockStreams;
        let tested = 0;
        let found = 0;

        const startTime = Date.now();

        // Simulate progressive discovery
        const interval = setInterval(() => {
            const increment = Math.floor(Math.random() * 8) + 3;
            tested = Math.min(tested + increment, totalToScan);
            const remaining = totalToScan - tested;

            // Send progress event
            if (callbacks.onProgress) {
                callbacks.onProgress({
                    tested: tested,
                    found: found,
                    remaining: remaining
                });
            }

            // Randomly find streams
            if (found < streamsToFind.length && Math.random() > 0.6) {
                const stream = streamsToFind[found];
                found++;

                if (callbacks.onStreamFound) {
                    callbacks.onStreamFound({
                        stream: stream
                    });
                }
            }

            // Complete when done
            if (tested >= totalToScan) {
                clearInterval(interval);

                // Send any remaining streams
                while (found < streamsToFind.length) {
                    const stream = streamsToFind[found];
                    found++;

                    if (callbacks.onStreamFound) {
                        callbacks.onStreamFound({
                            stream: stream
                        });
                    }
                }

                const duration = (Date.now() - startTime) / 1000;

                if (callbacks.onComplete) {
                    callbacks.onComplete({
                        total_tested: totalToScan,
                        total_found: found,
                        duration: duration
                    });
                }
            }
        }, 400);
    }

    close() {
        // Nothing to close in mock mode
    }
}

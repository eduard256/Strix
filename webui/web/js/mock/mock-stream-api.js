// Mock implementation of StreamDiscoveryAPI for development
export class MockStreamAPI {
    constructor() {
        this.mockStreams = [
            // RTSP Main streams (1920x1080)
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
                url: "rtsp://192.168.1.100:554/live/main",
                path: "/live/main",
                type: "FFMPEG",
                resolution: "1920x1080",
                codec: "H.264",
                fps: 30,
                bitrate: 4608000,
                has_audio: true
            },
            {
                url: "rtsp://192.168.1.100:554/stream1",
                path: "/stream1",
                type: "FFMPEG",
                resolution: "1920x1080",
                codec: "H.265",
                fps: 25,
                bitrate: 3584000,
                has_audio: true
            },
            // JPEG snapshots (5 items in different positions)
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
            // RTSP Sub streams (640x480)
            {
                url: "rtsp://192.168.1.100:554/Streaming/Channels/102",
                path: "/Streaming/Channels/102",
                type: "FFMPEG",
                resolution: "640x480",
                codec: "H.264",
                fps: 5,
                bitrate: 512000,
                has_audio: true
            },
            {
                url: "rtsp://192.168.1.100:554/live/sub",
                path: "/live/sub",
                type: "FFMPEG",
                resolution: "640x480",
                codec: "H.264",
                fps: 10,
                bitrate: 768000,
                has_audio: false
            },
            // JPEG #2
            {
                url: "http://192.168.1.100/cgi-bin/snapshot.cgi",
                path: "/cgi-bin/snapshot.cgi",
                type: "JPEG",
                resolution: "1920x1080",
                codec: "JPEG",
                fps: 1,
                bitrate: 0,
                has_audio: false
            },
            {
                url: "rtsp://192.168.1.100:554/stream2",
                path: "/stream2",
                type: "FFMPEG",
                resolution: "640x480",
                codec: "H.264",
                fps: 15,
                bitrate: 640000,
                has_audio: false
            },
            // ONVIF streams
            {
                url: "rtsp://192.168.1.100:554/onvif/profile0",
                path: "/onvif/profile0",
                type: "ONVIF",
                resolution: "1920x1080",
                codec: "H.264",
                fps: 25,
                bitrate: 4096000,
                has_audio: true
            },
            {
                url: "rtsp://192.168.1.100:554/onvif/profile1",
                path: "/onvif/profile1",
                type: "ONVIF",
                resolution: "640x480",
                codec: "H.264",
                fps: 15,
                bitrate: 512000,
                has_audio: false
            },
            // JPEG #3
            {
                url: "http://192.168.1.100/image/jpeg.cgi",
                path: "/image/jpeg.cgi",
                type: "JPEG",
                resolution: "1920x1080",
                codec: "JPEG",
                fps: 1,
                bitrate: 0,
                has_audio: false
            },
            // More RTSP variants
            {
                url: "rtsp://192.168.1.100:554/cam/realmonitor?channel=1&subtype=0",
                path: "/cam/realmonitor?channel=1&subtype=0",
                type: "FFMPEG",
                resolution: "1920x1080",
                codec: "H.265",
                fps: 30,
                bitrate: 5120000,
                has_audio: true
            },
            {
                url: "rtsp://192.168.1.100:554/cam/realmonitor?channel=1&subtype=1",
                path: "/cam/realmonitor?channel=1&subtype=1",
                type: "FFMPEG",
                resolution: "640x480",
                codec: "H.264",
                fps: 10,
                bitrate: 512000,
                has_audio: false
            },
            // MJPEG
            {
                url: "http://192.168.1.100/video.mjpg",
                path: "/video.mjpg",
                type: "MJPEG",
                resolution: "1920x1080",
                codec: "MJPEG",
                fps: 10,
                bitrate: 3072000,
                has_audio: false
            },
            // JPEG #4
            {
                url: "http://192.168.1.100/Streaming/channels/1/picture",
                path: "/Streaming/channels/1/picture",
                type: "JPEG",
                resolution: "1920x1080",
                codec: "JPEG",
                fps: 1,
                bitrate: 0,
                has_audio: false
            },
            // HLS
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
            // HTTP Video
            {
                url: "http://192.168.1.100/videostream.cgi?user=admin&pwd=12345",
                path: "/videostream.cgi?user=admin&pwd=12345",
                type: "HTTP_VIDEO",
                resolution: "1920x1080",
                codec: "H.264",
                fps: 20,
                bitrate: 2560000,
                has_audio: false
            },
            // BUBBLE
            {
                url: "bubble://192.168.1.100:34567/bubble/live?ch=0&stream=0",
                path: "/bubble/live?ch=0&stream=0",
                type: "BUBBLE",
                resolution: "1920x1080",
                codec: "H.264",
                fps: 25,
                bitrate: 3584000,
                has_audio: true
            },
            // JPEG #5
            {
                url: "http://192.168.1.100/tmpfs/auto.jpg",
                path: "/tmpfs/auto.jpg",
                type: "JPEG",
                resolution: "1920x1080",
                codec: "JPEG",
                fps: 1,
                bitrate: 0,
                has_audio: false
            },
            // Additional RTSP
            {
                url: "rtsp://192.168.1.100:554/h264_stream",
                path: "/h264_stream",
                type: "FFMPEG",
                resolution: "1920x1080",
                codec: "H.264",
                fps: 30,
                bitrate: 4096000,
                has_audio: true
            },
            {
                url: "rtsp://192.168.1.100:554/av0_0",
                path: "/av0_0",
                type: "FFMPEG",
                resolution: "1920x1080",
                codec: "H.264",
                fps: 25,
                bitrate: 3840000,
                has_audio: true
            },
            {
                url: "rtsp://192.168.1.100:554/av0_1",
                path: "/av0_1",
                type: "FFMPEG",
                resolution: "640x480",
                codec: "H.264",
                fps: 10,
                bitrate: 512000,
                has_audio: false
            },
            {
                url: "rtsp://192.168.1.100:554/unicast/c1/s0/live",
                path: "/unicast/c1/s0/live",
                type: "FFMPEG",
                resolution: "1920x1080",
                codec: "H.265",
                fps: 25,
                bitrate: 4608000,
                has_audio: true
            }
        ];
    }

    discover(request, callbacks) {
        const totalToScan = 450;
        const streamsToFind = this.mockStreams;
        let tested = 0;
        let found = 0;

        const startTime = Date.now();

        // Simulate progressive discovery - 1 stream per second
        const progressInterval = setInterval(() => {
            const increment = Math.floor(Math.random() * 15) + 10;
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

            // Complete when done
            if (tested >= totalToScan) {
                clearInterval(progressInterval);

                const duration = (Date.now() - startTime) / 1000;

                if (callbacks.onComplete) {
                    callbacks.onComplete({
                        total_tested: totalToScan,
                        total_found: found,
                        duration: duration
                    });
                }
            }
        }, 300);

        // Find streams at ~1 per second
        const streamInterval = setInterval(() => {
            if (found < streamsToFind.length) {
                const stream = streamsToFind[found];
                found++;

                if (callbacks.onStreamFound) {
                    callbacks.onStreamFound({
                        stream: stream
                    });
                }
            } else {
                clearInterval(streamInterval);
            }
        }, 1000);
    }

    close() {
        // Nothing to close in mock mode
    }
}

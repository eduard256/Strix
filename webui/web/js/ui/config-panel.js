import { Go2RTCGenerator } from '../config-generators/go2rtc/index.js';
import { FrigateGenerator } from '../config-generators/frigate/index.js';

export class ConfigPanel {
    constructor() {
        this.stream = null;
    }

    render(stream) {
        this.stream = stream;

        // Update selected stream info
        document.getElementById('selected-stream-type').textContent = stream.type;
        document.getElementById('selected-stream-url').textContent = this.maskCredentials(stream.url);

        // Generate configs
        const urlConfig = stream.url;
        const go2rtcConfig = Go2RTCGenerator.generate(stream);
        const frigateConfig = FrigateGenerator.generate(stream);

        // Update config displays
        document.getElementById('config-url').textContent = urlConfig;
        document.getElementById('config-go2rtc').textContent = go2rtcConfig;
        document.getElementById('config-frigate').textContent = frigateConfig;
    }

    maskCredentials(url) {
        try {
            const urlObj = new URL(url);
            if (urlObj.username || urlObj.password) {
                urlObj.username = urlObj.username ? '***' : '';
                urlObj.password = urlObj.password ? '***' : '';
            }
            return urlObj.toString();
        } catch (e) {
            return url;
        }
    }
}

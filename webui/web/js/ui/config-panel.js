import { Go2RTCGenerator } from '../config-generators/go2rtc/index.js';
import { FrigateGenerator } from '../config-generators/frigate/index.js';

export class ConfigPanel {
    constructor() {
        this.mainStream = null;
        this.subStream = null;
    }

    render(mainStream, subStream = null) {
        this.mainStream = mainStream;
        this.subStream = subStream;

        // Update main stream info
        document.getElementById('selected-main-type').textContent = mainStream.type;
        document.getElementById('selected-main-url').textContent = this.maskCredentials(mainStream.url);

        // Update sub stream info if provided
        if (subStream) {
            document.getElementById('selected-sub-type').textContent = subStream.type;
            document.getElementById('selected-sub-url').textContent = this.maskCredentials(subStream.url);
        }

        // Generate configs
        const urlConfig = this.generateURLConfig();
        const go2rtcConfig = Go2RTCGenerator.generate(mainStream, subStream);
        const frigateConfig = FrigateGenerator.generate(mainStream, subStream);

        // Update config displays
        document.getElementById('config-url').textContent = urlConfig;
        document.getElementById('config-go2rtc').textContent = go2rtcConfig;
        document.getElementById('config-frigate').textContent = frigateConfig;
    }

    generateURLConfig() {
        if (this.subStream) {
            return `Main Stream:\n${this.mainStream.url}\n\nSub Stream:\n${this.subStream.url}`;
        }
        return this.mainStream.url;
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

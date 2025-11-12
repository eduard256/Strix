import { Go2RTCGenerator } from '../config-generators/go2rtc/index.js';
// import { FrigateGenerator } from '../config-generators/frigate/index.js'; // Reserved for future use

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

        // Generate configs for URL and Go2RTC (as before)
        const urlConfig = this.generateURLConfig();
        const go2rtcConfig = Go2RTCGenerator.generate(mainStream, subStream);

        // Update config displays
        document.getElementById('config-url').textContent = urlConfig;
        document.getElementById('config-go2rtc').textContent = go2rtcConfig;

        // For Frigate: initialize the tab instead of generating automatically
        this.initializeFrigateTab();
    }

    /**
     * Initialize Frigate tab with example config
     */
    initializeFrigateTab() {
        const textarea = document.getElementById('existing-frigate-config');
        const outputSection = document.getElementById('frigate-output-section');

        // Show example config if field is empty
        if (!textarea.value || textarea.value.trim() === '') {
            textarea.value = this.getExampleConfig();
        }

        // Hide output section
        outputSection.classList.add('hidden');
        document.getElementById('config-frigate').textContent = '';
    }

    /**
     * Get example Frigate config
     */
    getExampleConfig() {
        return `mqtt:
  enabled: false

# Global Recording Settings
record:
  enabled: true
  retain:
    days: 7
    mode: motion  # Record only on motion detection

# Go2RTC Configuration (Frigate built-in)
go2rtc:
  streams:
    # Your existing streams will be preserved here

# Frigate Camera Configuration
cameras:
  # Your existing cameras will be preserved here

version: 0.16-0`;
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

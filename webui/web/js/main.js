import { CameraSearchAPI } from './api/camera-search.js';
import { StreamDiscoveryAPI } from './api/stream-discovery.js';
import { MockCameraAPI } from './mock/mock-camera-api.js';
import { MockStreamAPI } from './mock/mock-stream-api.js';
import { SearchForm } from './ui/search-form.js';
import { StreamList } from './ui/stream-list.js';
import { ConfigPanel } from './ui/config-panel.js';
import { FrigateGenerator } from './config-generators/frigate/index.js';
import { showToast } from './utils/toast.js';

class StrixApp {
    constructor() {
        // Check if mock mode is enabled via URL parameter
        const urlParams = new URLSearchParams(window.location.search);
        const isMockMode = urlParams.get('mock') === 'true';

        if (isMockMode) {
            console.log('🎭 Mock mode enabled - using fake data');
            this.cameraAPI = new MockCameraAPI();
            this.streamAPI = new MockStreamAPI();

            // Show mock mode badge
            const mockBadge = document.getElementById('mock-mode-badge');
            if (mockBadge) {
                mockBadge.classList.remove('hidden');
            }
        } else {
            this.cameraAPI = new CameraSearchAPI();
            this.streamAPI = new StreamDiscoveryAPI();
        }

        this.searchForm = new SearchForm();
        this.streamList = new StreamList();
        this.configPanel = new ConfigPanel();

        this.currentAddress = '';
        this.currentStreams = [];
        this.selectedMainStream = null;
        this.selectedSubStream = null;
        this.isSelectingSubStream = false;
        this.frigateConfigGenerated = false; // Track if Frigate config has been generated

        this.init();
    }

    init() {
        this.setupEventListeners();
        this.prefillNetworkAddress();
        this.showScreen('address');
    }

    /**
     * Pre-fill network address input with smart default based on server IP
     */
    prefillNetworkAddress() {
        const hostname = window.location.hostname;
        const input = document.getElementById('network-address');

        // Skip if localhost or empty
        if (!hostname || hostname === 'localhost' || hostname === '127.0.0.1' || hostname === '0.0.0.0') {
            return;
        }

        // Check if hostname is an IP address (matches pattern like 192.168.1.1)
        const ipPattern = /^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/;
        const match = hostname.match(ipPattern);

        if (match) {
            // Extract first three octets (e.g., "192.168.1." from "192.168.1.254")
            const networkPrefix = `${match[1]}.${match[2]}.${match[3]}.`;
            input.value = networkPrefix;
            input.placeholder = `${networkPrefix}100`;
        }
    }

    setupEventListeners() {
        // Screen 1: Address input
        document.getElementById('btn-check-address').addEventListener('click', () => this.checkAddress());
        document.getElementById('network-address').addEventListener('keypress', (e) => {
            if (e.key === 'Enter') this.checkAddress();
        });

        // Screen 2: Configuration form
        document.getElementById('btn-back-to-address').addEventListener('click', () => {
            this.showScreen('address');
        });

        document.getElementById('btn-discover').addEventListener('click', () => this.discoverStreams());

        // Password toggle
        document.querySelector('.btn-toggle-password').addEventListener('click', () => {
            const input = document.getElementById('password');
            input.type = input.type === 'password' ? 'text' : 'password';
        });

        // Camera model autocomplete
        const modelInput = document.getElementById('camera-model');
        let debounceTimer;
        let extendedSearchTimer;
        modelInput.addEventListener('input', (e) => {
            clearTimeout(debounceTimer);
            clearTimeout(extendedSearchTimer);
            const query = e.target.value.trim();

            if (query.length >= 2) {
                debounceTimer = setTimeout(() => {
                    this.searchCameraModels(query, 10);

                    extendedSearchTimer = setTimeout(() => {
                        this.searchCameraModels(query, 50, true);
                    }, 1000);
                }, 300);
            } else {
                this.hideAutocomplete();
            }
        });

        // Screen 3: Stream discovery
        document.getElementById('btn-back-to-config').addEventListener('click', () => {
            this.streamAPI.close();
            this.showScreen('config');
        });

        // Screen 4: Configuration output
        document.getElementById('btn-back-to-streams').addEventListener('click', () => {
            this.isSelectingSubStream = false;
            this.showScreen('discovery');
        });

        document.getElementById('btn-copy-config').addEventListener('click', () => this.copyConfig());
        document.getElementById('btn-download-config').addEventListener('click', () => this.downloadConfig());

        document.getElementById('btn-add-sub-stream').addEventListener('click', () => this.addSubStream());
        document.getElementById('btn-remove-sub').addEventListener('click', () => this.removeSubStream());

        // Frigate config generation
        document.getElementById('btn-generate-frigate').addEventListener('click', () => this.generateFrigateConfig());

        document.getElementById('btn-new-search').addEventListener('click', () => {
            this.reset();
            this.showScreen('address');
        });

        // Tab switching
        document.querySelectorAll('.tab').forEach(tab => {
            tab.addEventListener('click', (e) => this.switchTab(e.target.dataset.tab));
        });
    }

    showScreen(screenName) {
        document.querySelectorAll('.screen').forEach(s => s.classList.remove('active'));
        document.getElementById(`screen-${screenName}`).classList.add('active');
    }

    async checkAddress() {
        const input = document.getElementById('network-address');
        const address = input.value.trim();

        if (!address) {
            showToast('Please enter a network address');
            return;
        }

        // Check if it's a full URL with credentials
        if (this.isFullURL(address)) {
            this.parseFullURL(address);
        } else {
            // Just an IP or hostname
            this.currentAddress = address;
            document.getElementById('address-validated').value = address;
        }

        this.showScreen('config');
    }

    isFullURL(str) {
        return str.startsWith('rtsp://') || str.startsWith('http://') || str.startsWith('https://');
    }

    parseFullURL(url) {
        try {
            const urlObj = new URL(url);

            // Extract credentials (only override if provided in URL)
            if (urlObj.username) {
                document.getElementById('username').value = urlObj.username;
            }
            // If no username in URL, keep the default "admin" value

            if (urlObj.password) {
                document.getElementById('password').value = urlObj.password;
            }

            // Extract IP/hostname
            this.currentAddress = urlObj.hostname;
            document.getElementById('address-validated').value = url;

            // Disable model input
            const modelInput = document.getElementById('camera-model');
            modelInput.disabled = true;
            modelInput.placeholder = 'Detected from URL';
            document.getElementById('model-disabled-hint').classList.remove('hidden');

        } catch (e) {
            this.currentAddress = url;
            document.getElementById('address-validated').value = url;
        }
    }

    async searchCameraModels(query, limit = 10, append = false) {
        const dropdown = document.getElementById('autocomplete-dropdown');

        // Keep dropdown open and show loading state smoothly
        if (!append) {
            const isOpen = !dropdown.classList.contains('hidden');
            if (!isOpen) {
                dropdown.classList.remove('hidden');
            }
            // Show loading only if dropdown was empty or closed
            if (!isOpen || dropdown.children.length === 0) {
                dropdown.innerHTML = '<div class="autocomplete-loading">Searching...</div>';
            }
        }

        try {
            const response = await this.cameraAPI.search(query, limit);

            if (response.cameras && response.cameras.length > 0) {
                this.renderAutocomplete(response.cameras, append);
            } else if (!append) {
                dropdown.innerHTML = '<div class="autocomplete-loading">No cameras found</div>';
            }
        } catch (error) {
            console.error('Search error:', error);
            if (!append) {
                dropdown.innerHTML = '<div class="autocomplete-loading">Search failed</div>';
            }
        }
    }

    renderAutocomplete(cameras, append = false) {
        const dropdown = document.getElementById('autocomplete-dropdown');
        const modelInput = document.getElementById('camera-model');

        const existingValues = new Set();
        if (append) {
            dropdown.querySelectorAll('.autocomplete-item').forEach(item => {
                existingValues.add(item.dataset.value);
            });
        }

        const newItems = cameras
            .map(camera => {
                const fullName = `${camera.brand}: ${camera.model}`;
                if (append && existingValues.has(fullName)) {
                    return null;
                }
                return `<div class="autocomplete-item" data-value="${fullName}">${fullName}</div>`;
            })
            .filter(item => item !== null)
            .join('');

        if (append) {
            dropdown.insertAdjacentHTML('beforeend', newItems);
        } else {
            dropdown.innerHTML = newItems;
        }

        dropdown.querySelectorAll('.autocomplete-item').forEach(item => {
            if (!item.hasAttribute('data-listener')) {
                item.setAttribute('data-listener', 'true');
                item.addEventListener('click', () => {
                    modelInput.value = item.dataset.value;
                    this.hideAutocomplete();
                });
            }
        });
    }

    hideAutocomplete() {
        document.getElementById('autocomplete-dropdown').classList.add('hidden');
    }

    async discoverStreams() {
        const model = document.getElementById('camera-model').value.trim();
        const username = document.getElementById('username').value.trim();
        const password = document.getElementById('password').value.trim();
        const channel = parseInt(document.getElementById('channel').value) || 0;
        const maxStreams = parseInt(document.getElementById('max-streams').value) || 10;

        const request = {
            target: this.currentAddress,
            model: model || 'auto',
            username: username,
            password: password,
            channel: channel,
            max_streams: maxStreams,
            timeout: 240
        };

        this.showScreen('discovery');
        this.resetDiscoveryUI();

        // Start SSE stream
        this.streamAPI.discover(request, {
            onProgress: (data) => this.handleProgress(data),
            onStreamFound: (data) => this.handleStreamFound(data),
            onComplete: (data) => this.handleComplete(data),
            onError: (error) => this.handleError(error)
        });
    }

    resetDiscoveryUI() {
        document.getElementById('progress-fill').style.width = '0%';
        document.getElementById('progress-text').textContent = 'Starting scan...';
        document.getElementById('streams-section').classList.add('hidden');
        this.currentStreams = [];
    }

    handleProgress(data) {
        const total = data.tested + data.remaining;
        const percentage = total > 0 ? (data.tested / total) * 100 : 0;

        document.getElementById('progress-fill').style.width = `${percentage}%`;
        document.getElementById('progress-text').textContent = `Testing streams... ${Math.round(percentage)}%`;
    }

    handleStreamFound(data) {
        this.currentStreams.push(data.stream);

        // Show streams section if hidden
        const streamsSection = document.getElementById('streams-section');
        if (streamsSection.classList.contains('hidden')) {
            streamsSection.classList.remove('hidden');
        }

        // Update stream list
        this.streamList.render(this.currentStreams, (stream, index) => {
            this.selectStream(stream, index);
        });
    }

    handleComplete(data) {
        document.getElementById('progress-fill').style.width = '100%';
        document.getElementById('progress-text').textContent =
            `Scan complete! Found ${data.total_found} stream(s) in ${data.duration.toFixed(1)}s`;

        if (this.currentStreams.length === 0) {
            showToast('No streams found. Try different credentials or model.');
        }
    }

    handleError(error) {
        console.error('Discovery error:', error);
        showToast(`Error: ${error}`);
    }

    selectStream(stream, index) {
        if (!this.isSelectingSubStream) {
            // Selecting main stream
            this.selectedMainStream = stream;
            this.selectedSubStream = null;
            this.frigateConfigGenerated = false; // Reset Frigate config state
            this.configPanel.render(this.selectedMainStream, this.selectedSubStream);
            this.updateSubStreamUI();
            this.showScreen('output');
            // Hide action buttons initially since Frigate tab is active by default
            document.querySelector('.actions').style.display = 'none';
        } else {
            // Selecting sub stream
            this.selectedSubStream = stream;
            this.isSelectingSubStream = false;
            this.frigateConfigGenerated = false; // Reset Frigate config state
            this.configPanel.render(this.selectedMainStream, this.selectedSubStream);
            this.updateSubStreamUI();
            this.showScreen('output');
            // Hide action buttons initially since Frigate tab is active by default
            document.querySelector('.actions').style.display = 'none';
        }
    }

    addSubStream() {
        if (this.currentStreams.length === 0) {
            showToast('No streams available to select');
            return;
        }

        this.isSelectingSubStream = true;

        // Clear Frigate output section (but NOT the user's input textarea)
        document.getElementById('frigate-output-section').classList.add('hidden');
        document.getElementById('config-frigate').textContent = '';

        showToast('Select a sub stream from available streams');
        this.showScreen('discovery');
    }

    removeSubStream() {
        this.selectedSubStream = null;
        this.frigateConfigGenerated = false; // Reset Frigate config state when sub stream is removed
        this.configPanel.render(this.selectedMainStream, this.selectedSubStream);
        this.updateSubStreamUI();

        // Hide action buttons if on Frigate tab
        const activeTab = document.querySelector('.tab.active').dataset.tab;
        if (activeTab === 'frigate') {
            document.querySelector('.actions').style.display = 'none';
        }

        showToast('Sub stream removed');
    }

    /**
     * Generate Frigate config by adding camera to existing config
     */
    generateFrigateConfig() {
        const existingConfig = document.getElementById('existing-frigate-config').value;
        const mainStream = this.selectedMainStream;
        const subStream = this.selectedSubStream;

        if (!mainStream) {
            showToast('No main stream selected', 'error');
            return;
        }

        try {
            // Generate config using FrigateGenerator
            const newConfig = FrigateGenerator.generate(existingConfig, mainStream, subStream);

            // Show result
            document.getElementById('config-frigate').textContent = newConfig;
            document.getElementById('frigate-output-section').classList.remove('hidden');

            // Mark as generated and show action buttons
            this.frigateConfigGenerated = true;
            document.querySelector('.actions').style.display = 'flex';

            // Scroll to result
            document.getElementById('frigate-output-section').scrollIntoView({
                behavior: 'smooth',
                block: 'nearest'
            });

            showToast('Config generated successfully!');
        } catch (error) {
            showToast(`Error: ${error.message}`, 'error');
            console.error('Config generation error:', error);
        }
    }

    updateSubStreamUI() {
        const subStreamInfo = document.getElementById('sub-stream-info');
        const addSubStreamBtn = document.getElementById('btn-add-sub-stream');

        if (this.selectedSubStream) {
            subStreamInfo.classList.remove('hidden');
            addSubStreamBtn.style.display = 'none';
        } else {
            subStreamInfo.classList.add('hidden');
            addSubStreamBtn.style.display = 'inline-flex';
        }
    }

    switchTab(tabName) {
        // Update tab buttons
        document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
        document.querySelector(`.tab[data-tab="${tabName}"]`).classList.add('active');

        // Update tab panes
        document.querySelectorAll('.tab-pane').forEach(p => p.classList.remove('active'));
        document.querySelector(`.tab-pane[data-pane="${tabName}"]`).classList.add('active');

        // Show/hide action buttons based on tab and Frigate config state
        const actionsContainer = document.querySelector('.actions');
        if (tabName === 'frigate' && !this.frigateConfigGenerated) {
            // Hide buttons on Frigate tab until config is generated
            actionsContainer.style.display = 'none';
        } else {
            // Show buttons for other tabs or after Frigate config is generated
            actionsContainer.style.display = 'flex';
        }
    }

    copyConfig() {
        const activeTab = document.querySelector('.tab.active').dataset.tab;
        const configElement = document.getElementById(`config-${activeTab}`);
        const text = configElement.textContent;

        const textarea = document.createElement('textarea');
        textarea.value = text;
        textarea.style.position = 'fixed';
        textarea.style.left = '-9999px';
        document.body.appendChild(textarea);
        textarea.select();

        try {
            document.execCommand('copy');
            showToast('Copied to clipboard!');
        } catch (err) {
            showToast('Failed to copy');
            console.error('Copy error:', err);
        } finally {
            document.body.removeChild(textarea);
        }
    }

    downloadConfig() {
        const activeTab = document.querySelector('.tab.active').dataset.tab;
        const configElement = document.getElementById(`config-${activeTab}`);
        const text = configElement.textContent;

        const filename = activeTab === 'url' ? 'stream-url.txt' : `${activeTab}-config.yaml`;
        const blob = new Blob([text], { type: 'text/plain' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = filename;
        a.click();
        URL.revokeObjectURL(url);

        showToast('Downloaded!');
    }

    reset() {
        this.currentAddress = '';
        this.currentStreams = [];
        this.selectedMainStream = null;
        this.selectedSubStream = null;
        this.isSelectingSubStream = false;

        document.getElementById('network-address').value = '';
        document.getElementById('camera-model').value = '';
        document.getElementById('camera-model').disabled = false;
        document.getElementById('camera-model').placeholder = 'Start typing...';
        document.getElementById('username').value = 'admin'; // Reset to default value
        document.getElementById('password').value = '';
        document.getElementById('channel').value = '0';
        document.getElementById('max-streams').value = '10';
        document.getElementById('model-disabled-hint').classList.add('hidden');

        this.hideAutocomplete();
        this.streamAPI.close();
    }
}

// Initialize app
const app = new StrixApp();

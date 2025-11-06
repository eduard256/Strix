import { CameraSearchAPI } from './api/camera-search.js';
import { StreamDiscoveryAPI } from './api/stream-discovery.js';
import { SearchForm } from './ui/search-form.js';
import { StreamCarousel } from './ui/stream-carousel.js';
import { ConfigPanel } from './ui/config-panel.js';
import { showToast } from './utils/toast.js';

class StrixApp {
    constructor() {
        this.cameraAPI = new CameraSearchAPI();
        this.streamAPI = new StreamDiscoveryAPI();

        this.searchForm = new SearchForm();
        this.carousel = new StreamCarousel();
        this.configPanel = new ConfigPanel();

        this.currentAddress = '';
        this.currentStreams = [];
        this.selectedMainStream = null;
        this.selectedSubStream = null;
        this.isSelectingSubStream = false;

        this.init();
    }

    init() {
        this.setupEventListeners();
        this.showScreen('address');
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

        // Carousel navigation
        document.getElementById('carousel-prev').addEventListener('click', () => {
            this.carousel.prev();
        });

        document.getElementById('carousel-next').addEventListener('click', () => {
            this.carousel.next();
        });

        // Keyboard navigation
        document.addEventListener('keydown', (e) => {
            const currentScreen = document.querySelector('.screen.active').id;
            if (currentScreen === 'screen-discovery') {
                if (e.key === 'ArrowLeft') this.carousel.prev();
                if (e.key === 'ArrowRight') this.carousel.next();
            }
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

            // Extract credentials
            if (urlObj.username) {
                document.getElementById('username').value = urlObj.username;
            }
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
        document.getElementById('stat-tested').textContent = '0';
        document.getElementById('stat-found').textContent = '0';
        document.getElementById('stat-remaining').textContent = '0';
        document.getElementById('streams-section').classList.add('hidden');
        this.currentStreams = [];
    }

    handleProgress(data) {
        const total = data.tested + data.remaining;
        const percentage = total > 0 ? (data.tested / total) * 100 : 0;

        document.getElementById('progress-fill').style.width = `${percentage}%`;
        document.getElementById('progress-text').textContent = `Testing streams... ${Math.round(percentage)}%`;
        document.getElementById('stat-tested').textContent = data.tested;
        document.getElementById('stat-found').textContent = data.found;
        document.getElementById('stat-remaining').textContent = data.remaining;
    }

    handleStreamFound(data) {
        this.currentStreams.push(data.stream);

        // Show streams section if hidden
        const streamsSection = document.getElementById('streams-section');
        if (streamsSection.classList.contains('hidden')) {
            streamsSection.classList.remove('hidden');
        }

        // Update carousel
        this.carousel.render(this.currentStreams, (stream, index) => {
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
            this.configPanel.render(this.selectedMainStream, this.selectedSubStream);
            this.updateSubStreamUI();
            this.showScreen('output');
        } else {
            // Selecting sub stream
            this.selectedSubStream = stream;
            this.isSelectingSubStream = false;
            this.configPanel.render(this.selectedMainStream, this.selectedSubStream);
            this.updateSubStreamUI();
            this.showScreen('output');
        }
    }

    addSubStream() {
        if (this.currentStreams.length === 0) {
            showToast('No streams available to select');
            return;
        }

        this.isSelectingSubStream = true;
        showToast('Select a sub stream from available streams');
        this.showScreen('discovery');
    }

    removeSubStream() {
        this.selectedSubStream = null;
        this.configPanel.render(this.selectedMainStream, this.selectedSubStream);
        this.updateSubStreamUI();
        showToast('Sub stream removed');
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
        document.getElementById('username').value = '';
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

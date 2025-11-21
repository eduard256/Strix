export class StreamList {
    constructor() {
        this.listContainer = document.getElementById('streams-list');
        this.streams = [];
        this.onUseCallback = null;
        this.expandedIndex = null;
    }

    render(streams, onUseCallback) {
        this.streams = streams;
        this.onUseCallback = onUseCallback;

        // Render stream items
        this.listContainer.innerHTML = streams.map((stream, index) => this.renderItem(stream, index)).join('');

        // Attach event listeners
        this.attachEventListeners();
    }

    renderItem(stream, index) {
        const icon = this.getStreamIcon(stream.type);
        const isExpanded = this.expandedIndex === index;
        const truncatedUrl = this.truncateURL(stream.url, 60);

        return `
            <div class="stream-item ${isExpanded ? 'expanded' : ''}" data-index="${index}">
                <div class="stream-item-header" data-index="${index}">
                    <div class="stream-item-main">
                        <div class="stream-info-left">
                            <div class="stream-type-badge">
                                ${icon}
                                <span>${stream.type}</span>
                                ${this.getStreamTypeTooltip(stream.type)}
                            </div>
                            <div class="stream-url-preview">${truncatedUrl}</div>
                        </div>
                        <button class="stream-toggle" data-index="${index}" aria-label="Toggle details">
                            <svg width="16" height="16" viewBox="0 0 16 16" fill="none" class="chevron">
                                <path d="M4 6l4 4 4-4" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
                            </svg>
                        </button>
                    </div>
                    <button class="btn btn-primary btn-use-stream" data-index="${index}">Use Stream</button>
                </div>
                <div class="stream-item-details ${isExpanded ? 'visible' : ''}">
                    <div class="stream-url-full">${stream.url}</div>
                    ${stream.resolution ? `<div class="stream-meta-item"><span class="meta-label">Resolution:</span> ${stream.resolution}</div>` : ''}
                    ${stream.codec ? `<div class="stream-meta-item"><span class="meta-label">Codec:</span> ${stream.codec}${stream.fps ? ` • ${stream.fps} fps` : ''}${stream.bitrate ? ` • ${Math.round(stream.bitrate / 1000)} Kbps` : ''}</div>` : ''}
                    ${stream.has_audio ? '<div class="stream-meta-item"><span class="meta-label">Audio:</span> Yes</div>' : ''}
                </div>
            </div>
        `;
    }

    truncateURL(url, maxLength = 60) {
        if (url.length <= maxLength) {
            return url;
        }
        return url.substring(0, maxLength) + '...';
    }

    getStreamIcon(type) {
        const icons = {
            'FFMPEG': '<svg width="20" height="20" viewBox="0 0 20 20" fill="none"><rect x="3" y="4" width="14" height="12" rx="1" stroke="currentColor" stroke-width="1.5"/><circle cx="7" cy="8" r="1" fill="currentColor"/><path d="M14 14l-3-2-3 2V8l3 2 3-2v6z" fill="currentColor"/></svg>',
            'ONVIF': '<svg width="20" height="20" viewBox="0 0 20 20" fill="none"><circle cx="10" cy="10" r="2" fill="currentColor"/><circle cx="10" cy="10" r="5" stroke="currentColor" stroke-width="1.5" stroke-dasharray="2 2"/><circle cx="10" cy="10" r="8" stroke="currentColor" stroke-width="1.5" stroke-dasharray="3 3"/></svg>',
            'JPEG': '<svg width="20" height="20" viewBox="0 0 20 20" fill="none"><rect x="3" y="4" width="14" height="12" rx="1" stroke="currentColor" stroke-width="1.5"/><circle cx="7" cy="8" r="1" fill="currentColor"/><path d="M3 13l4-4 3 3 5-5" stroke="currentColor" stroke-width="1.5"/></svg>',
            'MJPEG': '<svg width="20" height="20" viewBox="0 0 20 20" fill="none"><rect x="2" y="4" width="7" height="12" rx="1" stroke="currentColor" stroke-width="1.5"/><rect x="11" y="4" width="7" height="12" rx="1" stroke="currentColor" stroke-width="1.5"/><path d="M5 8l2 2-2 2M14 8l2 2-2 2" stroke="currentColor" stroke-width="1.5"/></svg>',
            'HLS': '<svg width="20" height="20" viewBox="0 0 20 20" fill="none"><circle cx="10" cy="10" r="7" stroke="currentColor" stroke-width="1.5"/><path d="M10 6v8M6 10h8" stroke="currentColor" stroke-width="1.5"/></svg>',
            'HTTP_VIDEO': '<svg width="20" height="20" viewBox="0 0 20 20" fill="none"><path d="M7 6l6 4-6 4V6z" fill="currentColor"/><circle cx="10" cy="10" r="8" stroke="currentColor" stroke-width="1.5"/></svg>',
            'BUBBLE': '<svg width="20" height="20" viewBox="0 0 20 20" fill="none"><circle cx="10" cy="10" r="7" stroke="currentColor" stroke-width="1.5"/><circle cx="7" cy="9" r="1.5" fill="currentColor"/><circle cx="10" cy="9" r="1.5" fill="currentColor"/><circle cx="13" cy="9" r="1.5" fill="currentColor"/><path d="M6 13q2 2 4 2t4-2" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/></svg>'
        };
        return icons[type] || icons['FFMPEG'];
    }

    getStreamTypeTooltip(type) {
        const tooltips = {
            'FFMPEG': `
                <span class="info-icon info-icon-stream">
                    <svg viewBox="0 0 16 16" fill="none">
                        <circle cx="8" cy="8" r="7" stroke="currentColor" stroke-width="1.5"/>
                        <path d="M8 7v4M8 5v.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
                    </svg>
                    <div class="tooltip tooltip-down">
                        <div class="tooltip-title">FFMPEG Stream</div>
                        <p class="tooltip-text">Standard video stream decoded by FFmpeg. Most compatible and widely supported format for RTSP, HTTP, and other protocols.</p>
                        <div class="tooltip-examples">
                            <div class="tooltip-examples-title">Features:</div>
                            <code class="tooltip-example">✓ Universal compatibility</code>
                            <code class="tooltip-example">✓ Supports H.264, H.265, MJPEG</code>
                            <code class="tooltip-example">✓ Works with most cameras</code>
                            <code class="tooltip-example">✓ Best for recording</code>
                        </div>
                        <p class="tooltip-text"><strong>Best for:</strong> Main streams, recording, high-quality playback. Default choice for most use cases.</p>
                    </div>
                </span>
            `,
            'ONVIF': `
                <span class="info-icon info-icon-stream">
                    <svg viewBox="0 0 16 16" fill="none">
                        <circle cx="8" cy="8" r="7" stroke="currentColor" stroke-width="1.5"/>
                        <path d="M8 7v4M8 5v.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
                    </svg>
                    <div class="tooltip tooltip-down">
                        <div class="tooltip-title">ONVIF Stream</div>
                        <p class="tooltip-text">Industry standard protocol for IP cameras. Discovered via ONVIF specification, ensuring maximum compatibility with camera features.</p>
                        <div class="tooltip-examples">
                            <div class="tooltip-examples-title">Features:</div>
                            <code class="tooltip-example">✓ Standardized protocol</code>
                            <code class="tooltip-example">✓ Auto-discovery support</code>
                            <code class="tooltip-example">✓ PTZ control capable</code>
                            <code class="tooltip-example">✓ Vendor-independent</code>
                        </div>
                        <p class="tooltip-text"><strong>Best for:</strong> Enterprise cameras, systems requiring standardization, cameras with PTZ controls.</p>
                    </div>
                </span>
            `,
            'JPEG': `
                <span class="info-icon info-icon-stream">
                    <svg viewBox="0 0 16 16" fill="none">
                        <circle cx="8" cy="8" r="7" stroke="currentColor" stroke-width="1.5"/>
                        <path d="M8 7v4M8 5v.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
                    </svg>
                    <div class="tooltip tooltip-down">
                        <div class="tooltip-title">JPEG Snapshot</div>
                        <p class="tooltip-text">Single static image endpoint. Can be converted to video stream by repeatedly fetching images.</p>
                        <div class="tooltip-examples">
                            <div class="tooltip-examples-title">Features:</div>
                            <code class="tooltip-example">✓ Low bandwidth</code>
                            <code class="tooltip-example">✓ Simple HTTP request</code>
                            <code class="tooltip-example">✓ No streaming protocol needed</code>
                            <code class="tooltip-example">⚠ Limited framerate (1-10 fps)</code>
                        </div>
                        <p class="tooltip-text"><strong>Best for:</strong> Thumbnails, snapshots, very low bandwidth scenarios. Not recommended for recording.</p>
                    </div>
                </span>
            `,
            'MJPEG': `
                <span class="info-icon info-icon-stream">
                    <svg viewBox="0 0 16 16" fill="none">
                        <circle cx="8" cy="8" r="7" stroke="currentColor" stroke-width="1.5"/>
                        <path d="M8 7v4M8 5v.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
                    </svg>
                    <div class="tooltip tooltip-down">
                        <div class="tooltip-title">MJPEG Stream</div>
                        <p class="tooltip-text">Motion JPEG - sequence of JPEG images transmitted continuously. Simple but bandwidth-intensive.</p>
                        <div class="tooltip-examples">
                            <div class="tooltip-examples-title">Features:</div>
                            <code class="tooltip-example">✓ Simple HTTP streaming</code>
                            <code class="tooltip-example">✓ No complex codecs</code>
                            <code class="tooltip-example">✓ Frame-by-frame</code>
                            <code class="tooltip-example">⚠ High bandwidth usage</code>
                        </div>
                        <p class="tooltip-text"><strong>Best for:</strong> Sub streams, low-latency monitoring, simple camera integration. Higher bandwidth than H.264.</p>
                    </div>
                </span>
            `,
            'HLS': `
                <span class="info-icon info-icon-stream">
                    <svg viewBox="0 0 16 16" fill="none">
                        <circle cx="8" cy="8" r="7" stroke="currentColor" stroke-width="1.5"/>
                        <path d="M8 7v4M8 5v.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
                    </svg>
                    <div class="tooltip tooltip-down">
                        <div class="tooltip-title">HLS Stream</div>
                        <p class="tooltip-text">HTTP Live Streaming - Apple's adaptive bitrate streaming protocol. Delivers video in small chunks over HTTP.</p>
                        <div class="tooltip-examples">
                            <div class="tooltip-examples-title">Features:</div>
                            <code class="tooltip-example">✓ Adaptive bitrate</code>
                            <code class="tooltip-example">✓ Wide browser support</code>
                            <code class="tooltip-example">✓ Firewall-friendly (HTTP)</code>
                            <code class="tooltip-example">⚠ Higher latency (5-30s)</code>
                        </div>
                        <p class="tooltip-text"><strong>Best for:</strong> Web playback, public streaming, CDN delivery. Not ideal for real-time monitoring.</p>
                    </div>
                </span>
            `,
            'HTTP_VIDEO': `
                <span class="info-icon info-icon-stream">
                    <svg viewBox="0 0 16 16" fill="none">
                        <circle cx="8" cy="8" r="7" stroke="currentColor" stroke-width="1.5"/>
                        <path d="M8 7v4M8 5v.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
                    </svg>
                    <div class="tooltip tooltip-down">
                        <div class="tooltip-title">HTTP Video Stream</div>
                        <p class="tooltip-text">Generic HTTP-based video stream. Simple protocol that works over standard web connections.</p>
                        <div class="tooltip-examples">
                            <div class="tooltip-examples-title">Features:</div>
                            <code class="tooltip-example">✓ Simple HTTP protocol</code>
                            <code class="tooltip-example">✓ No special ports</code>
                            <code class="tooltip-example">✓ Firewall-friendly</code>
                            <code class="tooltip-example">✓ Direct browser playback</code>
                        </div>
                        <p class="tooltip-text"><strong>Best for:</strong> Quick viewing, simple setups, scenarios where RTSP ports are blocked.</p>
                    </div>
                </span>
            `,
            'BUBBLE': `
                <span class="info-icon info-icon-stream">
                    <svg viewBox="0 0 16 16" fill="none">
                        <circle cx="8" cy="8" r="7" stroke="currentColor" stroke-width="1.5"/>
                        <path d="M8 7v4M8 5v.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
                    </svg>
                    <div class="tooltip tooltip-down">
                        <div class="tooltip-title">BUBBLE / DVRIP Protocol</div>
                        <p class="tooltip-text">Proprietary protocol for Chinese DVR/NVR cameras. Also known as: ESeeCloud, dvr163, DVR-IP, NetSurveillance, Sofia protocol, XMeye SDK.</p>
                        <div class="tooltip-examples">
                            <div class="tooltip-examples-title">Compatible brands:</div>
                            <code class="tooltip-example">XMEye, Floureon, ZOSI</code>
                            <code class="tooltip-example">Sannce, Annke, DVR163</code>
                            <code class="tooltip-example">ESeeCloud, NetSurveillance</code>
                        </div>
                        <div class="tooltip-examples">
                            <div class="tooltip-examples-title">Features:</div>
                            <code class="tooltip-example">⚠ Proprietary protocol</code>
                            <code class="tooltip-example">✓ Go2RTC converts to standard</code>
                            <code class="tooltip-example">✓ Two-way audio support</code>
                            <code class="tooltip-example">⚠ TCP only (port 34567)</code>
                        </div>
                        <p class="tooltip-text"><strong>Note:</strong> Automatically converted to standard RTSP format by Go2RTC. Works seamlessly with Frigate without additional configuration.</p>
                    </div>
                </span>
            `
        };
        return tooltips[type] || '';
    }

    attachEventListeners() {
        // Click on header to toggle
        this.listContainer.querySelectorAll('.stream-item-header').forEach(header => {
            header.addEventListener('click', (e) => {
                // Don't toggle if clicking "Use Stream" button
                if (e.target.closest('.btn-use-stream')) {
                    return;
                }

                const index = parseInt(header.dataset.index);
                this.toggleExpand(index);
            });
        });

        // Use Stream buttons
        this.listContainer.querySelectorAll('.btn-use-stream').forEach(btn => {
            btn.addEventListener('click', (e) => {
                e.stopPropagation(); // Prevent toggle
                const index = parseInt(e.target.dataset.index);
                if (this.onUseCallback) {
                    this.onUseCallback(this.streams[index], index);
                }
            });
        });
    }

    toggleExpand(index) {
        if (this.expandedIndex === index) {
            // Collapse if already expanded
            this.expandedIndex = null;
        } else {
            // Expand new item
            this.expandedIndex = index;
        }

        // Re-render to update state
        this.render(this.streams, this.onUseCallback);
    }
}

export class StreamList {
    constructor() {
        this.listContainer = document.getElementById('streams-list');
        this.streams = [];
        this.onUseCallback = null;
        this.expandedIndex = null;
        // Track collapsed state for groups and subgroups
        this.collapsedGroups = new Set();
        this.collapsedSubgroups = new Set();
        // Selection mode: 'main' or 'sub'
        this.selectionMode = 'main';
        // Flag to apply smart defaults on first render after reset
        this.needsSmartDefaults = true;
    }

    /**
     * Set selection mode and apply smart defaults for collapsed state
     * Only resets collapsed state when mode actually changes
     */
    setSelectionMode(mode) {
        if (this.selectionMode === mode) return;

        this.selectionMode = mode;
        this.applySmartDefaults();
    }

    /**
     * Apply smart collapsed defaults based on current selection mode and available streams
     */
    applySmartDefaults() {
        // Get current stream classification
        const recommended = this.streams.filter(s => this.isRecommended(s));
        const { main, sub, other } = this.classifyRecommendedStreams(
            recommended.map((stream, i) => ({ stream, index: i }))
        );

        // Reset all collapsed states
        this.collapsedGroups.clear();
        this.collapsedSubgroups.clear();

        if (this.selectionMode === 'main') {
            // Main mode: show Main, collapse Sub/Other/Alternative
            if (main.length > 0) {
                // Has main streams - collapse everything except Main
                this.collapsedGroups.add('alternative');
                this.collapsedSubgroups.add('recommended-sub');
                this.collapsedSubgroups.add('recommended-other');
            }
            // If no main streams - leave everything open
        } else {
            // Sub mode: show Sub, collapse Main/Other/Alternative
            if (sub.length > 0) {
                // Has sub streams - collapse everything except Sub
                this.collapsedGroups.add('alternative');
                this.collapsedSubgroups.add('recommended-main');
                this.collapsedSubgroups.add('recommended-other');
            }
            // If no sub streams - leave everything open
        }
    }

    // Stream types considered "recommended" (standard video streams)
    static RECOMMENDED_TYPES = ['FFMPEG', 'ONVIF'];

    // Minimum width threshold for Main streams (HD quality)
    static MIN_MAIN_WIDTH = 720;

    // Minimum gap between resolutions to split Main/Sub
    static MIN_GAP_FOR_SPLIT = 400;

    isRecommended(stream) {
        return StreamList.RECOMMENDED_TYPES.includes(stream.type);
    }

    /**
     * Parse resolution string "1920x1080" to width number
     * Returns null if resolution is missing or invalid
     */
    parseResolutionWidth(resolution) {
        if (!resolution) return null;
        const match = resolution.match(/^(\d+)x(\d+)$/);
        if (!match) return null;
        return parseInt(match[1], 10);
    }

    /**
     * Classify recommended streams into Main/Sub/Other using clustering algorithm
     *
     * Algorithm:
     * 1. Streams with width >= 720 are candidates for Main
     * 2. Streams with width < 720 go to Sub
     * 3. Streams without resolution go to Other
     * 4. Among Main candidates, find max gap between sorted resolutions
     * 5. If gap > 400px, split into Main (higher) and Sub (lower)
     */
    classifyRecommendedStreams(items) {
        const main = [];
        const sub = [];
        const other = [];

        // First pass: separate by resolution availability and threshold
        const mainCandidates = []; // width >= 720

        items.forEach(item => {
            const width = this.parseResolutionWidth(item.stream.resolution);

            if (width === null) {
                other.push(item);
            } else if (width < StreamList.MIN_MAIN_WIDTH) {
                sub.push(item);
            } else {
                mainCandidates.push({ ...item, width });
            }
        });

        // If no main candidates or only one, no need to cluster
        if (mainCandidates.length <= 1) {
            mainCandidates.forEach(item => main.push({ stream: item.stream, index: item.index }));
            return { main, sub, other };
        }

        // Sort candidates by width descending
        mainCandidates.sort((a, b) => b.width - a.width);

        // Find the largest gap between adjacent resolutions
        let maxGap = 0;
        let splitIndex = -1;

        for (let i = 0; i < mainCandidates.length - 1; i++) {
            const gap = mainCandidates[i].width - mainCandidates[i + 1].width;
            if (gap > maxGap) {
                maxGap = gap;
                splitIndex = i;
            }
        }

        // If max gap is significant, split into Main and Sub
        if (maxGap > StreamList.MIN_GAP_FOR_SPLIT && splitIndex >= 0) {
            mainCandidates.forEach((item, i) => {
                const cleanItem = { stream: item.stream, index: item.index };
                if (i <= splitIndex) {
                    main.push(cleanItem);
                } else {
                    sub.push(cleanItem);
                }
            });
        } else {
            // All candidates stay in Main
            mainCandidates.forEach(item => {
                main.push({ stream: item.stream, index: item.index });
            });
        }

        return { main, sub, other };
    }

    render(streams, onUseCallback) {
        this.streams = streams;
        this.onUseCallback = onUseCallback;

        // Apply smart defaults on first render after reset
        if (this.needsSmartDefaults && streams.length > 0) {
            this.needsSmartDefaults = false;
            this.applySmartDefaults();
        }

        // Split streams into groups while preserving original indices
        const recommended = [];
        const alternative = [];

        streams.forEach((stream, index) => {
            if (this.isRecommended(stream)) {
                recommended.push({ stream, index });
            } else {
                alternative.push({ stream, index });
            }
        });

        // Render only non-empty groups
        let html = '';
        if (recommended.length > 0) {
            html += this.renderRecommendedGroup(recommended);
        }
        if (alternative.length > 0) {
            html += this.renderGroup('Alternative', alternative, 'alternative');
        }
        this.listContainer.innerHTML = html;

        // Attach event listeners
        this.attachEventListeners();
    }

    /**
     * Render Recommended group with Main/Sub/Other subgroups
     */
    renderRecommendedGroup(items) {
        const { main, sub, other } = this.classifyRecommendedStreams(items);
        const totalCount = items.length;
        const isCollapsed = this.collapsedGroups.has('recommended');

        let subgroupsHtml = '';

        if (main.length > 0) {
            subgroupsHtml += this.renderSubgroup('Main', main, 'recommended');
        }
        if (sub.length > 0) {
            subgroupsHtml += this.renderSubgroup('Sub', sub, 'recommended');
        }
        if (other.length > 0) {
            subgroupsHtml += this.renderSubgroup('Other', other, 'recommended');
        }

        return `
            <div class="stream-group stream-group-recommended ${isCollapsed ? 'collapsed' : ''}">
                <div class="stream-group-header" data-group="recommended">
                    <button class="stream-group-toggle" aria-label="Toggle group">
                        <svg width="12" height="12" viewBox="0 0 12 12" fill="none" class="chevron">
                            <path d="M3 4.5l3 3 3-3" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
                        </svg>
                    </button>
                    <span class="stream-group-title">Recommended</span>
                    <span class="stream-group-count">(${totalCount})</span>
                </div>
                <div class="stream-group-content">
                    ${subgroupsHtml}
                </div>
            </div>
        `;
    }

    /**
     * Render a subgroup (Main/Sub/Other) within Recommended
     */
    renderSubgroup(title, items, parentGroup) {
        const subgroupKey = `${parentGroup}-${title.toLowerCase()}`;
        const isCollapsed = this.collapsedSubgroups.has(subgroupKey);

        return `
            <div class="stream-subgroup ${isCollapsed ? 'collapsed' : ''}" data-subgroup="${subgroupKey}">
                <div class="stream-subgroup-header" data-subgroup="${subgroupKey}">
                    <button class="stream-subgroup-toggle" aria-label="Toggle subgroup">
                        <svg width="10" height="10" viewBox="0 0 10 10" fill="none" class="chevron">
                            <path d="M2.5 3.75l2.5 2.5 2.5-2.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
                        </svg>
                    </button>
                    <span class="stream-subgroup-title">${title}</span>
                    <span class="stream-subgroup-count">(${items.length})</span>
                </div>
                <div class="stream-subgroup-content">
                    ${items.map(({ stream, index }) => this.renderItem(stream, index)).join('')}
                </div>
            </div>
        `;
    }

    renderGroup(title, items, groupClass) {
        const count = items.length;
        const isCollapsed = this.collapsedGroups.has(groupClass);

        return `
            <div class="stream-group stream-group-${groupClass} ${isCollapsed ? 'collapsed' : ''}">
                <div class="stream-group-header" data-group="${groupClass}">
                    <button class="stream-group-toggle" aria-label="Toggle group">
                        <svg width="12" height="12" viewBox="0 0 12 12" fill="none" class="chevron">
                            <path d="M3 4.5l3 3 3-3" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
                        </svg>
                    </button>
                    <span class="stream-group-title">${title}</span>
                    <span class="stream-group-count">(${count})</span>
                </div>
                <div class="stream-group-content">
                    ${items.map(({ stream, index }) => this.renderItem(stream, index)).join('')}
                </div>
            </div>
        `;
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
        // Group header toggle (Recommended, Alternative)
        this.listContainer.querySelectorAll('.stream-group-header').forEach(header => {
            header.addEventListener('click', (e) => {
                const groupKey = header.dataset.group;
                if (groupKey) {
                    this.toggleGroup(groupKey);
                }
            });
        });

        // Subgroup header toggle (Main, Sub, Other)
        this.listContainer.querySelectorAll('.stream-subgroup-header').forEach(header => {
            header.addEventListener('click', (e) => {
                e.stopPropagation(); // Don't bubble to group header
                const subgroupKey = header.dataset.subgroup;
                if (subgroupKey) {
                    this.toggleSubgroup(subgroupKey);
                }
            });
        });

        // Click on stream item header to toggle details
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

    toggleGroup(groupKey) {
        if (this.collapsedGroups.has(groupKey)) {
            this.collapsedGroups.delete(groupKey);
        } else {
            this.collapsedGroups.add(groupKey);
        }
        this.render(this.streams, this.onUseCallback);
    }

    toggleSubgroup(subgroupKey) {
        if (this.collapsedSubgroups.has(subgroupKey)) {
            this.collapsedSubgroups.delete(subgroupKey);
        } else {
            this.collapsedSubgroups.add(subgroupKey);
        }
        this.render(this.streams, this.onUseCallback);
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

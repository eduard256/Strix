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
            'HTTP_VIDEO': '<svg width="20" height="20" viewBox="0 0 20 20" fill="none"><path d="M7 6l6 4-6 4V6z" fill="currentColor"/><circle cx="10" cy="10" r="8" stroke="currentColor" stroke-width="1.5"/></svg>'
        };
        return icons[type] || icons['FFMPEG'];
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

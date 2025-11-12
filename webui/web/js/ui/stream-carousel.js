export class StreamCarousel {
    constructor() {
        this.track = document.getElementById('carousel-track');
        this.prevBtn = document.getElementById('carousel-prev');
        this.nextBtn = document.getElementById('carousel-next');
        this.counter = document.getElementById('carousel-counter');
        this.dotsContainer = document.getElementById('carousel-dots');

        this.streams = [];
        this.currentIndex = 0;
        this.onUseCallback = null;
    }

    render(streams, onUseCallback) {
        this.streams = streams;
        this.onUseCallback = onUseCallback;
        this.currentIndex = Math.min(this.currentIndex, streams.length - 1);

        // Render stream cards
        this.track.innerHTML = streams.map((stream, index) => this.renderCard(stream, index)).join('');

        // Render dots
        this.dotsContainer.innerHTML = streams.map((_, index) =>
            `<button class="carousel-dot ${index === this.currentIndex ? 'active' : ''}"
                     data-index="${index}"
                     aria-label="Go to stream ${index + 1}"></button>`
        ).join('');

        // Attach event listeners
        this.attachEventListeners();

        // Update view
        this.updateView();
    }

    renderCard(stream, index) {
        const icon = this.getStreamIcon(stream.type);

        return `
            <div class="stream-card" data-index="${index}">
                <div class="stream-type">
                    ${icon}
                    ${stream.type}
                </div>
                <div class="stream-url">${this.truncateURL(stream.url)}</div>
                ${stream.resolution ? `<div class="stream-meta">Resolution: ${stream.resolution}</div>` : ''}
                ${stream.codec ? `<div class="stream-meta">Codec: ${stream.codec}${stream.fps ? ` • ${stream.fps} fps` : ''}${stream.bitrate ? ` • ${Math.round(stream.bitrate / 1000)} Kbps` : ''}</div>` : ''}
                ${stream.has_audio ? '<div class="stream-meta">Audio: Yes</div>' : ''}
                <div class="stream-actions">
                    <button class="btn btn-primary btn-use" data-index="${index}">Use Stream</button>
                </div>
            </div>
        `;
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

    truncateURL(url) {
        if (url.length > 50) {
            return url.substring(0, 47) + '...';
        }
        return url;
    }

    attachEventListeners() {
        // Use buttons
        this.track.querySelectorAll('.btn-use').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const index = parseInt(e.target.dataset.index);
                if (this.onUseCallback) {
                    this.onUseCallback(this.streams[index], index);
                }
            });
        });

        // Dots
        this.dotsContainer.querySelectorAll('.carousel-dot').forEach(dot => {
            dot.addEventListener('click', (e) => {
                const index = parseInt(e.target.dataset.index);
                this.goTo(index);
            });
        });

        // Touch gestures
        let touchStartX = 0;
        let touchEndX = 0;

        this.track.addEventListener('touchstart', (e) => {
            touchStartX = e.changedTouches[0].screenX;
        });

        this.track.addEventListener('touchend', (e) => {
            touchEndX = e.changedTouches[0].screenX;
            this.handleSwipe(touchStartX, touchEndX);
        });
    }

    handleSwipe(startX, endX) {
        const swipeThreshold = 50;
        const diff = startX - endX;

        if (Math.abs(diff) > swipeThreshold) {
            if (diff > 0) {
                this.next();
            } else {
                this.prev();
            }
        }
    }

    prev() {
        if (this.currentIndex > 0) {
            this.goTo(this.currentIndex - 1);
        }
    }

    next() {
        if (this.currentIndex < this.streams.length - 1) {
            this.goTo(this.currentIndex + 1);
        }
    }

    goTo(index) {
        if (index < 0 || index >= this.streams.length) return;

        this.currentIndex = index;
        this.updateView();
    }

    updateView() {
        // Update track position
        const offset = -100 * this.currentIndex;
        this.track.style.transform = `translateX(${offset}%)`;

        // Update counter
        this.counter.textContent = `Stream ${this.currentIndex + 1} of ${this.streams.length}`;

        // Update dots
        this.dotsContainer.querySelectorAll('.carousel-dot').forEach((dot, i) => {
            dot.classList.toggle('active', i === this.currentIndex);
        });

        // Update arrow buttons
        this.prevBtn.disabled = this.currentIndex === 0;
        this.nextBtn.disabled = this.currentIndex === this.streams.length - 1;
    }
}

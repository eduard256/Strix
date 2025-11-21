import { MockStreamDiscovery } from '../mock/mock-data.js';

export class StreamDiscoveryAPI {
    constructor(baseURL = null, useMock = false) {
        // Use relative URLs since API and UI are on the same port
        if (!baseURL) {
            this.baseURL = '';
        } else {
            this.baseURL = baseURL;
        }
        this.eventSource = null;
        this.useMock = useMock;
        this.mockAPI = useMock ? new MockStreamDiscovery() : null;
    }

    discover(request, callbacks) {
        this.close();

        // Use mock API if enabled
        if (this.useMock) {
            this.mockAPI.discover(request, callbacks);
            return;
        }

        fetch(`${this.baseURL}api/v1/streams/discover`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Accept': 'text/event-stream',
            },
            body: JSON.stringify(request),
        }).then(response => {
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const reader = response.body.getReader();
            const decoder = new TextDecoder();

            const processStream = ({ done, value }) => {
                if (done) {
                    return;
                }

                const chunk = decoder.decode(value, { stream: true });
                const lines = chunk.split('\n');

                for (const line of lines) {
                    if (line.startsWith('event:')) {
                        // Parse event type (not currently used, but available for future features)
                        // const eventType = line.substring(6).trim();
                        continue;
                    }

                    if (line.startsWith('data:')) {
                        const data = line.substring(5).trim();

                        try {
                            const parsed = JSON.parse(data);
                            this.handleEvent(parsed, callbacks);
                        } catch (e) {
                            console.error('Failed to parse SSE data:', e);
                        }
                    }
                }

                return reader.read().then(processStream);
            };

            return reader.read().then(processStream);
        }).catch(error => {
            if (callbacks.onError) {
                callbacks.onError(error.message);
            }
        });
    }

    handleEvent(data, callbacks) {
        // Determine event type from data
        if (data.tested !== undefined && data.found !== undefined) {
            // Progress event
            if (callbacks.onProgress) {
                callbacks.onProgress(data);
            }
        } else if (data.stream) {
            // Stream found event
            if (callbacks.onStreamFound) {
                callbacks.onStreamFound(data);
            }
        } else if (data.total_tested !== undefined) {
            // Complete event
            if (callbacks.onComplete) {
                callbacks.onComplete(data);
            }
        } else if (data.error) {
            // Error event
            if (callbacks.onError) {
                callbacks.onError(data.error);
            }
        }
    }

    close() {
        if (this.useMock && this.mockAPI) {
            this.mockAPI.close();
        }
        if (this.eventSource) {
            this.eventSource.close();
            this.eventSource = null;
        }
    }
}

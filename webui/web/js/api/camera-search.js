import { MockCameraSearch } from '../mock/mock-data.js';

export class CameraSearchAPI {
    constructor(baseURL = null, useMock = false) {
        // Use relative URLs since API and UI are on the same port
        if (!baseURL) {
            this.baseURL = '';
        } else {
            this.baseURL = baseURL;
        }
        this.useMock = useMock;
        this.mockAPI = useMock ? new MockCameraSearch() : null;
    }

    async search(query, limit = 10) {
        // Use mock API if enabled
        if (this.useMock) {
            return await this.mockAPI.search(query, limit);
        }

        const response = await fetch(`${this.baseURL}api/v1/cameras/search`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ query, limit }),
        });

        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }

        return await response.json();
    }
}

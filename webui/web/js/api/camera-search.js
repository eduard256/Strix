export class CameraSearchAPI {
    constructor(baseURL = null) {
        // Auto-detect API URL based on current host
        if (!baseURL) {
            const currentHost = window.location.hostname;
            this.baseURL = `http://${currentHost}:8080`;
        } else {
            this.baseURL = baseURL;
        }
    }

    async search(query, limit = 10) {
        const response = await fetch(`${this.baseURL}/api/v1/cameras/search`, {
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

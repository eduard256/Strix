export class CameraSearchAPI {
    constructor(baseURL = null) {
        // Use relative URLs since API and UI are on the same port
        if (!baseURL) {
            this.baseURL = '';
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

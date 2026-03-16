export class ProbeAPI {
    constructor(baseURL = '') {
        this.baseURL = baseURL;
    }

    /**
     * Probe a device at the given IP address.
     * Returns device info: reachable status, vendor, hostname, mDNS data.
     * @param {string} ip - IP address to probe
     * @returns {Promise<Object>} Probe response
     */
    async probe(ip) {
        const response = await fetch(
            `${this.baseURL}api/v1/probe?ip=${encodeURIComponent(ip)}`
        );

        if (!response.ok) {
            const text = await response.text();
            throw new Error(text || `HTTP ${response.status}`);
        }

        return await response.json();
    }
}

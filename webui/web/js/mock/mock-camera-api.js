// Mock implementation of CameraSearchAPI for development
export class MockCameraAPI {
    constructor() {
        this.mockCameras = [
            { brand: "Hikvision", model: "DS-2CD2042WD-I" },
            { brand: "Hikvision", model: "DS-2CD2142FWD-I" },
            { brand: "Hikvision", model: "DS-2CD2032-I" },
            { brand: "Hikvision", model: "DS-2CD2385G1-I" },
            { brand: "Dahua", model: "IPC-HFW4431R-Z" },
            { brand: "Dahua", model: "IPC-HDBW4433R-ZS" },
            { brand: "Dahua", model: "DH-IPC-HFW2431S-S-S2" },
            { brand: "Dahua", model: "IPC-HDW2531T-AS-S2" },
            { brand: "Axis", model: "M3046-V" },
            { brand: "Axis", model: "P3245-LVE" },
            { brand: "Axis", model: "M5525-E" },
            { brand: "Uniview", model: "IPC322SR3-DVS28-F" },
            { brand: "Uniview", model: "IPC2124SR3-DPF40" },
            { brand: "Reolink", model: "RLC-410" },
            { brand: "Reolink", model: "RLC-520A" },
            { brand: "Reolink", model: "RLC-810A" },
            { brand: "TP-Link", model: "VIGI C300HP-4" },
            { brand: "TP-Link", model: "VIGI C540V" },
            { brand: "Amcrest", model: "IP8M-2496EW" },
            { brand: "Amcrest", model: "IP4M-1041B" },
            { brand: "Foscam", model: "FI9900P" },
            { brand: "Foscam", model: "R5" },
        ];
    }

    async search(query, limit = 10) {
        // Simulate network delay
        await this.delay(150);

        const lowerQuery = query.toLowerCase();
        const filtered = this.mockCameras.filter(camera => {
            const searchText = `${camera.brand} ${camera.model}`.toLowerCase();
            return searchText.includes(lowerQuery);
        });

        return {
            cameras: filtered.slice(0, limit),
            total: filtered.length
        };
    }

    delay(ms) {
        return new Promise(resolve => setTimeout(resolve, ms));
    }
}

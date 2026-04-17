package probe

type Response struct {
	IP        string  `json:"ip"`
	Reachable bool    `json:"reachable"`
	Type string `json:"type"` // "unreachable", "standard", "homekit"
	Error     string  `json:"error,omitempty"`
	Probes    Probes  `json:"probes"`
}

type Probes struct {
	Ports  *PortsResult  `json:"ports"`
	DNS    *DNSResult    `json:"dns"`
	ARP    *ARPResult    `json:"arp"`
	MDNS   *MDNSResult   `json:"mdns"`
	HTTP   *HTTPResult   `json:"http"`
	ONVIF  *ONVIFResult  `json:"onvif"`
	Xiaomi *XiaomiResult `json:"xiaomi"`
}

type PortsResult struct {
	Open []int `json:"open"`
}

type DNSResult struct {
	Hostname string `json:"hostname"`
}

type ARPResult struct {
	MAC    string `json:"mac"`
	Vendor string `json:"vendor"`
}

type MDNSResult struct {
	Name     string `json:"name"`
	DeviceID string `json:"device_id"`
	Model    string `json:"model"`
	Category string `json:"category"` // "camera", "doorbell"
	Paired   bool   `json:"paired"`
	Port     int    `json:"port"`
}

type HTTPResult struct {
	Port       int    `json:"port"`
	StatusCode int    `json:"status_code"`
	Server     string `json:"server"`
}

type ONVIFResult struct {
	URL      string `json:"url"`
	Port     int    `json:"port"`
	Name     string `json:"name,omitempty"`
	Hardware string `json:"hardware,omitempty"`
}

type XiaomiResult struct {
	DeviceID uint32 `json:"device_id"`
	Stamp    uint32 `json:"stamp"`
}

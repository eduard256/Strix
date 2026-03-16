package models

// ProbeResponse represents the result of probing an IP address.
// The Type field determines which UI flow the frontend should use:
//   - "unreachable" -- device did not respond to ping
//   - "standard"    -- normal IP camera (RTSP/HTTP/ONVIF)
//   - "homekit"     -- Apple HomeKit camera (needs PIN pairing)
type ProbeResponse struct {
	IP        string       `json:"ip"`
	Reachable bool         `json:"reachable"`
	LatencyMs float64      `json:"latency_ms,omitempty"`
	Type      string       `json:"type"`
	Error     string       `json:"error,omitempty"`
	Probes    ProbeResults `json:"probes"`
}

// ProbeResults contains results from all parallel probers.
// Nil fields mean the prober did not find anything or timed out.
type ProbeResults struct {
	DNS  *DNSProbeResult  `json:"dns"`
	ARP  *ARPProbeResult  `json:"arp"`
	MDNS *MDNSProbeResult `json:"mdns"`
}

// DNSProbeResult contains reverse DNS lookup result.
type DNSProbeResult struct {
	Hostname string `json:"hostname"`
}

// ARPProbeResult contains ARP table lookup + OUI vendor identification.
type ARPProbeResult struct {
	MAC    string `json:"mac"`
	Vendor string `json:"vendor"`
}

// MDNSProbeResult contains mDNS service discovery result (HomeKit).
type MDNSProbeResult struct {
	Name     string `json:"name"`
	DeviceID string `json:"device_id"`
	Model    string `json:"model"`
	Category string `json:"category"` // "camera", "doorbell"
	Paired   bool   `json:"paired"`
	Port     int    `json:"port"`
	Feature  string `json:"feature"`
}

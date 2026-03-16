package discovery

import (
	"context"
	"sync"
	"time"

	"github.com/eduard256/Strix/internal/models"
)

const (
	// ProbeTimeout is the overall timeout for all probes combined.
	ProbeTimeout = 3 * time.Second

	// ProbeTypeUnreachable indicates the device did not respond to ping.
	ProbeTypeUnreachable = "unreachable"
	// ProbeTypeStandard indicates a normal IP camera (RTSP/HTTP/ONVIF).
	ProbeTypeStandard = "standard"
	// ProbeTypeHomeKit indicates an Apple HomeKit camera that needs PIN pairing.
	ProbeTypeHomeKit = "homekit"
)

// Prober is an interface for network probe implementations.
// Each prober discovers specific information about a device at a given IP.
// New probers can be added by implementing this interface and registering
// them with ProbeService.
type Prober interface {
	// Name returns a unique identifier for this prober (e.g., "dns", "arp", "mdns").
	Name() string
	// Probe runs the probe against the given IP address.
	// Must respect context cancellation/timeout.
	// Returns nil result if nothing was found (not an error).
	Probe(ctx context.Context, ip string) (any, error)
}

// ProbeService orchestrates multiple probers to gather information about a device.
// It first pings the device, then runs all registered probers in parallel.
type ProbeService struct {
	pinger  *PingProber
	probers []Prober
	logger  interface {
		Debug(string, ...any)
		Error(string, error, ...any)
		Info(string, ...any)
	}
}

// NewProbeService creates a new ProbeService with the given probers.
// The ping prober is always included and runs first.
func NewProbeService(
	probers []Prober,
	logger interface {
		Debug(string, ...any)
		Error(string, error, ...any)
		Info(string, ...any)
	},
) *ProbeService {
	return &ProbeService{
		pinger:  &PingProber{},
		probers: probers,
		logger:  logger,
	}
}

// Probe runs ping + all registered probers against the given IP.
// Overall timeout is 3 seconds. Results are collected from whatever
// finishes in time; slow probers are omitted (nil in response).
func (s *ProbeService) Probe(ctx context.Context, ip string) *models.ProbeResponse {
	ctx, cancel := context.WithTimeout(ctx, ProbeTimeout)
	defer cancel()

	response := &models.ProbeResponse{
		IP:   ip,
		Type: ProbeTypeStandard,
	}

	// Step 1: Ping
	s.logger.Debug("probing device", "ip", ip)

	pingResult, err := s.pinger.Ping(ctx, ip)
	if err != nil || !pingResult.Reachable {
		errMsg := "device unreachable"
		if err != nil {
			errMsg = err.Error()
		}
		s.logger.Debug("ping failed", "ip", ip, "error", errMsg)
		response.Reachable = false
		response.Type = ProbeTypeUnreachable
		response.Error = errMsg
		return response
	}

	response.Reachable = true
	response.LatencyMs = pingResult.LatencyMs
	s.logger.Debug("ping OK", "ip", ip, "latency_ms", pingResult.LatencyMs)

	// Step 2: Run all probers in parallel
	type probeResult struct {
		name string
		data any
		err  error
	}

	results := make(chan probeResult, len(s.probers))
	var wg sync.WaitGroup

	for _, p := range s.probers {
		wg.Add(1)
		go func(prober Prober) {
			defer wg.Done()
			data, err := prober.Probe(ctx, ip)
			results <- probeResult{
				name: prober.Name(),
				data: data,
				err:  err,
			}
		}(p)
	}

	// Close results channel when all probers finish
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	for r := range results {
		if r.err != nil {
			s.logger.Debug("prober failed", "prober", r.name, "error", r.err.Error())
			continue
		}
		if r.data == nil {
			continue
		}

		switch r.name {
		case "dns":
			if v, ok := r.data.(*models.DNSProbeResult); ok {
				response.Probes.DNS = v
			}
		case "arp":
			if v, ok := r.data.(*models.ARPProbeResult); ok {
				response.Probes.ARP = v
			}
		case "mdns":
			if v, ok := r.data.(*models.MDNSProbeResult); ok {
				response.Probes.MDNS = v
			}
		}
	}

	// Step 3: Determine type based on probe results
	response.Type = s.determineType(response)

	s.logger.Info("probe completed",
		"ip", ip,
		"reachable", response.Reachable,
		"type", response.Type,
		"latency_ms", response.LatencyMs,
	)

	return response
}

// determineType decides the device type based on collected probe results.
func (s *ProbeService) determineType(response *models.ProbeResponse) string {
	if !response.Reachable {
		return ProbeTypeUnreachable
	}

	// HomeKit camera that is not yet paired
	if response.Probes.MDNS != nil && !response.Probes.MDNS.Paired {
		category := response.Probes.MDNS.Category
		if category == "camera" || category == "doorbell" {
			return ProbeTypeHomeKit
		}
	}

	return ProbeTypeStandard
}

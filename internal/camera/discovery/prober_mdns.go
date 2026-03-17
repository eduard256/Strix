package discovery

import (
	"context"
	"time"

	"github.com/AlexxIT/go2rtc/pkg/hap"
	"github.com/AlexxIT/go2rtc/pkg/mdns"
	"github.com/eduard256/Strix/internal/models"
)

const (
	// mdnsTimeout is the maximum time to wait for mDNS response.
	// HomeKit devices respond in 2-10ms. If no response in 100ms,
	// the device is definitely not a HomeKit camera.
	// The underlying mdns.Query has a 1s internal timeout, but we
	// cut it short with this context-based wrapper.
	mdnsTimeout = 100 * time.Millisecond
)

// MDNSProber performs mDNS unicast query to detect HomeKit devices.
// It sends a DNS query to ip:5353 for the _hap._tcp.local. service
// and parses TXT records to extract device information.
// Uses a 100ms timeout wrapper around go2rtc's mdns.Query to avoid
// waiting the full 1s on non-HomeKit devices.
type MDNSProber struct{}

func (p *MDNSProber) Name() string { return "mdns" }

// Probe queries the device for HomeKit (HAP) mDNS service.
// Returns nil if the device does not advertise HomeKit or is not a camera/doorbell.
func (p *MDNSProber) Probe(ctx context.Context, ip string) (any, error) {
	// Run mdns.Query in a goroutine with 100ms timeout.
	// mdns.Query has an internal 1s timeout and doesn't accept context,
	// so we wrap it. The background goroutine will clean up on its own
	// after the internal timeout expires (~1s, negligible resource cost).
	type queryResult struct {
		entry *mdns.ServiceEntry
		err   error
	}

	ch := make(chan queryResult, 1)
	go func() {
		entry, err := mdns.Query(ip, mdns.ServiceHAP)
		ch <- queryResult{entry, err}
	}()

	// Wait for result or timeout
	timer := time.NewTimer(mdnsTimeout)
	defer timer.Stop()

	var entry *mdns.ServiceEntry

	select {
	case r := <-ch:
		if r.err != nil || r.entry == nil {
			return nil, nil
		}
		entry = r.entry
	case <-timer.C:
		return nil, nil // No response within 100ms -- not a HomeKit device
	case <-ctx.Done():
		return nil, nil
	}

	// Check if it's complete (has IP, port, and TXT records)
	if !entry.Complete() {
		return nil, nil
	}

	// Check if it's a camera or doorbell
	category := entry.Info[hap.TXTCategory]
	if category != hap.CategoryCamera && category != hap.CategoryDoorbell {
		return nil, nil // Not a camera/doorbell, ignore
	}

	// Map category ID to human-readable name
	categoryName := "camera"
	if category == hap.CategoryDoorbell {
		categoryName = "doorbell"
	}

	// Determine paired status: sf=0 means paired, sf=1 means not paired
	paired := entry.Info[hap.TXTStatusFlags] == hap.StatusPaired

	return &models.MDNSProbeResult{
		Name:     entry.Name,
		DeviceID: entry.Info[hap.TXTDeviceID],
		Model:    entry.Info[hap.TXTModel],
		Category: categoryName,
		Paired:   paired,
		Port:     int(entry.Port),
		Feature:  entry.Info[hap.TXTFeatureFlags],
	}, nil
}

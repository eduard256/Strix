package discovery

import (
	"context"

	"github.com/AlexxIT/go2rtc/pkg/hap"
	"github.com/AlexxIT/go2rtc/pkg/mdns"
	"github.com/eduard256/Strix/internal/models"
)

// MDNSProber performs mDNS unicast query to detect HomeKit devices.
// It sends a DNS query to ip:5353 for the _hap._tcp.local. service
// and parses TXT records to extract device information.
type MDNSProber struct{}

func (p *MDNSProber) Name() string { return "mdns" }

// Probe queries the device for HomeKit (HAP) mDNS service.
// Returns nil if the device does not advertise HomeKit or is not a camera/doorbell.
func (p *MDNSProber) Probe(ctx context.Context, ip string) (any, error) {
	// Unicast mDNS query directly to the device IP.
	// mdns.Query has internal timeouts (~1s), which fits within our 3s budget.
	entry, err := mdns.Query(ip, mdns.ServiceHAP)
	if err != nil || entry == nil {
		return nil, nil // Not a HomeKit device is not an error
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

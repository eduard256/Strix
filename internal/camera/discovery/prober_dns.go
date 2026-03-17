package discovery

import (
	"context"
	"net"
	"strings"

	"github.com/eduard256/Strix/internal/models"
)

// DNSProber performs reverse DNS lookup to find the hostname of a device.
type DNSProber struct{}

func (p *DNSProber) Name() string { return "dns" }

// Probe performs a reverse DNS lookup on the given IP.
// Returns nil if no hostname is found (not an error).
func (p *DNSProber) Probe(ctx context.Context, ip string) (any, error) {
	resolver := net.DefaultResolver

	names, err := resolver.LookupAddr(ctx, ip)
	if err != nil || len(names) == 0 {
		return nil, nil // No hostname found is not an error
	}

	// LookupAddr returns FQDNs with trailing dot, remove it
	hostname := strings.TrimSuffix(names[0], ".")

	if hostname == "" {
		return nil, nil
	}

	return &models.DNSProbeResult{
		Hostname: hostname,
	}, nil
}

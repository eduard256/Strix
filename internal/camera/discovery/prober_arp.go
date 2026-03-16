package discovery

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/eduard256/Strix/internal/models"
)

// ARPProber looks up the MAC address from the system ARP table
// and resolves it to a vendor name using the OUI database.
type ARPProber struct {
	ouiDB *OUIDatabase
}

// NewARPProber creates a new ARP prober with the given OUI database.
func NewARPProber(ouiDB *OUIDatabase) *ARPProber {
	return &ARPProber{ouiDB: ouiDB}
}

func (p *ARPProber) Name() string { return "arp" }

// Probe looks up the MAC address for the given IP in the ARP table.
// Returns nil if the IP is not in the ARP table (e.g., different subnet, VPN).
// This only works on Linux (reads /proc/net/arp).
func (p *ARPProber) Probe(ctx context.Context, ip string) (any, error) {
	mac, err := p.lookupARP(ip)
	if err != nil || mac == "" {
		return nil, nil // Not in ARP table is not an error
	}

	vendor := ""
	if p.ouiDB != nil {
		vendor = p.ouiDB.LookupVendor(mac)
	}

	return &models.ARPProbeResult{
		MAC:    mac,
		Vendor: vendor,
	}, nil
}

// lookupARP reads /proc/net/arp to find the MAC address for the given IP.
//
// Format of /proc/net/arp:
//
//	IP address       HW type     Flags       HW address            Mask     Device
//	192.168.1.1      0x1         0x2         aa:bb:cc:dd:ee:ff     *        eth0
func (p *ARPProber) lookupARP(ip string) (string, error) {
	file, err := os.Open("/proc/net/arp")
	if err != nil {
		return "", fmt.Errorf("failed to open ARP table: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Scan() // Skip header line

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 4 {
			continue
		}

		// fields[0] = IP address, fields[3] = HW address
		if fields[0] == ip {
			mac := fields[3]
			// "00:00:00:00:00:00" means incomplete ARP entry
			if mac == "00:00:00:00:00:00" {
				return "", nil
			}
			return strings.ToUpper(mac), nil
		}
	}

	return "", nil
}

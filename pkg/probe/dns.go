package probe

import (
	"context"
	"net"
	"strings"
)

func ReverseDNS(ctx context.Context, ip string) (*DNSResult, error) {
	names, err := net.DefaultResolver.LookupAddr(ctx, ip)
	if err != nil || len(names) == 0 {
		return nil, nil
	}

	hostname := strings.TrimSuffix(names[0], ".")
	if hostname == "" {
		return nil, nil
	}

	return &DNSResult{Hostname: hostname}, nil
}

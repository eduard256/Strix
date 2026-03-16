package discovery

import (
	"context"
	"fmt"
	"net"
	"time"
)

// PingResult contains the result of a ping probe.
type PingResult struct {
	Reachable bool
	LatencyMs float64
}

// PingProber checks if a device is reachable on the network.
// It tries ICMP ping first (requires root/CAP_NET_RAW), then falls back
// to TCP connect on common camera ports (80, 554, 443, 8080).
type PingProber struct{}

// Ping checks if the device at the given IP is reachable.
func (p *PingProber) Ping(ctx context.Context, ip string) (*PingResult, error) {
	// Try ICMP first (works if running as root or with CAP_NET_RAW)
	result, err := p.tryICMP(ctx, ip)
	if err == nil {
		return result, nil
	}

	// Fallback: TCP connect on common camera ports
	result, err = p.tryTCP(ctx, ip)
	if err == nil {
		return result, nil
	}

	return &PingResult{Reachable: false}, fmt.Errorf("device unreachable: %s", ip)
}

// tryICMP attempts an ICMP ping using raw socket.
func (p *PingProber) tryICMP(ctx context.Context, ip string) (*PingResult, error) {
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(2 * time.Second)
	}

	timeout := time.Until(deadline)
	if timeout <= 0 {
		return nil, context.DeadlineExceeded
	}
	// Cap ICMP timeout to 2 seconds to leave time for other probes
	if timeout > 2*time.Second {
		timeout = 2 * time.Second
	}

	start := time.Now()
	conn, err := net.DialTimeout("ip4:icmp", ip, timeout)
	if err != nil {
		return nil, err
	}
	conn.Close()

	return &PingResult{
		Reachable: true,
		LatencyMs: float64(time.Since(start).Microseconds()) / 1000.0,
	}, nil
}

// tryTCP attempts TCP connect on common camera ports as a ping fallback.
// This works without root privileges and is reliable for cameras since
// they almost always have at least one of these ports open.
func (p *PingProber) tryTCP(ctx context.Context, ip string) (*PingResult, error) {
	commonPorts := []int{80, 554, 443, 8080, 8443, 34567, 5353}

	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(2 * time.Second)
	}

	timeout := time.Until(deadline)
	if timeout <= 0 {
		return nil, context.DeadlineExceeded
	}
	// Cap per-port timeout
	perPortTimeout := timeout / time.Duration(len(commonPorts))
	if perPortTimeout > 500*time.Millisecond {
		perPortTimeout = 500 * time.Millisecond
	}

	type tcpResult struct {
		latency time.Duration
		err     error
	}

	results := make(chan tcpResult, len(commonPorts))

	for _, port := range commonPorts {
		go func(port int) {
			addr := fmt.Sprintf("%s:%d", ip, port)
			start := time.Now()
			conn, err := net.DialTimeout("tcp", addr, perPortTimeout)
			if err != nil {
				results <- tcpResult{err: err}
				return
			}
			conn.Close()
			results <- tcpResult{latency: time.Since(start)}
		}(port)
	}

	// Wait for first success or all failures
	var lastErr error
	for range commonPorts {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case r := <-results:
			if r.err == nil {
				return &PingResult{
					Reachable: true,
					LatencyMs: float64(r.latency.Microseconds()) / 1000.0,
				}, nil
			}
			lastErr = r.err
		}
	}

	return nil, fmt.Errorf("all TCP ports closed: %w", lastErr)
}

package probe

import (
	"context"
	"net"
	"time"
)

func CanICMP() bool {
	conn, err := net.DialTimeout("ip4:icmp", "127.0.0.1", 100*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func Ping(ctx context.Context, ip string) (*PingResult, error) {
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(100 * time.Millisecond)
	}

	timeout := time.Until(deadline)
	if timeout <= 0 {
		return nil, context.DeadlineExceeded
	}

	start := time.Now()
	conn, err := net.DialTimeout("ip4:icmp", ip, timeout)
	if err != nil {
		return nil, err
	}
	conn.Close()

	return &PingResult{
		LatencyMs: float64(time.Since(start).Microseconds()) / 1000.0,
	}, nil
}

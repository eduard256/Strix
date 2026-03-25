package probe

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"
)

func ScanPorts(ctx context.Context, ip string, ports []int) (*PortsResult, error) {
	if len(ports) == 0 {
		return nil, nil
	}

	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(100 * time.Millisecond)
	}
	timeout := time.Until(deadline)
	if timeout <= 0 {
		return nil, context.DeadlineExceeded
	}

	type hit struct {
		port    int
		latency time.Duration
	}

	var mu sync.Mutex
	var hits []hit
	var wg sync.WaitGroup

	for _, port := range ports {
		wg.Add(1)
		go func(port int) {
			defer wg.Done()

			start := time.Now()
			conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", ip, port), timeout)
			if err != nil {
				return
			}
			conn.Close()

			mu.Lock()
			hits = append(hits, hit{port: port, latency: time.Since(start)})
			mu.Unlock()
		}(port)
	}

	wg.Wait()

	if len(hits) == 0 {
		return nil, nil
	}

	open := make([]int, len(hits))
	for i, h := range hits {
		open[i] = h.port
	}

	return &PortsResult{Open: open}, nil
}

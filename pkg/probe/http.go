package probe

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
)

func ProbeHTTP(ctx context.Context, ip string, ports []int) (*HTTPResult, error) {
	if len(ports) == 0 {
		ports = []int{80, 8080}
	}

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	type result struct {
		resp *http.Response
		port int
	}

	ch := make(chan result, len(ports))

	for _, port := range ports {
		go func(port int) {
			url := fmt.Sprintf("http://%s:%d/", ip, port)
			req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
			if err != nil {
				return
			}
			req.Header.Set("User-Agent", "Strix/2.0")

			resp, err := client.Do(req)
			if err != nil {
				return
			}
			ch <- result{resp: resp, port: port}
		}(port)
	}

	for range ports {
		select {
		case <-ctx.Done():
			return nil, nil
		case r := <-ch:
			if r.resp.Body != nil {
				r.resp.Body.Close()
			}
			return &HTTPResult{
				Port:       r.port,
				StatusCode: r.resp.StatusCode,
				Server:     r.resp.Header.Get("Server"),
			}, nil
		}
	}

	return nil, nil
}

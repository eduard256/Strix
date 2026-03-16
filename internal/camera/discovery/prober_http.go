package discovery

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"

	"github.com/eduard256/Strix/internal/models"
)

// HTTPProber identifies the device by checking HTTP server headers.
// It sends HEAD and GET requests in parallel to port 80 (some devices
// like XMEye/JAWS don't respond to HEAD), and returns whichever
// responds first.
type HTTPProber struct{}

func (p *HTTPProber) Name() string { return "http" }

// Probe sends parallel HEAD+GET to port 80 and extracts Server header.
// Returns nil if no HTTP server is found.
func (p *HTTPProber) Probe(ctx context.Context, ip string) (any, error) {
	ports := []int{80, 8080}

	client := &http.Client{
		// Don't follow redirects -- we want the original response headers
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
		err  error
	}

	for _, port := range ports {
		url := fmt.Sprintf("http://%s:%d/", ip, port)
		ch := make(chan result, 2)

		// HEAD and GET in parallel -- take whichever responds first
		for _, method := range []string{"HEAD", "GET"} {
			go func(method string) {
				req, err := http.NewRequestWithContext(ctx, method, url, nil)
				if err != nil {
					ch <- result{err: err}
					return
				}
				req.Header.Set("User-Agent", "Strix/1.0")
				resp, err := client.Do(req)
				ch <- result{resp: resp, port: port, err: err}
			}(method)
		}

		// Wait for first success
		for i := 0; i < 2; i++ {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case r := <-ch:
				if r.err != nil {
					continue
				}
				if r.resp.Body != nil {
					r.resp.Body.Close()
				}

				server := r.resp.Header.Get("Server")
				if server == "" && r.resp.StatusCode == 0 {
					continue
				}

				return &models.HTTPProbeResult{
					Port:       r.port,
					StatusCode: r.resp.StatusCode,
					Server:     server,
				}, nil
			}
		}
	}

	return nil, nil
}

package tester

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/AlexxIT/go2rtc/pkg/core"
	"github.com/AlexxIT/go2rtc/pkg/hls"
	"github.com/AlexxIT/go2rtc/pkg/image"
	"github.com/AlexxIT/go2rtc/pkg/magic"
	"github.com/AlexxIT/go2rtc/pkg/mpjpeg"
	"github.com/AlexxIT/go2rtc/pkg/tcp"
)

func init() {
	RegisterSource("http", httpHandler)
	RegisterSource("https", httpHandler)
	RegisterSource("httpx", httpHandler)
}

// httpHandler -- HTTP GET with content-type detection.
// Supports JPEG snapshots, MJPEG streams, HLS, MPEG-TS, and auto-detect via magic.Open.
// Uses go2rtc tcp.Do for Basic + Digest auth and TLS handling.
// ex. "http://admin:pass@192.168.1.100/cgi-bin/snapshot.cgi"
func httpHandler(rawURL string) (core.Producer, error) {
	rawURL, _, _ = strings.Cut(rawURL, "#")

	// httpx -> https with insecure TLS (handled inside tcp.Do)
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("http: request: %w", err)
	}

	res, err := tcp.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http: dial: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		tcp.Close(res)
		return nil, errors.New("http: " + res.Status)
	}

	ct := res.Header.Get("Content-Type")
	if i := strings.IndexByte(ct, ';'); i > 0 {
		ct = ct[:i]
	}

	var ext string
	if i := strings.LastIndexByte(req.URL.Path, '.'); i > 0 {
		ext = req.URL.Path[i+1:]
	}

	switch {
	case ct == "application/vnd.apple.mpegurl" || ext == "m3u8":
		return hls.OpenURL(req.URL, res.Body)
	case ct == "image/jpeg":
		return image.Open(res)
	case ct == "multipart/x-mixed-replace":
		return mpjpeg.Open(res.Body)
	}

	return magic.Open(res.Body)
}

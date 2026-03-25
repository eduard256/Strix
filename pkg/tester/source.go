package tester

import (
	"fmt"
	"strings"

	"github.com/AlexxIT/go2rtc/pkg/core"
	"github.com/AlexxIT/go2rtc/pkg/rtsp"
)

// SourceHandler tests stream URL, returns Producer or error
type SourceHandler func(rawURL string) (core.Producer, error)

var handlers = map[string]SourceHandler{}

func RegisterSource(scheme string, handler SourceHandler) {
	handlers[scheme] = handler
}

func GetHandler(rawURL string) SourceHandler {
	if i := strings.IndexByte(rawURL, ':'); i > 0 {
		return handlers[rawURL[:i]]
	}
	return nil
}

func init() {
	RegisterSource("rtsp", rtspHandler)
	RegisterSource("rtsps", rtspHandler)
	RegisterSource("rtspx", rtspHandler)
}

// rtspHandler -- Dial + Describe. Proves: port open, RTSP responds, auth OK, SDP received.
func rtspHandler(rawURL string) (core.Producer, error) {
	rawURL, _, _ = strings.Cut(rawURL, "#")

	conn := rtsp.NewClient(rawURL)
	conn.Backchannel = false

	if err := conn.Dial(); err != nil {
		return nil, fmt.Errorf("rtsp: dial: %w", err)
	}

	if err := conn.Describe(); err != nil {
		_ = conn.Stop()
		return nil, fmt.Errorf("rtsp: describe: %w", err)
	}

	return conn, nil
}

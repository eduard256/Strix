package tester

import (
	"fmt"
	"strings"

	"github.com/AlexxIT/go2rtc/pkg/bubble"
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
	RegisterSource("bubble", bubbleHandler)
}

// bubbleHandler -- Dial handles TCP connect, HTTP handshake, XML parsing, and auth.
// ex. "bubble://admin:pass@192.168.1.100:80/bubble/live?ch=0&stream=0"
func bubbleHandler(rawURL string) (core.Producer, error) {
	return bubble.Dial(rawURL)
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

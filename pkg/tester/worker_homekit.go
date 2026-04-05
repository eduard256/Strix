package tester

import (
	"fmt"
	"time"

	"github.com/AlexxIT/go2rtc/pkg/hap"
)

// testHomeKit -- snapshot via HAP GetImage, bypasses SRTP/Producer flow
func testHomeKit(s *Session, rawURL string) {
	start := time.Now()

	conn, err := hap.Dial(rawURL)
	if err != nil {
		return
	}
	defer conn.Close()

	jpeg, err := conn.GetImage(1920, 1080)
	if err != nil {
		return
	}

	latency := time.Since(start).Milliseconds()

	r := &Result{
		Source:    rawURL,
		Codecs:   []string{"JPEG"},
		LatencyMs: latency,
	}

	if len(jpeg) > 0 {
		idx := s.AddScreenshot(jpeg)
		r.Screenshot = fmt.Sprintf("api/test/screenshot?id=%s&i=%d", s.ID, idx)
		r.Width, r.Height = jpegSize(jpeg)
	}

	s.AddResult(r)
}

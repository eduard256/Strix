package tester

import (
	"fmt"
	"time"

	"github.com/AlexxIT/go2rtc/pkg/core"
	"github.com/AlexxIT/go2rtc/pkg/onvif"
)

// testOnvif resolves all ONVIF profiles, tests each via RTSP,
// and adds two Results per profile (onvif:// + rtsp://).
// ex. "onvif://admin:pass@10.0.20.111" or "onvif://admin:pass@10.0.20.119:2020"
func testOnvif(s *Session, rawURL string) {
	client, err := onvif.NewClient(rawURL)
	if err != nil {
		return
	}

	tokens, err := client.GetProfilesTokens()
	if err != nil {
		return
	}

	for _, token := range tokens {
		profileURL := rawURL + "?subtype=" + token

		pc, err := onvif.NewClient(profileURL)
		if err != nil {
			continue
		}

		rtspURI, err := pc.GetURI()
		if err != nil {
			continue
		}

		testOnvifProfile(s, profileURL, rtspURI)
	}
}

// testOnvifProfile tests a single RTSP stream and adds two Results (onvif + rtsp)
func testOnvifProfile(s *Session, onvifURL, rtspURL string) {
	start := time.Now()

	prod, err := rtspHandler(rtspURL)
	if err != nil {
		return
	}
	defer func() { _ = prod.Stop() }()

	latency := time.Since(start).Milliseconds()

	var codecs []string
	for _, media := range prod.GetMedias() {
		if media.Direction != core.DirectionRecvonly {
			continue
		}
		for _, codec := range media.Codecs {
			codecs = append(codecs, codec.Name)
		}
	}

	// capture screenshot
	var screenshotPath string
	var width, height int

	if raw, codecName := getScreenshot(prod); raw != nil {
		var jpeg []byte

		switch codecName {
		case core.CodecH264, core.CodecH265:
			jpeg = toJPEG(raw)
		default:
			jpeg = raw
		}

		if jpeg != nil {
			idx := s.AddScreenshot(jpeg)
			screenshotPath = fmt.Sprintf("api/test/screenshot?id=%s&i=%d", s.ID, idx)
			width, height = jpegSize(jpeg)
		}
	}

	// add onvif:// result
	s.AddResult(&Result{
		Source:     onvifURL,
		Screenshot: screenshotPath,
		Codecs:     codecs,
		Width:      width,
		Height:     height,
		LatencyMs:  latency,
	})

	// add rtsp:// result (same screenshot, same codecs)
	s.AddResult(&Result{
		Source:     rtspURL,
		Screenshot: screenshotPath,
		Codecs:     codecs,
		Width:      width,
		Height:     height,
		LatencyMs:  latency,
	})
}

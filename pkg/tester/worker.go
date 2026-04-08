package tester

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/AlexxIT/go2rtc/pkg/core"
	"github.com/AlexxIT/go2rtc/pkg/magic"
)

const workers = 20

func RunWorkers(s *Session, urls []string) {
	ch := make(chan string, len(urls))
	for _, u := range urls {
		ch <- u
	}
	close(ch)

	done := make(chan struct{})

	n := workers
	if len(urls) < n {
		n = len(urls)
	}

	for i := 0; i < n; i++ {
		go func() {
			for rawURL := range ch {
				select {
				case <-s.Cancelled():
					return
				default:
				}
				testURL(s, rawURL)
			}
			done <- struct{}{}
		}()
	}

	for i := 0; i < n; i++ {
		<-done
	}

	s.Done()
}

func testURL(s *Session, rawURL string) {
	defer s.AddTested()

	if strings.HasPrefix(rawURL, "homekit://") {
		testHomeKit(s, rawURL)
		return
	}

	if strings.HasPrefix(rawURL, "onvif://") {
		testOnvif(s, rawURL)
		return
	}

	handler := GetHandler(rawURL)
	if handler == nil {
		return
	}

	start := time.Now()

	prod, err := handler(rawURL)
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

	r := &Result{
		Source:    rawURL,
		Codecs:   codecs,
		LatencyMs: latency,
	}

	if raw, codecName := getScreenshot(prod); raw != nil {
		var jpeg []byte

		switch codecName {
		case core.CodecH264, core.CodecH265:
			jpeg = toJPEG(raw)
		case core.CodecJPEG:
			jpeg = raw
		default:
			jpeg = raw
		}

		if jpeg != nil {
			idx := s.AddScreenshot(jpeg)
			r.Screenshot = fmt.Sprintf("api/test/screenshot?id=%s&i=%d", s.ID, idx)
			r.Width, r.Height = jpegSize(jpeg)
		}
	}

	s.AddResult(r)
}

// getScreenshot connects Keyframe consumer to producer, waits for first keyframe with 10s timeout
func getScreenshot(prod core.Producer) ([]byte, string) {
	cons := magic.NewKeyframe()

	for _, prodMedia := range prod.GetMedias() {
		if prodMedia.Kind != core.KindVideo || prodMedia.Direction != core.DirectionRecvonly {
			continue
		}
		for _, consMedia := range cons.GetMedias() {
			prodCodec, consCodec := prodMedia.MatchMedia(consMedia)
			if prodCodec == nil {
				continue
			}

			track, err := prod.GetTrack(prodMedia, prodCodec)
			if err != nil {
				continue
			}

			if err = cons.AddTrack(consMedia, consCodec, track); err != nil {
				continue
			}

			goto matched
		}
	}

	return nil, ""

matched:
	go func() {
		_ = prod.Start()
	}()

	once := &core.OnceBuffer{}
	done := make(chan struct{})
	go func() {
		_, _ = cons.WriteTo(once)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		_ = prod.Stop()
		return nil, ""
	}

	return once.Buffer(), cons.CodecName()
}

// jpegSize extracts width and height from JPEG SOF0/SOF2 marker
func jpegSize(data []byte) (int, int) {
	for i := 2; i < len(data)-9; {
		if data[i] != 0xFF {
			return 0, 0
		}
		marker := data[i+1]
		size := int(data[i+2])<<8 | int(data[i+3])

		// SOF0 (0xC0) or SOF2 (0xC2) -- baseline or progressive
		if marker == 0xC0 || marker == 0xC2 {
			h := int(data[i+5])<<8 | int(data[i+6])
			w := int(data[i+7])<<8 | int(data[i+8])
			return w, h
		}

		i += 2 + size
	}
	return 0, 0
}

func toJPEG(raw []byte) []byte {
	cmd := exec.Command("ffmpeg",
		"-hide_banner", "-loglevel", "error",
		"-i", "-",
		"-frames:v", "1",
		"-f", "image2", "-c:v", "mjpeg",
		"-",
	)
	cmd.Stdin = bytes.NewReader(raw)

	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	return out
}

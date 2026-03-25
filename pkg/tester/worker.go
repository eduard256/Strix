package tester

import (
	"bytes"
	"fmt"
	"os/exec"
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
			r.Screenshot = fmt.Sprintf("/api/test/screenshot?id=%s&i=%d", s.ID, idx)
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

package generate

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

var needMP4 = map[string]bool{"bubble": true}

var reIPv4 = regexp.MustCompile(`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`)

func Generate(req *Request) (*Response, error) {
	if req.MainStream == "" {
		return nil, fmt.Errorf("generate: mainStream required")
	}

	info := buildInfo(req)

	if len(req.Objects) > 0 && (req.Detect == nil || !req.Detect.Enabled) {
		if req.Detect == nil {
			req.Detect = &DetectConfig{Enabled: true}
		} else {
			req.Detect.Enabled = true
		}
	}

	existing := strings.TrimSpace(req.ExistingConfig)

	// generate from scratch if no config or config has no go2rtc streams section
	if existing == "" || !strings.Contains(existing, "go2rtc:") {
		config := newConfig(info, req)
		lines := strings.Count(config, "\n") + 1
		added := make([]int, lines)
		for i := range added {
			added[i] = i + 1
		}
		return &Response{Config: config, Added: added}, nil
	}

	return addToConfig(req.ExistingConfig, info, req)
}

func buildInfo(req *Request) *cameraInfo {
	mainScheme := urlScheme(req.MainStream)
	ip := extractIP(req.MainStream)
	sanitized := strings.NewReplacer(".", "_", ":", "_").Replace(ip)

	base := "camera"
	streamBase := "stream"
	if ip != "" {
		base = "camera_" + sanitized
		streamBase = sanitized
	}

	info := &cameraInfo{
		CameraName:     base,
		MainStreamName: streamBase + "_main",
		MainSource:     req.MainStream,
	}

	if req.Name != "" {
		info.CameraName = req.Name
		info.MainStreamName = req.Name + "_main"
	}

	if req.Go2RTC != nil {
		if req.Go2RTC.MainStreamName != "" {
			info.MainStreamName = req.Go2RTC.MainStreamName
		}
		if req.Go2RTC.MainStreamSource != "" {
			info.MainSource = req.Go2RTC.MainStreamSource
		}
	}

	info.MainPath = "rtsp://127.0.0.1:8554/" + info.MainStreamName
	if needMP4[mainScheme] {
		info.MainPath += "?mp4"
	}
	info.MainInputArgs = "preset-rtsp-restream"

	if req.Frigate != nil {
		if req.Frigate.MainStreamPath != "" {
			info.MainPath = req.Frigate.MainStreamPath
		}
		if req.Frigate.MainStreamInputArgs != "" {
			info.MainInputArgs = req.Frigate.MainStreamInputArgs
		}
	}

	if req.SubStream != "" {
		subScheme := urlScheme(req.SubStream)
		subName := streamBase + "_sub"
		if req.Name != "" {
			subName = req.Name + "_sub"
		}
		subSource := req.SubStream
		subPath := "rtsp://127.0.0.1:8554/" + subName
		if needMP4[subScheme] {
			subPath += "?mp4"
		}
		subInputArgs := "preset-rtsp-restream"

		if req.Go2RTC != nil {
			if req.Go2RTC.SubStreamName != "" {
				subName = req.Go2RTC.SubStreamName
			}
			if req.Go2RTC.SubStreamSource != "" {
				subSource = req.Go2RTC.SubStreamSource
			}
		}
		if req.Frigate != nil {
			if req.Frigate.SubStreamPath != "" {
				subPath = req.Frigate.SubStreamPath
			}
			if req.Frigate.SubStreamInputArgs != "" {
				subInputArgs = req.Frigate.SubStreamInputArgs
			}
		}

		info.SubStreamName = subName
		info.SubSource = subSource
		info.SubPath = subPath
		info.SubInputArgs = subInputArgs
	}

	return info
}

func newConfig(info *cameraInfo, req *Request) string {
	var b strings.Builder

	b.WriteString("mqtt:\n  enabled: false\n\n")
	b.WriteString("record:\n  enabled: true\n\n")

	b.WriteString("go2rtc:\n  streams:\n")
	writeStreamLines(&b, info)

	b.WriteString("cameras:\n")
	writeCameraBlock(&b, info, req)

	b.WriteString("version: 0.17-0\n")
	return b.String()
}

// internals

type cameraInfo struct {
	CameraName     string
	MainStreamName string
	MainSource     string
	MainPath       string
	MainInputArgs  string
	SubStreamName  string
	SubSource      string
	SubPath        string
	SubInputArgs   string
}

func urlScheme(rawURL string) string {
	if i := strings.IndexByte(rawURL, ':'); i > 0 {
		return rawURL[:i]
	}
	return ""
}

func extractIP(rawURL string) string {
	if u, err := url.Parse(rawURL); err == nil && u.Hostname() != "" {
		return u.Hostname()
	}
	if m := reIPv4.FindString(rawURL); m != "" {
		return m
	}
	return ""
}

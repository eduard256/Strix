package go2rtc

import (
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/eduard256/strix/internal/api"
	"github.com/eduard256/strix/internal/app"
	"github.com/rs/zerolog"
)

var log zerolog.Logger

var go2rtcURL string
var go2rtcOnce sync.Once

var candidates = []string{
	"http://localhost:1984",
	"http://localhost:11984",
}

const probeTimeout = 50 * time.Millisecond
const requestTimeout = 5 * time.Second

func Init() {
	log = app.GetLogger("go2rtc")

	if url := app.Env("STRIX_GO2RTC_URL", ""); url != "" {
		go2rtcURL = url
		log.Info().Str("url", go2rtcURL).Msg("[go2rtc] using STRIX_GO2RTC_URL")
	}

	api.HandleFunc("api/go2rtc/streams", apiStreams)
}

func getURL() string {
	if go2rtcURL != "" {
		return go2rtcURL
	}

	go2rtcOnce.Do(func() {
		go2rtcURL = probe()
		if go2rtcURL != "" {
			log.Info().Str("url", go2rtcURL).Msg("[go2rtc] discovered")
		}
	})

	return go2rtcURL
}

func probe() string {
	client := &http.Client{Timeout: probeTimeout}

	for _, url := range candidates {
		resp, err := client.Get(url + "/api")
		if err != nil {
			continue
		}
		resp.Body.Close()
		if resp.StatusCode == 200 {
			return url
		}
	}

	return ""
}

// PUT /api/go2rtc/streams?name=...&src=... -- proxy to go2rtc
func apiStreams(w http.ResponseWriter, r *http.Request) {
	if r.Method != "PUT" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	url := getURL()
	if url == "" {
		api.ResponseJSON(w, map[string]any{"success": false, "error": "go2rtc not found"})
		return
	}

	// forward query params as-is
	target := url + "/api/streams?" + r.URL.RawQuery

	client := &http.Client{Timeout: requestTimeout}
	req, err := http.NewRequest("PUT", target, nil)
	if err != nil {
		api.ResponseJSON(w, map[string]any{"success": false, "error": err.Error()})
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		api.ResponseJSON(w, map[string]any{"success": false, "error": err.Error()})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	w.Header().Set("Content-Type", "application/json")
	if resp.StatusCode == 200 {
		api.ResponseJSON(w, map[string]any{"success": true})
	} else {
		api.ResponseJSON(w, map[string]any{"success": false, "error": string(body)})
	}
}

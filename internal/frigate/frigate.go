package frigate

import (
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/eduard256/strix/internal/api"
	"github.com/eduard256/strix/internal/app"
	"github.com/rs/zerolog"
)

var log zerolog.Logger

// resolved Frigate URL, cached after first successful probe
var frigateURL string
var frigateOnce sync.Once

// candidates to try when no explicit URL is set
var candidates = []string{
	"http://localhost:5000",
	"http://ccab4aaf-frigate:5000",
}

const probeTimeout = 50 * time.Millisecond
const requestTimeout = 5 * time.Second

func Init() {
	log = app.GetLogger("frigate")

	if url := app.Env("STRIX_FRIGATE_URL", ""); url != "" {
		frigateURL = url
		log.Info().Str("url", frigateURL).Msg("[frigate] using STRIX_FRIGATE_URL")
	}

	api.HandleFunc("api/frigate/config", apiConfig)
	api.HandleFunc("api/frigate/config/save", apiConfigSave)
}

// getFrigateURL returns resolved Frigate URL. Probes candidates on first call.
func getFrigateURL() string {
	if frigateURL != "" {
		return frigateURL
	}

	frigateOnce.Do(func() {
		frigateURL = probeFrigate()
		if frigateURL != "" {
			log.Info().Str("url", frigateURL).Msg("[frigate] discovered")
		} else {
			log.Warn().Msg("[frigate] not found on any candidate")
		}
	})

	return frigateURL
}

// probeFrigate tries candidates sequentially with short timeout, returns first that responds
func probeFrigate() string {
	client := &http.Client{Timeout: probeTimeout}

	for _, url := range candidates {
		resp, err := client.Get(url + "/api/config")
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

// GET /api/frigate/config -- proxy Frigate config
func apiConfig(w http.ResponseWriter, r *http.Request) {
	url := getFrigateURL()
	if url == "" {
		api.ResponseJSON(w, map[string]any{
			"connected": false,
			"config":    "",
		})
		return
	}

	client := &http.Client{Timeout: requestTimeout}
	resp, err := client.Get(url + "/api/config/raw")
	if err != nil {
		api.ResponseJSON(w, map[string]any{
			"connected": false,
			"error":     err.Error(),
			"config":    "",
		})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// Frigate /api/config/raw returns JSON-encoded string, unquote it
	config := string(body)
	var unquoted string
	if err := json.Unmarshal(body, &unquoted); err == nil {
		config = unquoted
	}

	api.ResponseJSON(w, map[string]any{
		"connected": true,
		"url":       url,
		"config":    config,
	})
}

// POST /api/frigate/config/save?save_option=restart -- proxy config save to Frigate
func apiConfigSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	url := getFrigateURL()
	if url == "" {
		http.Error(w, "frigate not connected", http.StatusBadGateway)
		return
	}

	saveOption := r.URL.Query().Get("save_option")
	if saveOption == "" {
		saveOption = "saveonly"
	}

	client := &http.Client{Timeout: 30 * time.Second}

	req, err := http.NewRequest("POST", url+"/api/config/save?save_option="+saveOption, r.Body)
	if err != nil {
		api.Error(w, err, http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-Type", "text/plain")

	resp, err := client.Do(req)
	if err != nil {
		api.Error(w, err, http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(body)
}

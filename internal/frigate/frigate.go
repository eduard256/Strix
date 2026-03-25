package frigate

import (
	"net/http"
	"os"
	"time"

	"github.com/eduard256/strix/internal/api"
	"github.com/eduard256/strix/internal/app"
	"github.com/rs/zerolog"
)

var log zerolog.Logger
var frigateURL string

func Init() {
	log = app.GetLogger("frigate")

	frigateURL = detectFrigateURL()

	log.Info().Str("url", frigateURL).Msg("[frigate] target")

	api.HandleFunc("api/frigate/check", apiCheck)
}

func apiCheck(w http.ResponseWriter, r *http.Request) {
	client := &http.Client{Timeout: 3 * time.Second}

	result := map[string]any{
		"url":       frigateURL,
		"detection": detectMethod(),
	}

	resp, err := client.Get(frigateURL + "/api/config")
	if err != nil {
		result["connected"] = false
		result["error"] = err.Error()
		api.ResponseJSON(w, result)
		return
	}
	resp.Body.Close()

	result["connected"] = true
	result["status_code"] = resp.StatusCode
	api.ResponseJSON(w, result)
}

// detectFrigateURL determines Frigate URL:
// 1. STRIX_FRIGATE_URL env
// 2. HA addon -- known hostname ccab4aaf-frigate:5000
// 3. fallback localhost:5000
func detectFrigateURL() string {
	if url := os.Getenv("STRIX_FRIGATE_URL"); url != "" {
		return url
	}

	if os.Getenv("SUPERVISOR_TOKEN") != "" {
		return "http://ccab4aaf-frigate:5000"
	}

	return "http://localhost:5000"
}

func detectMethod() string {
	if os.Getenv("STRIX_FRIGATE_URL") != "" {
		return "env"
	}
	if os.Getenv("SUPERVISOR_TOKEN") != "" {
		return "ha"
	}
	return "localhost"
}

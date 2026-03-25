package frigate

import (
	"fmt"
	"net/http"
	"time"

	"github.com/eduard256/strix/internal/api"
	"github.com/eduard256/strix/internal/app"
	"github.com/rs/zerolog"
)

var log zerolog.Logger
var frigateURL string

func Init() {
	log = app.GetLogger("frigate")

	frigateURL = app.Env("STRIX_FRIGATE_URL", "http://localhost:5000")

	log.Info().Str("url", frigateURL).Msg("[frigate] target")

	api.HandleFunc("api/frigate/check", apiCheck)
}

func apiCheck(w http.ResponseWriter, r *http.Request) {
	client := &http.Client{Timeout: 3 * time.Second}

	resp, err := client.Get(frigateURL + "/api/config")
	if err != nil {
		api.ResponseJSON(w, map[string]any{
			"connected": false,
			"url":       frigateURL,
			"error":     err.Error(),
		})
		return
	}
	resp.Body.Close()

	api.ResponseJSON(w, map[string]any{
		"connected":   true,
		"url":         frigateURL,
		"status_code": resp.StatusCode,
		"version":     resp.Header.Get("X-Frigate-Version"),
		"message":     fmt.Sprintf("Frigate API responded with %d", resp.StatusCode),
	})
}

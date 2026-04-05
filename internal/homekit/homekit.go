package homekit

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/AlexxIT/go2rtc/pkg/hap"
	"github.com/eduard256/strix/internal/api"
	"github.com/eduard256/strix/internal/app"
	"github.com/rs/zerolog"
)

var log zerolog.Logger

func Init() {
	log = app.GetLogger("homekit")

	api.HandleFunc("api/homekit/pair", apiPair)
}

func apiPair(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		IP       string `json:"ip"`
		Port     int    `json:"port"`
		DeviceID string `json:"device_id"`
		PIN      string `json:"pin"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.IP == "" || req.Port == 0 || req.DeviceID == "" || req.PIN == "" {
		http.Error(w, "ip, port, device_id and pin required", http.StatusBadRequest)
		return
	}

	// ex. "homekit://10.0.10.52:45959?device_id=90:8C:0F:F2:EC:F3&pin=12345678"
	rawURL := fmt.Sprintf("homekit://%s:%d?device_id=%s&pin=%s", req.IP, req.Port, req.DeviceID, req.PIN)

	log.Debug().Str("ip", req.IP).Int("port", req.Port).Str("device_id", req.DeviceID).Msg("[homekit] pair")

	conn, err := hap.Pair(rawURL)
	if err != nil {
		log.Warn().Err(err).Str("ip", req.IP).Msg("[homekit] pair failed")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	url := conn.URL()
	log.Info().Str("ip", req.IP).Str("device_id", req.DeviceID).Msg("[homekit] paired")

	api.ResponseJSON(w, map[string]string{"url": url})
}

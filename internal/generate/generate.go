package generate

import (
	"encoding/json"
	"net/http"

	"github.com/eduard256/strix/internal/api"
	"github.com/eduard256/strix/internal/app"
	gen "github.com/eduard256/strix/pkg/generate"
	"github.com/rs/zerolog"
)

var log zerolog.Logger

func Init() {
	log = app.GetLogger("generate")

	api.HandleFunc("api/generate", apiGenerate)
}

func apiGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req gen.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json: "+err.Error(), http.StatusBadRequest)
		return
	}

	resp, err := gen.Generate(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	api.ResponseJSON(w, resp)
}

package test

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/eduard256/strix/internal/api"
	"github.com/eduard256/strix/internal/app"
	"github.com/eduard256/strix/pkg/tester"
	"github.com/rs/zerolog"
)

var log zerolog.Logger

var sessions = map[string]*tester.Session{}
var sessionsMu sync.Mutex

func Init() {
	log = app.GetLogger("test")

	api.HandleFunc("api/test", apiTest)
	api.HandleFunc("api/test/screenshot", apiScreenshot)

	// cleanup expired sessions
	go func() {
		for {
			time.Sleep(time.Minute)

			sessionsMu.Lock()
			for id, s := range sessions {
				s.Lock()
				expired := s.Status == "done" && time.Since(s.ExpiresAt) > 0
				s.Unlock()
				if expired {
					delete(sessions, id)
				}
			}
			sessionsMu.Unlock()
		}
	}()
}

func apiTest(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		id := r.URL.Query().Get("id")
		if id == "" {
			apiTestList(w)
			return
		}
		apiTestGet(w, id)

	case "POST":
		apiTestCreate(w, r)

	case "DELETE":
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "id required", http.StatusBadRequest)
			return
		}
		apiTestDelete(w, id)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func apiTestList(w http.ResponseWriter) {
	type summary struct {
		ID         string `json:"session_id"`
		Status     string `json:"status"`
		Total      int    `json:"total"`
		Tested     int    `json:"tested"`
		Alive      int    `json:"alive"`
		WithScreen int    `json:"with_screenshot"`
	}

	sessionsMu.Lock()
	items := make([]summary, 0, len(sessions))
	for _, s := range sessions {
		s.Lock()
		items = append(items, summary{
			ID:         s.ID,
			Status:     s.Status,
			Total:      s.Total,
			Tested:     s.Tested,
			Alive:      s.Alive,
			WithScreen: s.WithScreen,
		})
		s.Unlock()
	}
	sessionsMu.Unlock()

	api.ResponseJSON(w, map[string]any{"sessions": items})
}

func apiTestGet(w http.ResponseWriter, id string) {
	sessionsMu.Lock()
	s := sessions[id]
	sessionsMu.Unlock()

	if s == nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	s.Lock()
	api.ResponseJSON(w, s)
	s.Unlock()
}

func apiTestCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Sources struct {
			Streams []string `json:"streams"`
		} `json:"sources"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if len(req.Sources.Streams) == 0 {
		http.Error(w, "sources.streams required", http.StatusBadRequest)
		return
	}

	id := randID()
	s := tester.NewSession(id, len(req.Sources.Streams))

	sessionsMu.Lock()
	sessions[id] = s
	sessionsMu.Unlock()

	log.Debug().Str("id", id).Int("urls", len(req.Sources.Streams)).Msg("[test] session created")

	go tester.RunWorkers(s, req.Sources.Streams)

	api.ResponseJSON(w, map[string]string{"session_id": id})
}

func apiTestDelete(w http.ResponseWriter, id string) {
	sessionsMu.Lock()
	if s, ok := sessions[id]; ok {
		s.Cancel()
		delete(sessions, id)
	}
	sessionsMu.Unlock()

	api.ResponseJSON(w, map[string]string{"status": "deleted"})
}

func apiScreenshot(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query()
	id := q.Get("id")
	idx, err := strconv.Atoi(q.Get("i"))
	if id == "" || err != nil {
		http.Error(w, "id and i required", http.StatusBadRequest)
		return
	}

	sessionsMu.Lock()
	s := sessions[id]
	sessionsMu.Unlock()

	if s == nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	data := s.GetScreenshot(idx)
	if data == nil {
		http.Error(w, "screenshot not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Header().Set("Cache-Control", "no-cache")
	w.Write(data)
}

func randID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

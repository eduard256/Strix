package api

import (
	"embed"
	"encoding/json"
	"io/fs"
	"net"
	"net/http"
	"time"

	"github.com/eduard256/strix/internal/app"
	"github.com/rs/zerolog"
)

var log zerolog.Logger

var Handler http.Handler

func Init() {
	listen := app.Env("STRIX_LISTEN", ":4567")

	log = app.GetLogger("api")

	HandleFunc("api", apiHandler)
	HandleFunc("api/health", apiHealth)
	HandleFunc("api/log", apiLog)

	// serve frontend from embedded web/ directory
	if sub, err := fs.Sub(webFS, "web"); err == nil {
		http.Handle("/", http.FileServer(http.FS(sub)))
	}

	Handler = middlewareCORS(http.DefaultServeMux)

	if log.Trace().Enabled() {
		Handler = middlewareLog(Handler)
	}

	go listen_serve("tcp", listen)
}

//go:embed web
var webFS embed.FS

func listen_serve(network, address string) {
	ln, err := net.Listen(network, address)
	if err != nil {
		log.Error().Err(err).Msg("[api] listen")
		return
	}

	log.Info().Str("addr", address).Msg("[api] listen")

	server := http.Server{
		Handler:      Handler,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Minute, // long for test sessions
	}
	if err = server.Serve(ln); err != nil {
		log.Fatal().Err(err).Msg("[api] serve")
	}
}

// HandleFunc registers handler on http.DefaultServeMux with "/" prefix
func HandleFunc(pattern string, handler http.HandlerFunc) {
	if len(pattern) == 0 || pattern[0] != '/' {
		pattern = "/" + pattern
	}
	log.Trace().Str("path", pattern).Msg("[api] register")
	http.HandleFunc(pattern, handler)
}

// ResponseJSON writes JSON response with Content-Type header
func ResponseJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// Error logs error and writes HTTP error response
func Error(w http.ResponseWriter, err error, code int) {
	log.Error().Err(err).Caller(1).Send()
	http.Error(w, err.Error(), code)
}

func middlewareCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			return
		}
		next.ServeHTTP(w, r)
	})
}

func middlewareLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Trace().Msgf("[api] %s %s %s", r.Method, r.URL, r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}

func apiHandler(w http.ResponseWriter, r *http.Request) {
	ResponseJSON(w, app.Info)
}

func apiHealth(w http.ResponseWriter, r *http.Request) {
	ResponseJSON(w, map[string]any{
		"version": app.Version,
		"uptime":  time.Since(app.StartTime).Truncate(time.Second).String(),
	})
}

func apiLog(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		w.Header().Set("Content-Type", "application/jsonlines")
		app.MemoryLog.WriteTo(w)
	case "DELETE":
		app.MemoryLog.Reset()
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

package webui

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

//go:embed web
var webFiles embed.FS

// Server represents the Web UI server
type Server struct {
	router chi.Router
	logger interface{ Info(string, ...any); Error(string, error, ...any) }
}

// NewServer creates a new Web UI server
func NewServer(logger interface{ Info(string, ...any); Error(string, error, ...any) }) *Server {
	server := &Server{
		router: chi.NewRouter(),
		logger: logger,
	}

	server.setupRoutes()
	return server
}

// setupRoutes configures all routes for the web UI
func (s *Server) setupRoutes() {
	// Middleware
	s.router.Use(middleware.RequestID)
	s.router.Use(middleware.RealIP)
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)

	// CORS middleware
	s.router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-Request-ID")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	})

	// Get the embedded filesystem
	webFS, err := fs.Sub(webFiles, "web")
	if err != nil {
		s.logger.Error("failed to get web filesystem", err)
		return
	}

	// Serve static files
	fileServer := http.FileServer(http.FS(webFS))
	s.router.Handle("/*", fileServer)
}

// ServeHTTP implements http.Handler
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

// GetRouter returns the chi router
func (s *Server) GetRouter() chi.Router {
	return s.router
}

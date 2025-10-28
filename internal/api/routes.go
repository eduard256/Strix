package api

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/strix-project/strix/internal/api/handlers"
	"github.com/strix-project/strix/internal/camera/database"
	"github.com/strix-project/strix/internal/camera/discovery"
	"github.com/strix-project/strix/internal/camera/stream"
	"github.com/strix-project/strix/internal/config"
	"github.com/strix-project/strix/pkg/sse"
)

// Server represents the API server
type Server struct {
	router       chi.Router
	config       *config.Config
	loader       *database.Loader
	searchEngine *database.SearchEngine
	scanner      *discovery.Scanner
	sseServer    *sse.Server
	logger       interface{ Debug(string, ...any); Error(string, error, ...any); Info(string, ...any) }
}

// NewServer creates a new API server
func NewServer(
	cfg *config.Config,
	logger interface{ Debug(string, ...any); Error(string, error, ...any); Info(string, ...any) },
) (*Server, error) {
	// Initialize database loader
	loader := database.NewLoader(
		cfg.Database.BrandsPath,
		cfg.Database.PatternsPath,
		cfg.Database.ParametersPath,
		logger,
	)

	// Load query parameters for URL builder
	queryParams, err := loader.LoadQueryParameters()
	if err != nil {
		return nil, err
	}

	// Initialize search engine
	searchEngine := database.NewSearchEngine(loader, logger)

	// Initialize stream components
	builder := stream.NewBuilder(queryParams, logger)
	tester := stream.NewTester(cfg.Scanner.FFProbeTimeout, logger)

	// Initialize ONVIF discovery
	onvif := discovery.NewONVIFDiscovery(logger)

	// Initialize scanner
	scannerConfig := discovery.ScannerConfig{
		WorkerPoolSize:   cfg.Scanner.WorkerPoolSize,
		DefaultTimeout:   cfg.Scanner.DefaultTimeout,
		MaxStreams:       cfg.Scanner.MaxStreams,
		ModelSearchLimit: cfg.Scanner.ModelSearchLimit,
		FFProbeTimeout:   cfg.Scanner.FFProbeTimeout,
	}

	scanner := discovery.NewScanner(
		loader,
		searchEngine,
		builder,
		tester,
		onvif,
		scannerConfig,
		logger,
	)

	// Initialize SSE server
	sseServer := sse.NewServer(logger)

	// Create server
	server := &Server{
		router:       chi.NewRouter(),
		config:       cfg,
		loader:       loader,
		searchEngine: searchEngine,
		scanner:      scanner,
		sseServer:    sseServer,
		logger:       logger,
	}

	// Setup routes
	server.setupRoutes()

	return server, nil
}

// setupRoutes configures all routes and middleware
func (s *Server) setupRoutes() {
	// Global middleware
	s.router.Use(middleware.RequestID)
	s.router.Use(middleware.RealIP)
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)
	s.router.Use(middleware.Timeout(60 * time.Second))

	// CORS middleware
	s.router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-Request-ID")
			w.Header().Set("Access-Control-Max-Age", "3600")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	})

	// API version 1 routes
	s.router.Route("/api/v1", func(r chi.Router) {
		// Health check
		r.Get("/health", handlers.NewHealthHandler("1.0.0", s.logger).ServeHTTP)

		// Camera search
		r.Post("/cameras/search", handlers.NewSearchHandler(s.searchEngine, s.logger).ServeHTTP)

		// Stream discovery (SSE)
		r.Post("/streams/discover", handlers.NewDiscoverHandler(s.scanner, s.sseServer, s.logger).ServeHTTP)
	})

	// Root health check
	s.router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"name":"Strix","version":"1.0.0","api":"v1"}`))
	})

	// 404 handler
	s.router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"Not found"}`))
	})
}

// ServeHTTP implements http.Handler
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

// GetRouter returns the chi router
func (s *Server) GetRouter() chi.Router {
	return s.router
}
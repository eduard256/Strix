package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/strix-project/strix/internal/api"
	"github.com/strix-project/strix/internal/config"
	"github.com/strix-project/strix/internal/utils/logger"
)

const (
	// Version is the application version
	Version = "1.0.0"

	// Banner is the application banner
	Banner = `
███████╗████████╗██████╗ ██╗██╗  ██╗
██╔════╝╚══██╔══╝██╔══██╗██║╚██╗██╔╝
███████╗   ██║   ██████╔╝██║ ╚███╔╝
╚════██║   ██║   ██╔══██╗██║ ██╔██╗
███████║   ██║   ██║  ██║██║██╔╝ ██╗
╚══════╝   ╚═╝   ╚═╝  ╚═╝╚═╝╚═╝  ╚═╝

Smart IP Camera Stream Discovery System
Version: %s
`
)

func main() {
	// Print banner
	fmt.Printf(Banner, Version)
	fmt.Println()

	// Load configuration
	cfg := config.Load()

	// Setup logger
	slogger := cfg.SetupLogger()
	slog.SetDefault(slogger)

	// Create adapter for our interface
	log := logger.NewAdapter(slogger)

	log.Info("starting Strix",
		slog.String("version", Version),
		slog.String("go_version", os.Getenv("GO_VERSION")),
		slog.String("host", cfg.Server.Host),
		slog.String("port", cfg.Server.Port),
	)

	// Check if ffprobe is available
	if err := checkFFProbe(); err != nil {
		log.Warn("ffprobe not found, stream validation will be limited", slog.String("error", err.Error()))
	}

	// Create API server
	apiServer, err := api.NewServer(cfg, log)
	if err != nil {
		log.Error("failed to create API server", err)
		os.Exit(1)
	}

	// Create HTTP server
	httpServer := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port),
		Handler:      apiServer,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Info("HTTP server starting",
			slog.String("address", httpServer.Addr),
			slog.String("api_version", "v1"),
		)

		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("HTTP server failed", err)
			os.Exit(1)
		}
	}()

	// Print API endpoints
	printEndpoints(cfg.Server.Host, cfg.Server.Port)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Info("shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Error("server shutdown failed", err)
		os.Exit(1)
	}

	log.Info("server stopped gracefully")
}

// checkFFProbe checks if ffprobe is available
func checkFFProbe() error {
	// Try to execute ffprobe -version
	cmd := os.Getenv("PATH")
	if cmd == "" {
		return fmt.Errorf("PATH environment variable not set")
	}

	// For now, just check if ffprobe exists in common locations
	locations := []string{
		"/usr/bin/ffprobe",
		"/usr/local/bin/ffprobe",
		"/opt/homebrew/bin/ffprobe",
	}

	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			return nil
		}
	}

	return fmt.Errorf("ffprobe not found in common locations")
}

// printEndpoints prints available API endpoints
func printEndpoints(host, port string) {
	if host == "0.0.0.0" || host == "" {
		host = "localhost"
	}

	baseURL := fmt.Sprintf("http://%s:%s", host, port)

	fmt.Println("\n🚀 API Endpoints:")
	fmt.Println("────────────────────────────────────────────────")
	fmt.Printf("  Health Check:     GET  %s/api/v1/health\n", baseURL)
	fmt.Printf("  Camera Search:    POST %s/api/v1/cameras/search\n", baseURL)
	fmt.Printf("  Stream Discovery: POST %s/api/v1/streams/discover (SSE)\n", baseURL)
	fmt.Println("────────────────────────────────────────────────")

	fmt.Println("\n📝 Example Requests:")
	fmt.Println("\n1. Search for cameras:")
	fmt.Printf(`   curl -X POST %s/api/v1/cameras/search \
     -H "Content-Type: application/json" \
     -d '{"query": "zosi zg23213m", "limit": 10}'
`, baseURL)

	fmt.Println("\n2. Discover streams (SSE):")
	fmt.Printf(`   curl -X POST %s/api/v1/streams/discover \
     -H "Content-Type: application/json" \
     -d '{
       "target": "192.168.1.100",
       "model": "zosi zg23213m",
       "username": "admin",
       "password": "password",
       "timeout": 240,
       "max_streams": 10
     }'
`, baseURL)

	fmt.Println("\n3. Check health:")
	fmt.Printf("   curl %s/api/v1/health\n", baseURL)

	fmt.Println("\n────────────────────────────────────────────────")
	fmt.Println("📚 Documentation: https://github.com/strix-project/strix")
	fmt.Println("────────────────────────────────────────────────\n")
}
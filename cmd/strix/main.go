package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/eduard256/Strix/internal/api"
	"github.com/eduard256/Strix/internal/config"
	"github.com/eduard256/Strix/internal/utils/logger"
	"github.com/eduard256/Strix/webui"
	"github.com/go-chi/chi/v5"
)

const (
	// Version is the application version
	Version = "1.0.4"

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
		slog.String("listen", cfg.Server.Listen),
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

	// Create Web UI server
	webuiServer := webui.NewServer(log)

	// Create unified router combining API and WebUI
	unifiedRouter := chi.NewRouter()

	// Mount API routes at /api/v1/*
	unifiedRouter.Mount("/api/v1", apiServer.GetRouter())

	// Mount WebUI routes at /* (serves everything else including root)
	unifiedRouter.Mount("/", webuiServer.GetRouter())

	// Create unified HTTP server
	httpServer := &http.Server{
		Addr:         cfg.Server.Listen,
		Handler:      unifiedRouter,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Info("server starting",
			slog.String("address", httpServer.Addr),
			slog.String("api_version", "v1"),
		)

		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server failed", err)
			os.Exit(1)
		}
	}()

	// Print endpoints
	printEndpoints(cfg.Server.Listen)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Info("shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown server
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

// getLocalIP returns the local IP address of the machine
func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "localhost"
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}

	return "localhost"
}

// printEndpoints prints available endpoints
func printEndpoints(listen string) {
	// Extract port from listen address
	port := "4567"
	if len(listen) > 0 {
		if listen[0] == ':' {
			port = listen[1:]
		} else {
			// Parse host:port format
			for i := len(listen) - 1; i >= 0; i-- {
				if listen[i] == ':' {
					port = listen[i+1:]
					break
				}
			}
		}
	}

	// Get local IP
	localIP := getLocalIP()
	url := fmt.Sprintf("http://%s:%s", localIP, port)

	// ANSI escape codes for clickable link (OSC 8 hyperlink)
	clickableURL := fmt.Sprintf("\033]8;;%s\033\\%s\033]8;;\033\\", url, url)

	fmt.Println("\n🌐 Web Interface:")
	fmt.Println("────────────────────────────────────────────────")
	fmt.Printf("  Open in browser: %s\n", clickableURL)
	fmt.Println("────────────────────────────────────────────────")

	fmt.Println("\n📚 Documentation: https://github.com/eduard256/Strix")
}

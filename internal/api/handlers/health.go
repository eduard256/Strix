package handlers

import (
	"encoding/json"
	"net/http"
	"runtime"
	"time"
)

// HealthResponse represents health check response
type HealthResponse struct {
	Status    string            `json:"status"`
	Version   string            `json:"version"`
	Uptime    int64             `json:"uptime"` // seconds
	Timestamp string            `json:"timestamp"`
	System    SystemInfo        `json:"system"`
	Services  map[string]string `json:"services"`
}

// SystemInfo contains system information
type SystemInfo struct {
	GoVersion    string `json:"go_version"`
	NumGoroutine int    `json:"num_goroutines"`
	NumCPU       int    `json:"num_cpu"`
	MemoryMB     uint64 `json:"memory_mb"`
}

var startTime = time.Now()

// HealthHandler handles health check endpoint
type HealthHandler struct {
	version string
	logger  interface{ Info(string, ...any) }
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(version string, logger interface{ Info(string, ...any) }) *HealthHandler {
	return &HealthHandler{
		version: version,
		logger:  logger,
	}
}

// ServeHTTP handles health check requests
func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.logger.Info("health check requested", "remote_addr", r.RemoteAddr)

	// Get memory stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	response := HealthResponse{
		Status:    "healthy",
		Version:   h.version,
		Uptime:    int64(time.Since(startTime).Seconds()),
		Timestamp: time.Now().Format(time.RFC3339),
		System: SystemInfo{
			GoVersion:    runtime.Version(),
			NumGoroutine: runtime.NumGoroutine(),
			NumCPU:       runtime.NumCPU(),
			MemoryMB:     memStats.Alloc / 1024 / 1024,
		},
		Services: map[string]string{
			"api":      "running",
			"database": "loaded",
			"scanner":  "ready",
			"sse":      "active",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Info("failed to encode health response", "error", err.Error())
	}
}
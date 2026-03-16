package handlers

import (
	"encoding/json"
	"net"
	"net/http"

	"github.com/eduard256/Strix/internal/camera/discovery"
)

// ProbeHandler handles device probe requests.
// GET /api/v1/probe?ip=192.168.1.50
type ProbeHandler struct {
	probeService *discovery.ProbeService
	logger       interface {
		Debug(string, ...any)
		Error(string, error, ...any)
		Info(string, ...any)
	}
}

// NewProbeHandler creates a new probe handler.
func NewProbeHandler(
	probeService *discovery.ProbeService,
	logger interface {
		Debug(string, ...any)
		Error(string, error, ...any)
		Info(string, ...any)
	},
) *ProbeHandler {
	return &ProbeHandler{
		probeService: probeService,
		logger:       logger,
	}
}

// ServeHTTP handles probe requests.
func (h *ProbeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ip := r.URL.Query().Get("ip")
	if ip == "" {
		h.sendError(w, "Missing required parameter: ip", http.StatusBadRequest)
		return
	}

	// Validate IP format
	if net.ParseIP(ip) == nil {
		h.sendError(w, "Invalid IP address: "+ip, http.StatusBadRequest)
		return
	}

	h.logger.Info("probe requested", "ip", ip, "remote_addr", r.RemoteAddr)

	// Run probe
	result := h.probeService.Probe(r.Context(), ip)

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(result); err != nil {
		h.logger.Error("failed to encode probe response", err)
	}
}

// sendError sends a JSON error response.
func (h *ProbeHandler) sendError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := map[string]interface{}{
		"error":   true,
		"message": message,
		"code":    statusCode,
	}

	_ = json.NewEncoder(w).Encode(response)
}

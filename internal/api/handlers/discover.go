package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/strix-project/strix/internal/camera/discovery"
	"github.com/strix-project/strix/internal/models"
	"github.com/strix-project/strix/pkg/sse"
)

// DiscoverHandler handles stream discovery requests
type DiscoverHandler struct {
	scanner   *discovery.Scanner
	sseServer *sse.Server
	validator *validator.Validate
	logger    interface{ Debug(string, ...any); Error(string, error, ...any); Info(string, ...any) }
}

// NewDiscoverHandler creates a new discover handler
func NewDiscoverHandler(
	scanner *discovery.Scanner,
	sseServer *sse.Server,
	logger interface{ Debug(string, ...any); Error(string, error, ...any); Info(string, ...any) },
) *DiscoverHandler {
	return &DiscoverHandler{
		scanner:   scanner,
		sseServer: sseServer,
		validator: validator.New(),
		logger:    logger,
	}
}

// ServeHTTP handles discovery requests
func (h *DiscoverHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var req models.StreamDiscoveryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode discovery request", err)
		h.sendErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Set defaults
	if req.ModelLimit <= 0 {
		req.ModelLimit = 6
	}
	if req.Timeout <= 0 {
		req.Timeout = 240 // 4 minutes
	}
	if req.MaxStreams <= 0 {
		req.MaxStreams = 10
	}

	// Validate request
	if err := h.validator.Struct(req); err != nil {
		h.logger.Error("discovery request validation failed", err)
		h.sendErrorResponse(w, "Validation failed: "+err.Error(), http.StatusBadRequest)
		return
	}

	h.logger.Info("stream discovery requested",
		"target", req.Target,
		"model", req.Model,
		"timeout", req.Timeout,
		"max_streams", req.MaxStreams,
		"remote_addr", r.RemoteAddr,
	)

	// Check if SSE is supported
	flusher, ok := w.(http.Flusher)
	if !ok {
		h.logger.Info("SSE not supported by client", "remote_addr", r.RemoteAddr)
		h.sendErrorResponse(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("X-Accel-Buffering", "no") // Disable Nginx buffering

	// Flush headers
	flusher.Flush()

	// Create SSE stream writer
	streamWriter, err := h.sseServer.NewStreamWriter(w, r)
	if err != nil {
		h.logger.Error("failed to create SSE stream", err)
		return
	}
	defer streamWriter.Close()

	// Perform discovery
	result, err := h.scanner.Scan(r.Context(), req, streamWriter)
	if err != nil {
		h.logger.Error("discovery failed", err)
		streamWriter.SendError(err)
		return
	}

	// Send final summary
	streamWriter.SendJSON("summary", map[string]interface{}{
		"total_tested":  result.TotalTested,
		"total_found":   result.TotalFound,
		"duration":      result.Duration.Seconds(),
		"streams_count": len(result.Streams),
	})

	h.logger.Info("discovery completed",
		"target", req.Target,
		"tested", result.TotalTested,
		"found", result.TotalFound,
		"duration", result.Duration,
	)
}

// sendErrorResponse sends an error response for non-SSE requests
func (h *DiscoverHandler) sendErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := map[string]interface{}{
		"error":   true,
		"message": message,
		"code":    statusCode,
	}

	json.NewEncoder(w).Encode(response)
}
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/eduard256/Strix/internal/camera/database"
	"github.com/eduard256/Strix/internal/models"
)

// SearchHandler handles camera search requests
type SearchHandler struct {
	searchEngine *database.SearchEngine
	validator    *validator.Validate
	logger       interface{ Debug(string, ...any); Error(string, error, ...any); Info(string, ...any) }
}

// NewSearchHandler creates a new search handler
func NewSearchHandler(
	searchEngine *database.SearchEngine,
	logger interface{ Debug(string, ...any); Error(string, error, ...any); Info(string, ...any) },
) *SearchHandler {
	return &SearchHandler{
		searchEngine: searchEngine,
		validator:    validator.New(),
		logger:       logger,
	}
}

// ServeHTTP handles search requests
func (h *SearchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var req models.CameraSearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode search request", err)
		h.sendErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Set default limit if not provided
	if req.Limit <= 0 {
		req.Limit = 10
	}

	// Validate request
	if err := h.validator.Struct(req); err != nil {
		h.logger.Error("search request validation failed", err)
		h.sendErrorResponse(w, "Validation failed: "+err.Error(), http.StatusBadRequest)
		return
	}

	h.logger.Info("camera search requested",
		"query", req.Query,
		"limit", req.Limit,
		"remote_addr", r.RemoteAddr,
	)

	// Perform search
	response, err := h.searchEngine.Search(req.Query, req.Limit)
	if err != nil {
		h.logger.Error("search failed", err)
		h.sendErrorResponse(w, "Search failed", http.StatusInternalServerError)
		return
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("failed to encode search response", err)
	}

	h.logger.Info("search completed",
		"query", req.Query,
		"returned", response.Returned,
		"total", response.Total,
	)
}

// sendErrorResponse sends an error response
func (h *SearchHandler) sendErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := map[string]interface{}{
		"error":   true,
		"message": message,
		"code":    statusCode,
	}

	_ = json.NewEncoder(w).Encode(response)
}
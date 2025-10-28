package models

import "time"

// Camera represents a camera model from the database
type Camera struct {
	Brand       string         `json:"brand"`
	BrandID     string         `json:"brand_id"`
	Model       string         `json:"model"`
	LastUpdated string         `json:"last_updated"`
	Source      string         `json:"source"`
	Website     string         `json:"website,omitempty"`
	Entries     []CameraEntry  `json:"entries"`
	MatchScore  float64        `json:"match_score,omitempty"`
}

// CameraEntry represents a URL pattern entry for a camera
type CameraEntry struct {
	Models       []string `json:"models"`
	Type         string   `json:"type"` // FFMPEG, MJPEG, JPEG, VLC, H264
	Protocol     string   `json:"protocol"` // rtsp, http, https
	Port         int      `json:"port"`
	URL          string   `json:"url"`
	AuthRequired bool     `json:"auth_required,omitempty"`
	Notes        string   `json:"notes,omitempty"`
}

// StreamPattern represents a popular stream pattern
type StreamPattern struct {
	URL        string `json:"url"`
	Type       string `json:"type"`
	Protocol   string `json:"protocol"`
	Port       int    `json:"port"`
	Notes      string `json:"notes"`
	ModelCount int    `json:"model_count"`
}

// CameraSearchRequest represents a search request for cameras
type CameraSearchRequest struct {
	Query string `json:"query" validate:"required,min=1"`
	Limit int    `json:"limit" validate:"min=1,max=100"`
}

// CameraSearchResponse represents the response for camera search
type CameraSearchResponse struct {
	Cameras  []Camera `json:"cameras"`
	Total    int      `json:"total"`
	Returned int      `json:"returned"`
}

// StreamDiscoveryRequest represents a request to discover streams
type StreamDiscoveryRequest struct {
	Model      string `json:"model"`                             // Camera model name
	ModelLimit int    `json:"model_limit" validate:"min=1,max=20"` // Max models to search
	Timeout    int    `json:"timeout" validate:"min=10,max=600"`   // Timeout in seconds
	MaxStreams int    `json:"max_streams" validate:"min=1,max=50"` // Max streams to find
	Target     string `json:"target" validate:"required"`          // IP or stream URL
	Channel    int    `json:"channel" validate:"min=0,max=255"`    // Channel number
	Username   string `json:"username"`                            // Optional username
	Password   string `json:"password"`                            // Optional password
}

// DiscoveredStream represents a discovered stream
type DiscoveredStream struct {
	URL        string                 `json:"url"`
	Type       string                 `json:"type"` // RTSP, HTTP, MJPEG, etc
	Protocol   string                 `json:"protocol"`
	Port       int                    `json:"port"`
	Working    bool                   `json:"working"`
	Resolution string                 `json:"resolution,omitempty"`
	Codec      string                 `json:"codec,omitempty"`
	FPS        int                    `json:"fps,omitempty"`
	Bitrate    int                    `json:"bitrate,omitempty"`
	HasAudio   bool                   `json:"has_audio"`
	Error      string                 `json:"error,omitempty"`
	TestTime   time.Duration          `json:"test_time_ms"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// SSEMessage represents a Server-Sent Event message
type SSEMessage struct {
	Type    string      `json:"type"` // stream_found, progress, error, complete
	Data    interface{} `json:"data,omitempty"`
	Stream  *DiscoveredStream `json:"stream,omitempty"`
	Message string      `json:"message,omitempty"`
}

// ProgressMessage for SSE progress updates
type ProgressMessage struct {
	Tested    int `json:"tested"`
	Found     int `json:"found"`
	Remaining int `json:"remaining"`
}

// CompleteMessage for SSE completion
type CompleteMessage struct {
	TotalTested int     `json:"total_tested"`
	TotalFound  int     `json:"total_found"`
	Duration    float64 `json:"duration"` // seconds
}
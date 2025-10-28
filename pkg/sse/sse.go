package sse

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Event represents a Server-Sent Event
type Event struct {
	ID      string
	Type    string
	Data    interface{}
	Retry   int
	Comment string
}

// Client represents an SSE client connection
type Client struct {
	ID       string
	Channel  chan Event
	Response http.ResponseWriter
	Request  *http.Request
	Context  context.Context
	Cancel   context.CancelFunc
}

// Server manages SSE connections
type Server struct {
	clients    map[string]*Client
	register   chan *Client
	unregister chan *Client
	broadcast  chan Event
	logger     interface{ Debug(string, ...any); Error(string, error, ...any) }
}

// NewServer creates a new SSE server
func NewServer(logger interface{ Debug(string, ...any); Error(string, error, ...any) }) *Server {
	return &Server{
		clients:    make(map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan Event),
		logger:     logger,
	}
}

// Start starts the SSE server
func (s *Server) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				// Close all client connections
				for _, client := range s.clients {
					client.Cancel()
					close(client.Channel)
				}
				return

			case client := <-s.register:
				s.clients[client.ID] = client
				s.logger.Debug("SSE client registered", "id", client.ID)

			case client := <-s.unregister:
				if _, ok := s.clients[client.ID]; ok {
					delete(s.clients, client.ID)
					close(client.Channel)
					s.logger.Debug("SSE client unregistered", "id", client.ID)
				}

			case event := <-s.broadcast:
				for _, client := range s.clients {
					select {
					case client.Channel <- event:
					default:
						// Client's channel is full, close it
						s.unregister <- client
					}
				}
			}
		}
	}()
}

// ServeHTTP handles SSE connections
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Check if SSE is supported
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("X-Accel-Buffering", "no") // Disable Nginx buffering

	// Create client
	ctx, cancel := context.WithCancel(r.Context())
	client := &Client{
		ID:       generateClientID(),
		Channel:  make(chan Event, 100),
		Response: w,
		Request:  r,
		Context:  ctx,
		Cancel:   cancel,
	}

	// Register client
	s.register <- client

	// Remove client on disconnect
	defer func() {
		s.unregister <- client
		cancel()
	}()

	// Send initial connection event
	s.SendToClient(client, Event{
		Type: "connected",
		Data: map[string]string{"id": client.ID},
	})

	// Listen for events
	for {
		select {
		case <-ctx.Done():
			return

		case <-r.Context().Done():
			return

		case event := <-client.Channel:
			if err := s.writeEvent(w, flusher, event); err != nil {
				s.logger.Error("failed to write SSE event", err, "client", client.ID)
				return
			}
		}
	}
}

// SendToClient sends an event to a specific client
func (s *Server) SendToClient(client *Client, event Event) {
	select {
	case client.Channel <- event:
	default:
		// Channel is full, log warning
		s.logger.Debug("client channel full, dropping event", "client", client.ID)
	}
}

// Broadcast sends an event to all clients
func (s *Server) Broadcast(event Event) {
	s.broadcast <- event
}

// writeEvent writes an event to the response writer
func (s *Server) writeEvent(w http.ResponseWriter, flusher http.Flusher, event Event) error {
	// Write event ID if present
	if event.ID != "" {
		if _, err := fmt.Fprintf(w, "id: %s\n", event.ID); err != nil {
			return err
		}
	}

	// Write event type if present
	if event.Type != "" {
		if _, err := fmt.Fprintf(w, "event: %s\n", event.Type); err != nil {
			return err
		}
	}

	// Write retry if present
	if event.Retry > 0 {
		if _, err := fmt.Fprintf(w, "retry: %d\n", event.Retry); err != nil {
			return err
		}
	}

	// Write comment if present
	if event.Comment != "" {
		if _, err := fmt.Fprintf(w, ": %s\n", event.Comment); err != nil {
			return err
		}
	}

	// Write data
	if event.Data != nil {
		var dataStr string
		switch v := event.Data.(type) {
		case string:
			dataStr = v
		case []byte:
			dataStr = string(v)
		default:
			data, err := json.Marshal(v)
			if err != nil {
				return err
			}
			dataStr = string(data)
		}

		// Split data by newlines for proper SSE format
		for _, line := range splitLines(dataStr) {
			if _, err := fmt.Fprintf(w, "data: %s\n", line); err != nil {
				return err
			}
		}
	}

	// End event with double newline
	if _, err := fmt.Fprintf(w, "\n"); err != nil {
		return err
	}

	// Flush the data
	flusher.Flush()

	return nil
}

// splitLines splits a string into lines
func splitLines(s string) []string {
	var lines []string
	var current string

	for _, ch := range s {
		if ch == '\n' {
			lines = append(lines, current)
			current = ""
		} else {
			current += string(ch)
		}
	}

	if current != "" {
		lines = append(lines, current)
	}

	return lines
}

// generateClientID generates a unique client ID
func generateClientID() string {
	return fmt.Sprintf("client-%d-%d", time.Now().Unix(), time.Now().Nanosecond())
}

// StreamWriter provides a simple interface for writing SSE events
type StreamWriter struct {
	client *Client
	server *Server
}

// NewStreamWriter creates a new stream writer for a client
func (s *Server) NewStreamWriter(w http.ResponseWriter, r *http.Request) (*StreamWriter, error) {
	// Check if SSE is supported
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("SSE not supported")
	}

	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("X-Accel-Buffering", "no")

	// Send initial flush to establish connection
	flusher.Flush()

	// Create client
	ctx, cancel := context.WithCancel(r.Context())
	client := &Client{
		ID:       generateClientID(),
		Channel:  make(chan Event, 100),
		Response: w,
		Request:  r,
		Context:  ctx,
		Cancel:   cancel,
	}

	return &StreamWriter{
		client: client,
		server: s,
	}, nil
}

// SendEvent sends an event through the stream writer
func (sw *StreamWriter) SendEvent(eventType string, data interface{}) error {
	event := Event{
		Type: eventType,
		Data: data,
	}

	flusher, ok := sw.client.Response.(http.Flusher)
	if !ok {
		return fmt.Errorf("response does not support flushing")
	}

	return sw.server.writeEvent(sw.client.Response, flusher, event)
}

// SendJSON sends JSON data as an event
func (sw *StreamWriter) SendJSON(eventType string, v interface{}) error {
	return sw.SendEvent(eventType, v)
}

// SendMessage sends a simple message
func (sw *StreamWriter) SendMessage(message string) error {
	return sw.SendEvent("message", map[string]string{"message": message})
}

// SendError sends an error message
func (sw *StreamWriter) SendError(err error) error {
	return sw.SendEvent("error", map[string]string{"error": err.Error()})
}

// SendProgress sends a progress update
func (sw *StreamWriter) SendProgress(current, total int, message string) error {
	return sw.SendEvent("progress", map[string]interface{}{
		"current": current,
		"total":   total,
		"message": message,
		"percent": float64(current) / float64(total) * 100,
	})
}

// Close closes the stream writer
func (sw *StreamWriter) Close() {
	sw.client.Cancel()
}
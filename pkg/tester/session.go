package tester

import (
	"sync"
	"time"
)

const SessionTTL = 30 * time.Minute

type Session struct {
	ID          string    `json:"session_id"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at,omitempty"`
	Total       int       `json:"total"`
	Tested      int       `json:"tested"`
	Alive       int       `json:"alive"`
	WithScreen  int       `json:"with_screenshot"`
	Results     []*Result `json:"results"`
	Screenshots [][]byte  `json:"-"`

	cancel chan struct{}
	mu     sync.Mutex
}

type Result struct {
	Source     string   `json:"source"`
	Screenshot string   `json:"screenshot,omitempty"`
	Codecs     []string `json:"codecs,omitempty"`
	LatencyMs  int64    `json:"latency_ms,omitempty"`
	Skipped    bool     `json:"skipped,omitempty"`
}

func NewSession(id string, total int) *Session {
	return &Session{
		ID:        id,
		Status:    "running",
		CreatedAt: time.Now(),
		Total:     total,
		cancel:    make(chan struct{}),
	}
}

func (s *Session) AddResult(r *Result) {
	s.mu.Lock()
	s.Results = append(s.Results, r)
	s.Alive++
	if r.Screenshot != "" {
		s.WithScreen++
	}
	s.mu.Unlock()
}

func (s *Session) AddTested() {
	s.mu.Lock()
	s.Tested++
	s.mu.Unlock()
}

func (s *Session) AddScreenshot(data []byte) int {
	s.mu.Lock()
	idx := len(s.Screenshots)
	s.Screenshots = append(s.Screenshots, data)
	s.mu.Unlock()
	return idx
}

func (s *Session) GetScreenshot(idx int) []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	if idx < 0 || idx >= len(s.Screenshots) {
		return nil
	}
	return s.Screenshots[idx]
}

func (s *Session) Done() {
	s.mu.Lock()
	s.Status = "done"
	s.ExpiresAt = time.Now().Add(SessionTTL)
	s.mu.Unlock()
}

func (s *Session) Cancel() {
	select {
	case <-s.cancel:
	default:
		close(s.cancel)
	}
}

func (s *Session) Cancelled() <-chan struct{} {
	return s.cancel
}

func (s *Session) Lock()   { s.mu.Lock() }
func (s *Session) Unlock() { s.mu.Unlock() }

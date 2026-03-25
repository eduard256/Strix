package app

import (
	"io"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

var MemoryLog = NewRingLog(16, 64*1024)

func GetLogger(module string) zerolog.Logger {
	return Logger.With().Str("module", module).Logger()
}

func initLogger() {
	level := Env("STRIX_LOG_LEVEL", "info")
	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		lvl = zerolog.InfoLevel
	}

	writer := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.DateTime,
		NoColor:    !isTTY(),
	}

	multi := io.MultiWriter(&writer, &SecretWriter{w: MemoryLog})

	Logger = zerolog.New(multi).With().Timestamp().Logger().Level(lvl)
}

func isTTY() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// SecretWriter masks passwords in log output
type SecretWriter struct {
	w io.Writer
}

var reURLPassword = regexp.MustCompile(`://([^:]+):([^@]+)@`)
var reQueryPassword = regexp.MustCompile(`(?i)(pass(?:word)?|pwd)=([^&\s"]+)`)

func (s *SecretWriter) Write(p []byte) (int, error) {
	masked := reURLPassword.ReplaceAll(p, []byte("://$1:***@"))
	masked = reQueryPassword.ReplaceAll(masked, []byte("${1}=***"))
	return s.w.Write(masked)
}

// RingLog is a circular buffer for storing log entries in memory
type RingLog struct {
	chunks [][]byte
	pos    int
	mu     sync.Mutex
}

func NewRingLog(count, size int) *RingLog {
	chunks := make([][]byte, count)
	for i := range chunks {
		chunks[i] = make([]byte, 0, size)
	}
	return &RingLog{chunks: chunks}
}

func (r *RingLog) Write(p []byte) (int, error) {
	r.mu.Lock()

	chunk := r.chunks[r.pos]
	if len(chunk)+len(p) > cap(chunk) {
		r.pos = (r.pos + 1) % len(r.chunks)
		r.chunks[r.pos] = r.chunks[r.pos][:0]
		chunk = r.chunks[r.pos]
	}
	r.chunks[r.pos] = append(chunk, p...)

	r.mu.Unlock()
	return len(p), nil
}

func (r *RingLog) WriteTo(w io.Writer) (int64, error) {
	r.mu.Lock()

	var total int64
	start := (r.pos + 1) % len(r.chunks)
	for i := range r.chunks {
		idx := (start + i) % len(r.chunks)
		chunk := r.chunks[idx]
		if len(chunk) == 0 {
			continue
		}
		n, err := w.Write(chunk)
		total += int64(n)
		if err != nil {
			r.mu.Unlock()
			return total, err
		}
	}

	r.mu.Unlock()
	return total, nil
}

func (r *RingLog) Reset() {
	r.mu.Lock()
	for i := range r.chunks {
		r.chunks[i] = r.chunks[i][:0]
	}
	r.pos = 0
	r.mu.Unlock()
}

// MaskURL masks password in a URL string for use in log messages
func MaskURL(rawURL string) string {
	s := reURLPassword.ReplaceAllString(rawURL, "://$1:***@")
	s = reQueryPassword.ReplaceAllString(s, "${1}=***")
	return s
}

// MaskPlaceholders masks password placeholders like [PASSWORD], [PASS], [PWD]
func MaskPlaceholders(s string) string {
	r := strings.NewReplacer(
		"[PASSWORD]", "[***]", "[password]", "[***]",
		"[PASS]", "[***]", "[pass]", "[***]",
		"[PWD]", "[***]", "[pwd]", "[***]",
		"[PASWORD]", "[***]", "[pasword]", "[***]",
	)
	return r.Replace(s)
}

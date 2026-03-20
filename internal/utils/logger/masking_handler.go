package logger

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
)

// SecretStore holds a set of secret strings that should be masked in log output.
// It is safe for concurrent use by multiple goroutines. Multiple concurrent scans
// can register different passwords; all are masked simultaneously.
type SecretStore struct {
	mu      sync.RWMutex
	secrets map[string]struct{}
}

// NewSecretStore creates a new empty secret store.
func NewSecretStore() *SecretStore {
	return &SecretStore{
		secrets: make(map[string]struct{}),
	}
}

// Add registers a secret string to be masked in all future log output.
// Empty strings are ignored.
func (s *SecretStore) Add(secret string) {
	if secret == "" {
		return
	}
	s.mu.Lock()
	s.secrets[secret] = struct{}{}
	s.mu.Unlock()
}

// Remove unregisters a secret string so it is no longer masked.
func (s *SecretStore) Remove(secret string) {
	if secret == "" {
		return
	}
	s.mu.Lock()
	delete(s.secrets, secret)
	s.mu.Unlock()
}

// Mask replaces all registered secret strings in text with "***".
// Returns the original string unchanged if no secrets are registered.
func (s *SecretStore) Mask(text string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.secrets) == 0 {
		return text
	}

	for secret := range s.secrets {
		if strings.Contains(text, secret) {
			text = strings.ReplaceAll(text, secret, "***")
		}
	}
	return text
}

// SecretMaskingHandler wraps a slog.Handler and replaces registered secrets
// with "***" in all log record messages and attribute values before passing
// them to the inner handler. This ensures credentials never appear in log
// output regardless of where they originate in the code.
type SecretMaskingHandler struct {
	inner   slog.Handler
	secrets *SecretStore
}

// NewSecretMaskingHandler creates a handler that masks secrets in log output.
func NewSecretMaskingHandler(inner slog.Handler, secrets *SecretStore) *SecretMaskingHandler {
	return &SecretMaskingHandler{
		inner:   inner,
		secrets: secrets,
	}
}

// Enabled reports whether the inner handler handles records at the given level.
func (h *SecretMaskingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

// Handle masks secrets in the record message and all attributes, then
// delegates to the inner handler.
func (h *SecretMaskingHandler) Handle(ctx context.Context, record slog.Record) error {
	// Fast path: no secrets registered
	h.secrets.mu.RLock()
	hasSecrets := len(h.secrets.secrets) > 0
	h.secrets.mu.RUnlock()

	if !hasSecrets {
		return h.inner.Handle(ctx, record)
	}

	// Mask the message
	record.Message = h.secrets.Mask(record.Message)

	// Mask all attributes by collecting, masking, and replacing them
	maskedAttrs := make([]slog.Attr, 0, record.NumAttrs())
	record.Attrs(func(a slog.Attr) bool {
		maskedAttrs = append(maskedAttrs, h.maskAttr(a))
		return true
	})

	// Create a new record without the old attrs and add the masked ones.
	// slog.Record doesn't have a method to clear attrs, so we build a new one.
	newRecord := slog.NewRecord(record.Time, record.Level, record.Message, record.PC)
	newRecord.AddAttrs(maskedAttrs...)

	return h.inner.Handle(ctx, newRecord)
}

// WithAttrs returns a new handler with the given pre-masked attributes.
func (h *SecretMaskingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	masked := make([]slog.Attr, len(attrs))
	for i, a := range attrs {
		masked[i] = h.maskAttr(a)
	}
	return &SecretMaskingHandler{
		inner:   h.inner.WithAttrs(masked),
		secrets: h.secrets,
	}
}

// WithGroup returns a new handler with the given group name.
func (h *SecretMaskingHandler) WithGroup(name string) slog.Handler {
	return &SecretMaskingHandler{
		inner:   h.inner.WithGroup(name),
		secrets: h.secrets,
	}
}

// maskAttr masks secrets in an attribute value. Handles string values,
// error values, and recursively handles group attributes.
func (h *SecretMaskingHandler) maskAttr(a slog.Attr) slog.Attr {
	switch a.Value.Kind() {
	case slog.KindString:
		a.Value = slog.StringValue(h.secrets.Mask(a.Value.String()))

	case slog.KindGroup:
		attrs := a.Value.Group()
		masked := make([]slog.Attr, len(attrs))
		for i, ga := range attrs {
			masked[i] = h.maskAttr(ga)
		}
		a.Value = slog.GroupValue(masked...)

	case slog.KindAny:
		v := a.Value.Any()

		// Handle error values (Go's http.Client embeds full URLs in errors)
		if err, ok := v.(error); ok {
			masked := h.secrets.Mask(err.Error())
			a.Value = slog.StringValue(masked)
			return a
		}

		// Handle fmt.Stringer (e.g. time.Duration, url.URL, etc.)
		if stringer, ok := v.(fmt.Stringer); ok {
			masked := h.secrets.Mask(stringer.String())
			a.Value = slog.StringValue(masked)
			return a
		}

		// Handle string slices (used in BuildURLsFromEntry logging)
		if ss, ok := v.([]string); ok {
			maskedSlice := make([]string, len(ss))
			for i, s := range ss {
				maskedSlice[i] = h.secrets.Mask(s)
			}
			a.Value = slog.AnyValue(maskedSlice)
			return a
		}

		// For other Any values, convert to string and mask
		str := fmt.Sprintf("%v", v)
		masked := h.secrets.Mask(str)
		if masked != str {
			a.Value = slog.StringValue(masked)
		}
	}

	return a
}

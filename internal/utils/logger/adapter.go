package logger

import "log/slog"

// Adapter wraps slog.Logger to match our interface
type Adapter struct {
	*slog.Logger
}

// NewAdapter creates a new logger adapter
func NewAdapter(logger *slog.Logger) *Adapter {
	return &Adapter{Logger: logger}
}

// Debug logs a debug message
func (a *Adapter) Debug(msg string, args ...any) {
	a.Logger.Debug(msg, args...)
}

// Info logs an info message
func (a *Adapter) Info(msg string, args ...any) {
	a.Logger.Info(msg, args...)
}

// Error logs an error message
func (a *Adapter) Error(msg string, err error, args ...any) {
	allArgs := append([]any{"error", err}, args...)
	a.Logger.Error(msg, allArgs...)
}

// Warn logs a warning message
func (a *Adapter) Warn(msg string, args ...any) {
	a.Logger.Warn(msg, args...)
}
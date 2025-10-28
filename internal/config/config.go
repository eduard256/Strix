package config

import (
	"log/slog"
	"os"
	"time"
)

// Config holds application configuration
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Scanner  ScannerConfig
	Logger   LoggerConfig
}

// ServerConfig contains HTTP server settings
type ServerConfig struct {
	Host         string
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// DatabaseConfig contains database settings
type DatabaseConfig struct {
	DataPath       string
	BrandsPath     string
	PatternsPath   string
	ParametersPath string
	CacheEnabled   bool
	CacheTTL       time.Duration
}

// ScannerConfig contains stream scanner settings
type ScannerConfig struct {
	DefaultTimeout    time.Duration
	MaxStreams        int
	ModelSearchLimit  int
	WorkerPoolSize    int
	FFProbeTimeout    time.Duration
	RetryAttempts     int
	RetryDelay        time.Duration
}

// LoggerConfig contains logging settings
type LoggerConfig struct {
	Level  string
	Format string // "text" or "json"
}

// Load returns configuration with defaults
func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Host:         getEnv("STRIX_HOST", "0.0.0.0"),
			Port:         getEnv("STRIX_PORT", "8080"),
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
		},
		Database: DatabaseConfig{
			DataPath:       getEnv("STRIX_DATA_PATH", "/home/dev/Strix/data"),
			BrandsPath:     "/home/dev/Strix/data/brands",
			PatternsPath:   "/home/dev/Strix/data/popular_stream_patterns.json",
			ParametersPath: "/home/dev/Strix/data/query_parameters.json",
			CacheEnabled:   true,
			CacheTTL:       5 * time.Minute,
		},
		Scanner: ScannerConfig{
			DefaultTimeout:   4 * time.Minute,
			MaxStreams:       10,
			ModelSearchLimit: 6,
			WorkerPoolSize:   20,
			FFProbeTimeout:   5 * time.Second,
			RetryAttempts:    2,
			RetryDelay:       500 * time.Millisecond,
		},
		Logger: LoggerConfig{
			Level:  getEnv("STRIX_LOG_LEVEL", "info"),
			Format: getEnv("STRIX_LOG_FORMAT", "json"),
		},
	}
}

// SetupLogger configures the global logger
func (c *Config) SetupLogger() *slog.Logger {
	var level slog.Level
	switch c.Logger.Level {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	if c.Logger.Format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
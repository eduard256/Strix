package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
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
	Listen       string        // Address to listen on (e.g., ":4567" or "0.0.0.0:4567")
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
	DefaultTimeout   time.Duration
	MaxStreams       int
	ModelSearchLimit int
	WorkerPoolSize   int
	FFProbeTimeout   time.Duration
	RetryAttempts    int
	RetryDelay       time.Duration
	// Validation settings
	StrictValidation bool // Enable strict validation mode
	MinImageSize     int  // Minimum bytes for valid image (JPEG/PNG)
	MinVideoStreams  int  // Minimum video streams required
}

// LoggerConfig contains logging settings
type LoggerConfig struct {
	Level  string
	Format string // "text" or "json"
}

// yamlConfig represents the structure of strix.yaml
type yamlConfig struct {
	API struct {
		Listen string `yaml:"listen"`
	} `yaml:"api"`
}

// Load returns configuration with defaults
func Load() *Config {
	dataPath := getEnv("STRIX_DATA_PATH", "./data")

	cfg := &Config{
		Server: ServerConfig{
			Listen:       ":4567", // Default listen address
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
		},
		Database: DatabaseConfig{
			DataPath:       dataPath,
			BrandsPath:     filepath.Join(dataPath, "brands"),
			PatternsPath:   filepath.Join(dataPath, "popular_stream_patterns.json"),
			ParametersPath: filepath.Join(dataPath, "query_parameters.json"),
			CacheEnabled:   true,
			CacheTTL:       5 * time.Minute,
		},
		Scanner: ScannerConfig{
			DefaultTimeout:   4 * time.Minute,
			MaxStreams:       10,
			ModelSearchLimit: 6,
			WorkerPoolSize:   20,
			FFProbeTimeout:   30 * time.Second,
			RetryAttempts:    2,
			RetryDelay:       500 * time.Millisecond,
			// Strict validation enabled by default
			StrictValidation: true,
			MinImageSize:     5120, // 5KB minimum for valid images
			MinVideoStreams:  1,    // At least 1 video stream required
		},
		Logger: LoggerConfig{
			Level:  getEnv("STRIX_LOG_LEVEL", "info"),
			Format: getEnv("STRIX_LOG_FORMAT", "json"),
		},
	}

	// Load from strix.yaml if exists
	configSource := "default"
	if err := loadYAML(cfg); err == nil {
		configSource = "strix.yaml"
	}

	// Environment variable overrides everything
	if envListen := os.Getenv("STRIX_API_LISTEN"); envListen != "" {
		cfg.Server.Listen = envListen
		configSource = "environment variable STRIX_API_LISTEN"
	}

	// Validate listen address
	if err := validateListen(cfg.Server.Listen); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Invalid listen address '%s': %v\n", cfg.Server.Listen, err)
		fmt.Fprintf(os.Stderr, "Using default: :4567\n")
		cfg.Server.Listen = ":4567"
		configSource = "default (validation failed)"
	}

	// Log configuration source
	fmt.Printf("INFO: API listen address '%s' loaded from %s\n", cfg.Server.Listen, configSource)

	return cfg
}

// loadYAML attempts to load configuration from strix.yaml
func loadYAML(cfg *Config) error {
	data, err := os.ReadFile("./strix.yaml")
	if err != nil {
		return err
	}

	var yamlCfg yamlConfig
	if err := yaml.Unmarshal(data, &yamlCfg); err != nil {
		return fmt.Errorf("failed to parse strix.yaml: %w", err)
	}

	// Apply yaml configuration
	if yamlCfg.API.Listen != "" {
		cfg.Server.Listen = yamlCfg.API.Listen
	}

	return nil
}

// validateListen validates the listen address format and port range
func validateListen(listen string) error {
	if listen == "" {
		return fmt.Errorf("listen address cannot be empty")
	}

	// Parse the listen address
	parts := strings.Split(listen, ":")
	if len(parts) < 2 {
		return fmt.Errorf("invalid format, expected ':port' or 'host:port', got '%s'", listen)
	}

	// Get port (last part)
	portStr := parts[len(parts)-1]
	if portStr == "" {
		return fmt.Errorf("port cannot be empty")
	}

	// Validate port number
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("invalid port number '%s': %w", portStr, err)
	}

	if port < 1 || port > 65535 {
		return fmt.Errorf("port %d out of valid range (1-65535)", port)
	}

	return nil
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

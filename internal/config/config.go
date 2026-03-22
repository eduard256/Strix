package config

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/eduard256/Strix/internal/utils/logger"
	"gopkg.in/yaml.v3"
)

// Config holds application configuration
type Config struct {
	Version  string // Application version, set by caller after Load()
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
	OUIPath        string
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

// Load returns configuration with defaults. The provided logger is used for
// all startup messages so that output format stays consistent (JSON or text)
// with the rest of the application logs.
func Load(log *slog.Logger) *Config {
	dataPath := getEnv("STRIX_DATA_PATH", "./data")

	cfg := &Config{
		Server: ServerConfig{
			Listen:       ":4567", // Default listen address
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 5 * time.Minute, // Increased for SSE long-polling
		},
		Database: DatabaseConfig{
			DataPath:       dataPath,
			BrandsPath:     filepath.Join(dataPath, "brands"),
			PatternsPath:   filepath.Join(dataPath, "popular_stream_patterns.json"),
			ParametersPath: filepath.Join(dataPath, "query_parameters.json"),
			OUIPath:        filepath.Join(dataPath, "camera_oui.json"),
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

	// Load from Home Assistant options.json if running as HA add-on
	// Priority: defaults < HA options < strix.yaml < ENV
	configSource := "default"
	if err := loadHAOptions(cfg); err == nil {
		configSource = "/data/options.json (Home Assistant)"
	}

	// Load from strix.yaml if exists (overrides HA options)
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
		log.Error("invalid listen address, using default :4567",
			slog.String("address", cfg.Server.Listen),
			slog.String("error", err.Error()),
		)
		cfg.Server.Listen = ":4567"
		configSource = "default (validation failed)"
	}

	// Log configuration source
	log.Info("configuration loaded",
		slog.String("listen", cfg.Server.Listen),
		slog.String("source", configSource),
	)

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

// haOptions represents the structure of Home Assistant /data/options.json.
// When Strix runs as a Home Assistant add-on, HA creates this file from the
// add-on configuration UI. Fields are optional -- zero values are ignored.
type haOptions struct {
	LogLevel string `json:"log_level"`
	Port     int    `json:"port"`
}

// loadHAOptions loads configuration from Home Assistant's /data/options.json.
// This file only exists when running inside the HA add-on environment.
// Returns an error if the file doesn't exist or can't be parsed (callers
// should treat errors as "not running in HA" and silently continue).
func loadHAOptions(cfg *Config) error {
	data, err := os.ReadFile("/data/options.json")
	if err != nil {
		return err
	}

	var opts haOptions
	if err := json.Unmarshal(data, &opts); err != nil {
		return fmt.Errorf("failed to parse /data/options.json: %w", err)
	}

	if opts.LogLevel != "" {
		cfg.Logger.Level = opts.LogLevel
	}
	if opts.Port > 0 {
		cfg.Server.Listen = fmt.Sprintf(":%d", opts.Port)
	}

	// Home Assistant add-on always uses JSON logging for the HA log viewer
	cfg.Logger.Format = "json"

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

// SetupLogger creates the application logger by reading log configuration
// from environment variables and Home Assistant options. It must be called
// before Load() so that all startup messages use a consistent output format.
//
// Configuration priority: defaults < HA options < environment variables.
func SetupLogger() (*slog.Logger, *logger.SecretStore) {
	// Read log settings from environment (same defaults as Config)
	logLevel := getEnv("STRIX_LOG_LEVEL", "info")
	logFormat := getEnv("STRIX_LOG_FORMAT", "json")

	// Apply Home Assistant overrides if running as HA add-on
	if data, err := os.ReadFile("/data/options.json"); err == nil {
		var opts haOptions
		if err := json.Unmarshal(data, &opts); err == nil {
			if opts.LogLevel != "" {
				logLevel = opts.LogLevel
			}
			// Home Assistant add-on always uses JSON logging for the HA log viewer
			logFormat = "json"
		}
	}

	var level slog.Level
	switch logLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	handlerOpts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	if logFormat == "json" {
		handler = slog.NewJSONHandler(os.Stdout, handlerOpts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, handlerOpts)
	}

	secrets := logger.NewSecretStore()
	maskedHandler := logger.NewSecretMaskingHandler(handler, secrets)

	return slog.New(maskedHandler), secrets
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

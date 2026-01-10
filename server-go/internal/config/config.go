package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
)

// Config holds all application configuration
type Config struct {
	ServerAddress string       `json:"serverAddress"`
	DatabasePath  string       `json:"databasePath"`
	DatabaseURL   string       `json:"databaseUrl"`
	PhotoStorage  PhotoStorage `json:"photoStorage"`
	Security      Security     `json:"security"`
	FileScanner   FileScanner  `json:"fileScanner"`
}

// FileScanner configuration for background file integrity scanning
type FileScanner struct {
	Enabled       bool `json:"enabled"`
	IntervalHours int  `json:"intervalHours"`
	AutoStart     bool `json:"autoStart"`
}

// UsePostgres returns true if PostgreSQL should be used
func (c *Config) UsePostgres() bool {
	return c.DatabaseURL != ""
}

// PhotoStorage configuration
type PhotoStorage struct {
	BasePath          string   `json:"basePath"`
	MaxFileSizeMB     int64    `json:"maxFileSizeMB"`
	AllowedExtensions []string `json:"allowedExtensions"`
}

// Security configuration
type Security struct {
	APIKey       string `json:"apiKey"`
	APIKeyHeader string `json:"apiKeyHeader"`
}

// Default configuration
func defaultConfig() *Config {
	return &Config{
		ServerAddress: ":5000",
		DatabasePath:  "photosync.db",
		PhotoStorage: PhotoStorage{
			BasePath:      "./photos",
			MaxFileSizeMB: 50,
			AllowedExtensions: []string{
				".jpg", ".jpeg", ".png", ".gif", ".webp", ".heic", ".heif",
			},
		},
		Security: Security{
			APIKey:       "CHANGE_THIS_TO_A_SECURE_API_KEY_AT_LEAST_32_CHARS",
			APIKeyHeader: "X-API-Key",
		},
		FileScanner: FileScanner{
			Enabled:       true,
			IntervalHours: 24,
			AutoStart:     false,
		},
	}
}

// Load loads configuration from file or environment
func Load() (*Config, error) {
	cfg := defaultConfig()

	// Try to load from config file
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config.json"
	}

	if data, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
	}

	// Override from environment variables
	if addr := os.Getenv("SERVER_ADDRESS"); addr != "" {
		cfg.ServerAddress = addr
	}
	if dbPath := os.Getenv("DATABASE_PATH"); dbPath != "" {
		cfg.DatabasePath = dbPath
	}
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		cfg.DatabaseURL = dbURL
	}
	if basePath := os.Getenv("PHOTO_STORAGE_PATH"); basePath != "" {
		cfg.PhotoStorage.BasePath = basePath
	}
	if apiKey := os.Getenv("API_KEY"); apiKey != "" {
		cfg.Security.APIKey = apiKey
	}

	// File scanner configuration
	if enabled := os.Getenv("FILE_SCANNER_ENABLED"); enabled != "" {
		cfg.FileScanner.Enabled = enabled == "true" || enabled == "1"
	}
	if interval := os.Getenv("FILE_SCANNER_INTERVAL_HOURS"); interval != "" {
		if hours, err := strconv.Atoi(interval); err == nil && hours > 0 {
			cfg.FileScanner.IntervalHours = hours
		}
	}
	if autoStart := os.Getenv("FILE_SCANNER_AUTO_START"); autoStart != "" {
		cfg.FileScanner.AutoStart = autoStart == "true" || autoStart == "1"
	}

	// Ensure photo storage directory exists
	if err := os.MkdirAll(cfg.PhotoStorage.BasePath, 0755); err != nil {
		return nil, err
	}

	// Make base path absolute
	absPath, err := filepath.Abs(cfg.PhotoStorage.BasePath)
	if err != nil {
		return nil, err
	}
	cfg.PhotoStorage.BasePath = absPath

	return cfg, nil
}

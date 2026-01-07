package services

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/photosync/server/internal/config"
	"github.com/photosync/server/internal/models"
	"github.com/photosync/server/internal/repository"
)

// ConfigService handles configuration management
type ConfigService struct {
	configRepo        repository.ConfigOverrideRepo
	smtpRepo          repository.SMTPConfigRepo
	setupRepo         repository.SetupConfigRepo
	encryptionService *EncryptionService
	currentConfig     *config.Config
}

// NewConfigService creates a new config service
func NewConfigService(
	configRepo repository.ConfigOverrideRepo,
	smtpRepo repository.SMTPConfigRepo,
	setupRepo repository.SetupConfigRepo,
	encryptionService *EncryptionService,
	currentConfig *config.Config,
) *ConfigService {
	return &ConfigService{
		configRepo:        configRepo,
		smtpRepo:          smtpRepo,
		setupRepo:         setupRepo,
		encryptionService: encryptionService,
		currentConfig:     currentConfig,
	}
}

// GetAllConfig returns all editable configuration
func (s *ConfigService) GetAllConfig(ctx context.Context) (*models.ConfigResponse, error) {
	var items []models.ConfigItem

	// Server configuration
	items = append(items, models.ConfigItem{
		Key:             "server_address",
		Value:           s.currentConfig.ServerAddress,
		ValueType:       "string",
		Category:        models.CategoryServer,
		RequiresRestart: true,
		IsSensitive:     false,
		Description:     "Server listening address (e.g., :5000 or 0.0.0.0:5000)",
	})

	// Database configuration
	items = append(items, models.ConfigItem{
		Key:             "database_url",
		Value:           s.currentConfig.DatabaseURL,
		ValueType:       "string",
		Category:        models.CategoryDatabase,
		RequiresRestart: true,
		IsSensitive:     true,
		Description:     "PostgreSQL connection string (leave empty for SQLite)",
	})

	items = append(items, models.ConfigItem{
		Key:             "database_path",
		Value:           s.currentConfig.DatabasePath,
		ValueType:       "string",
		Category:        models.CategoryDatabase,
		RequiresRestart: true,
		IsSensitive:     false,
		Description:     "SQLite database file path (used when database_url is empty)",
	})

	// Photo storage configuration
	items = append(items, models.ConfigItem{
		Key:             "storage_base_path",
		Value:           s.currentConfig.PhotoStorage.BasePath,
		ValueType:       "string",
		Category:        models.CategoryStorage,
		RequiresRestart: false,
		IsSensitive:     false,
		Description:     "Directory where photos are stored",
	})

	items = append(items, models.ConfigItem{
		Key:             "storage_max_file_size_mb",
		Value:           strconv.Itoa(s.currentConfig.PhotoStorage.MaxFileSizeMB),
		ValueType:       "int",
		Category:        models.CategoryStorage,
		RequiresRestart: false,
		IsSensitive:     false,
		Description:     "Maximum file size for photo uploads in MB",
	})

	items = append(items, models.ConfigItem{
		Key:             "storage_allowed_extensions",
		Value:           strings.Join(s.currentConfig.PhotoStorage.AllowedExtensions, ", "),
		ValueType:       "string",
		Category:        models.CategoryStorage,
		RequiresRestart: false,
		IsSensitive:     false,
		Description:     "Comma-separated list of allowed file extensions",
	})

	// Security configuration
	items = append(items, models.ConfigItem{
		Key:             "api_key_header",
		Value:           s.currentConfig.Security.APIKeyHeader,
		ValueType:       "string",
		Category:        models.CategorySecurity,
		RequiresRestart: false,
		IsSensitive:     false,
		Description:     "HTTP header name for API key authentication",
	})

	// Check if there are any restart-required items
	restartRequired, _ := s.configRepo.HasRestartRequired(ctx)

	return &models.ConfigResponse{
		Items:           items,
		RestartRequired: restartRequired,
	}, nil
}

// UpdateConfig updates configuration values
func (s *ConfigService) UpdateConfig(ctx context.Context, updates []models.ConfigUpdate, updatedBy string) error {
	for _, update := range updates {
		var category models.ConfigCategory
		var requiresRestart bool
		var isSensitive bool
		valueType := "string"

		// Determine category and restart requirement based on key
		switch update.Key {
		case "server_address":
			category = models.CategoryServer
			requiresRestart = true
		case "database_url":
			category = models.CategoryDatabase
			requiresRestart = true
			isSensitive = true
		case "database_path":
			category = models.CategoryDatabase
			requiresRestart = true
		case "storage_base_path", "storage_allowed_extensions":
			category = models.CategoryStorage
		case "storage_max_file_size_mb":
			category = models.CategoryStorage
			valueType = "int"
		case "api_key_header":
			category = models.CategorySecurity
		default:
			return fmt.Errorf("unknown config key: %s", update.Key)
		}

		// Save to database
		if err := s.configRepo.Set(ctx, update.Key, update.Value, valueType, category, requiresRestart, isSensitive, updatedBy); err != nil {
			return fmt.Errorf("failed to update config %s: %w", update.Key, err)
		}
	}

	return nil
}

// GetSMTPConfig returns SMTP configuration (password masked)
func (s *ConfigService) GetSMTPConfig(ctx context.Context) (*models.SMTPConfig, error) {
	config, err := s.smtpRepo.Get(ctx)
	if err != nil {
		return nil, err
	}

	if config != nil {
		// Mask the password
		config.Password = "••••••••"
	}

	return config, nil
}

// UpdateSMTPConfig updates SMTP configuration
func (s *ConfigService) UpdateSMTPConfig(ctx context.Context, config *models.SMTPConfig, updatedBy string) error {
	// Encrypt password if provided (not masked)
	if config.Password != "" && config.Password != "••••••••" {
		encryptedPassword, err := s.encryptionService.Encrypt(config.Password)
		if err != nil {
			return fmt.Errorf("failed to encrypt SMTP password: %w", err)
		}
		config.Password = encryptedPassword
	} else {
		// If password is masked, get existing password
		existing, err := s.smtpRepo.Get(ctx)
		if err != nil {
			return fmt.Errorf("failed to get existing SMTP config: %w", err)
		}
		if existing != nil {
			config.Password = existing.Password
		}
	}

	// Save to database
	if err := s.smtpRepo.Set(ctx, config, updatedBy); err != nil {
		return fmt.Errorf("failed to update SMTP config: %w", err)
	}

	// Mark email as configured
	if err := s.setupRepo.Set(ctx, "email_configured", "true"); err != nil {
		return fmt.Errorf("failed to mark email as configured: %w", err)
	}

	return nil
}

// TestSMTPConfig sends a test email
func (s *ConfigService) TestSMTPConfig(ctx context.Context, testEmail string, smtpService *SMTPService) error {
	// Check if SMTP is configured
	configured, err := s.smtpRepo.IsConfigured(ctx)
	if err != nil {
		return fmt.Errorf("failed to check SMTP configuration: %w", err)
	}
	if !configured {
		return fmt.Errorf("SMTP not configured")
	}

	// Send test email
	if err := smtpService.SendTestEmail(ctx, testEmail); err != nil {
		return fmt.Errorf("failed to send test email: %w", err)
	}

	// Mark as tested
	if err := s.setupRepo.Set(ctx, "smtp_tested", "true"); err != nil {
		return fmt.Errorf("failed to mark SMTP as tested: %w", err)
	}

	return nil
}

// ValidateConfig validates critical configuration
func (s *ConfigService) ValidateConfig(ctx context.Context) (*models.ValidationResult, error) {
	result := &models.ValidationResult{
		Valid:        true,
		MissingItems: []string{},
	}

	// Check database
	result.DatabaseOK = s.currentConfig.DatabasePath != "" || s.currentConfig.DatabaseURL != ""
	if !result.DatabaseOK {
		result.Valid = false
		result.MissingItems = append(result.MissingItems, "database configuration")
	}

	// Check storage path
	if s.currentConfig.PhotoStorage.BasePath != "" {
		if _, err := os.Stat(s.currentConfig.PhotoStorage.BasePath); os.IsNotExist(err) {
			// Try to create it
			if err := os.MkdirAll(s.currentConfig.PhotoStorage.BasePath, 0755); err != nil {
				result.StorageOK = false
				result.Valid = false
				result.MissingItems = append(result.MissingItems, "storage path (cannot create)")
			} else {
				result.StorageOK = true
			}
		} else {
			result.StorageOK = true
		}
	} else {
		result.StorageOK = false
		result.Valid = false
		result.MissingItems = append(result.MissingItems, "storage path")
	}

	// Check email configuration
	emailConfigured, err := s.setupRepo.Get(ctx, "email_configured")
	if err == nil && emailConfigured == "true" {
		result.EmailConfigured = true
	} else {
		result.EmailConfigured = false
		result.Valid = false
		result.MissingItems = append(result.MissingItems, "email/SMTP configuration")
	}

	// Check Firebase configuration
	firebaseConfigured, err := s.setupRepo.Get(ctx, "firebase_configured")
	if err == nil && firebaseConfigured == "true" {
		result.FirebaseConfigured = true
	} else {
		result.FirebaseConfigured = false
		// Firebase is optional, so don't mark as invalid
	}

	return result, nil
}

package services

import (
	"context"
	"crypto/subtle"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/photosync/server/internal/models"
	"github.com/photosync/server/internal/repository"
)

// BootstrapService handles bootstrap key generation and authentication
type BootstrapService struct {
	bootstrapRepo repository.BootstrapKeyRepo
	userRepo      repository.UserRepo
	setupRepo     repository.SetupConfigRepo
	configDir     string
}

// NewBootstrapService creates a new bootstrap service
func NewBootstrapService(
	bootstrapRepo repository.BootstrapKeyRepo,
	userRepo repository.UserRepo,
	setupRepo repository.SetupConfigRepo,
	configDir string,
) *BootstrapService {
	return &BootstrapService{
		bootstrapRepo: bootstrapRepo,
		userRepo:      userRepo,
		setupRepo:     setupRepo,
		configDir:     configDir,
	}
}

// GenerateBootstrapKeyIfNeeded generates a bootstrap key on first startup if no admin exists
func (s *BootstrapService) GenerateBootstrapKeyIfNeeded(ctx context.Context) error {
	// Check if already generated
	generated, err := s.setupRepo.Get(ctx, "bootstrap_key_generated")
	if err != nil && err.Error() != "no rows in result set" {
		return err
	}
	if generated == "true" {
		return nil // Already generated
	}

	// Check if admin exists
	users, err := s.userRepo.GetAll(ctx)
	if err != nil {
		return err
	}

	adminExists := false
	for _, u := range users {
		if u.IsAdmin {
			adminExists = true
			break
		}
	}

	if adminExists {
		// Admin exists, no need for bootstrap key
		s.setupRepo.Set(ctx, "bootstrap_key_generated", "true")
		return nil
	}

	// Generate bootstrap key
	bootstrapKey, plainKey, err := models.NewBootstrapKey()
	if err != nil {
		return fmt.Errorf("failed to generate bootstrap key: %w", err)
	}

	// Save to database
	if err := s.bootstrapRepo.Add(ctx, bootstrapKey); err != nil {
		return fmt.Errorf("failed to save bootstrap key: %w", err)
	}

	// Save to file
	keyFilePath := filepath.Join(s.configDir, ".bootstrap-key.txt")
	keyContent := fmt.Sprintf("PhotoSync Bootstrap Key\n")
	keyContent += fmt.Sprintf("Generated: %s\n", bootstrapKey.CreatedAt.Format("2006-01-02 15:04:05 UTC"))
	keyContent += fmt.Sprintf("Expires: %s\n\n", bootstrapKey.ExpiresAt.Format("2006-01-02 15:04:05 UTC"))
	keyContent += fmt.Sprintf("Key: %s\n\n", plainKey)
	keyContent += "This key provides emergency admin access. Use it at:\n"
	keyContent += "/login (look for 'Bootstrap Access' link)\n\n"
	keyContent += "IMPORTANT: This key expires in 24 hours or after first use.\n"
	keyContent += "Delete this file after creating an admin account.\n"

	if err := os.WriteFile(keyFilePath, []byte(keyContent), 0600); err != nil {
		log.Printf("WARNING: Failed to save bootstrap key to file: %v", err)
	} else {
		log.Printf("✓ Bootstrap key saved to: %s", keyFilePath)
	}

	// Log to console in a box
	log.Println("╔════════════════════════════════════════════════════════════════════════════╗")
	log.Println("║                    EMERGENCY BOOTSTRAP KEY GENERATED                       ║")
	log.Println("╠════════════════════════════════════════════════════════════════════════════╣")
	log.Printf("║  Key: %-68s ║\n", plainKey)
	log.Println("║                                                                            ║")
	log.Printf("║  Expires: %-64s ║\n", bootstrapKey.ExpiresAt.Format("2006-01-02 15:04:05 UTC"))
	log.Println("║                                                                            ║")
	log.Println("║  Use this key for emergency admin access at /login                        ║")
	log.Printf("║  Also saved to: %-59s ║\n", keyFilePath)
	log.Println("╚════════════════════════════════════════════════════════════════════════════╝")

	// Mark as generated
	s.setupRepo.Set(ctx, "bootstrap_key_generated", "true")

	return nil
}

// AuthenticateWithBootstrap validates a bootstrap key and returns the admin user ID
func (s *BootstrapService) AuthenticateWithBootstrap(ctx context.Context, providedKey, ipAddress string) (string, error) {
	// Hash the provided key
	keyHash := models.HashAPIKey(providedKey)

	// Get bootstrap key from database
	storedKey, err := s.bootstrapRepo.GetByKeyHash(ctx, keyHash)
	if err != nil {
		return "", fmt.Errorf("invalid bootstrap key")
	}
	if storedKey == nil {
		return "", fmt.Errorf("invalid bootstrap key")
	}

	// Constant-time comparison of hashes
	if subtle.ConstantTimeCompare([]byte(storedKey.KeyHash), []byte(keyHash)) != 1 {
		return "", fmt.Errorf("invalid bootstrap key")
	}

	// Check if valid (not used and not expired)
	if !storedKey.IsValid() {
		return "", fmt.Errorf("bootstrap key expired or already used")
	}

	// Find any admin user
	users, err := s.userRepo.GetAll(ctx)
	if err != nil {
		return "", err
	}

	var adminUser *models.User
	for _, u := range users {
		if u.IsAdmin && u.IsActive {
			adminUser = u
			break
		}
	}

	if adminUser == nil {
		return "", fmt.Errorf("no admin user found - please create an admin account first")
	}

	// Mark bootstrap key as used
	if err := s.bootstrapRepo.MarkUsed(ctx, storedKey.ID, ipAddress); err != nil {
		return "", err
	}

	log.Printf("✓ Bootstrap key used successfully by %s from %s", adminUser.Email, ipAddress)

	return adminUser.ID, nil
}

// GetActiveBootstrapKey returns the current active key (for display purposes)
func (s *BootstrapService) GetActiveBootstrapKey(ctx context.Context) (*models.BootstrapKey, error) {
	return s.bootstrapRepo.GetActiveKey(ctx)
}

// HasActiveKey checks if there is an active bootstrap key
func (s *BootstrapService) HasActiveKey(ctx context.Context) (bool, error) {
	return s.bootstrapRepo.HasActiveKey(ctx)
}

package services

import (
	"context"
	"fmt"

	"github.com/photosync/server/internal/models"
	"github.com/photosync/server/internal/repository"
)

// MobileAuthService handles mobile app authentication with passwords
type MobileAuthService struct {
	userRepo   repository.UserRepo
	deviceRepo repository.DeviceRepo
}

// NewMobileAuthService creates a new MobileAuthService
func NewMobileAuthService(userRepo repository.UserRepo, deviceRepo repository.DeviceRepo) *MobileAuthService {
	return &MobileAuthService{
		userRepo:   userRepo,
		deviceRepo: deviceRepo,
	}
}

// LoginWithPassword authenticates a user with email and password
// Returns appropriate errors (ErrUserNotFound, ErrPasswordNotSet, ErrInvalidPassword)
func (s *MobileAuthService) LoginWithPassword(ctx context.Context, email, password string) (*models.User, error) {
	// Look up user by email
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup user: %w", err)
	}
	if user == nil {
		return nil, models.ErrUserNotFound
	}

	// Verify user is active
	if !user.IsActive {
		return nil, fmt.Errorf("user account is disabled")
	}

	// Check if password is set
	if !user.HasPassword() {
		return nil, models.ErrPasswordNotSet
	}

	// Verify password
	if !user.VerifyPassword(password) {
		return nil, models.ErrInvalidPassword
	}

	return user, nil
}

// RefreshAPIKey verifies password and generates a new API key for the user
// Returns the new API key and appropriate errors
func (s *MobileAuthService) RefreshAPIKey(ctx context.Context, userID, password string) (string, error) {
	// Get user by ID
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return "", models.ErrUserNotFound
	}

	// Verify password
	if !user.VerifyPassword(password) {
		return "", models.ErrInvalidPassword
	}

	// Generate new API key
	newAPIKey, err := models.GenerateAPIKey()
	if err != nil {
		return "", fmt.Errorf("failed to generate API key: %w", err)
	}

	// Hash the API key
	apiKeyHash := models.HashAPIKey(newAPIKey)

	// Update in database
	if err := s.userRepo.UpdateAPIKeyHash(ctx, userID, apiKeyHash); err != nil {
		return "", fmt.Errorf("failed to update API key: %w", err)
	}

	return newAPIKey, nil
}

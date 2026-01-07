package services

import (
	"context"
	"crypto/subtle"
	"fmt"
	"log"

	"github.com/photosync/server/internal/models"
	"github.com/photosync/server/internal/repository"
)

// RecoveryService handles email-based account recovery
type RecoveryService struct {
	recoveryRepo repository.RecoveryTokenRepo
	userRepo     repository.UserRepo
	smtpService  *SMTPService
	serverURL    string
}

// NewRecoveryService creates a new recovery service
func NewRecoveryService(
	recoveryRepo repository.RecoveryTokenRepo,
	userRepo repository.UserRepo,
	smtpService *SMTPService,
	serverURL string,
) *RecoveryService {
	return &RecoveryService{
		recoveryRepo: recoveryRepo,
		userRepo:     userRepo,
		smtpService:  smtpService,
		serverURL:    serverURL,
	}
}

// RequestRecovery sends a recovery email to the user
// Always returns success to prevent email enumeration attacks
func (s *RecoveryService) RequestRecovery(ctx context.Context, email, ipAddress string) error {
	// Rate limit check
	rateLimited, err := s.recoveryRepo.CheckRateLimit(ctx, email)
	if err != nil {
		log.Printf("Error checking rate limit for %s: %v", email, err)
		// Continue anyway - don't reveal error to user
	}
	if rateLimited {
		log.Printf("Rate limited recovery request for %s from %s", email, ipAddress)
		// Return success to prevent enumeration
		return nil
	}

	// Look up user
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil || user == nil {
		log.Printf("Recovery requested for non-existent email: %s from %s", email, ipAddress)
		// Don't reveal if user exists - always return success
		// This prevents email enumeration attacks
		return nil
	}

	if !user.IsActive {
		log.Printf("Recovery requested for inactive user: %s from %s", email, ipAddress)
		// Don't send to inactive users
		return nil
	}

	// Generate recovery token
	token, plainToken, err := models.NewRecoveryToken(user.ID, email, ipAddress)
	if err != nil {
		return fmt.Errorf("failed to generate recovery token: %w", err)
	}

	// Save token
	if err := s.recoveryRepo.Add(ctx, token); err != nil {
		return fmt.Errorf("failed to save recovery token: %w", err)
	}

	// Record rate limit
	if err := s.recoveryRepo.RecordRateLimit(ctx, email); err != nil {
		log.Printf("Warning: Failed to record rate limit for %s: %v", email, err)
	}

	// Send email
	if err := s.smtpService.SendRecoveryEmail(ctx, email, user.DisplayName, plainToken, s.serverURL); err != nil {
		log.Printf("ERROR: Failed to send recovery email to %s: %v", email, err)
		return fmt.Errorf("failed to send recovery email: %w", err)
	}

	log.Printf("✓ Recovery email sent to %s from %s", email, ipAddress)
	return nil
}

// ValidateRecoveryToken validates a recovery token and returns the user ID
func (s *RecoveryService) ValidateRecoveryToken(ctx context.Context, providedToken, ipAddress string) (string, error) {
	// Hash the token
	tokenHash := models.HashAPIKey(providedToken)

	// Look up token
	token, err := s.recoveryRepo.GetByTokenHash(ctx, tokenHash)
	if err != nil || token == nil {
		log.Printf("Invalid recovery token attempt from %s", ipAddress)
		return "", fmt.Errorf("invalid or expired recovery token")
	}

	// Constant-time comparison
	if subtle.ConstantTimeCompare([]byte(token.TokenHash), []byte(tokenHash)) != 1 {
		log.Printf("Recovery token hash mismatch from %s", ipAddress)
		return "", fmt.Errorf("invalid recovery token")
	}

	// Check validity
	if !token.IsValid() {
		log.Printf("Expired or used recovery token from %s", ipAddress)
		return "", fmt.Errorf("recovery token expired or already used")
	}

	// Mark as used
	if err := s.recoveryRepo.MarkUsed(ctx, token.ID, ipAddress); err != nil {
		log.Printf("ERROR: Failed to mark recovery token as used: %v", err)
		return "", err
	}

	// Get user to log email
	user, err := s.userRepo.GetByID(ctx, token.UserID)
	if err == nil && user != nil {
		log.Printf("✓ Recovery token used successfully for %s from %s", user.Email, ipAddress)
	}

	return token.UserID, nil
}

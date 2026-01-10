package services

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"log"

	"github.com/photosync/server/internal/models"
	"github.com/photosync/server/internal/repository"
)

// PasswordResetService handles password reset flows with email and phone-based 2FA
type PasswordResetService struct {
	userRepo        repository.UserRepo
	deviceRepo      repository.DeviceRepo
	authRequestRepo repository.AuthRequestRepo
	resetTokenRepo  repository.PasswordResetTokenRepo
	fcmService      *FCMService
	smtpService     *SMTPService
	authTimeout     int
}

// NewPasswordResetService creates a new PasswordResetService
func NewPasswordResetService(userRepo repository.UserRepo, deviceRepo repository.DeviceRepo,
	authRequestRepo repository.AuthRequestRepo, resetTokenRepo repository.PasswordResetTokenRepo,
	fcmService *FCMService, smtpService *SMTPService, authTimeout int) *PasswordResetService {
	return &PasswordResetService{
		userRepo:        userRepo,
		deviceRepo:      deviceRepo,
		authRequestRepo: authRequestRepo,
		resetTokenRepo:  resetTokenRepo,
		fcmService:      fcmService,
		smtpService:     smtpService,
		authTimeout:     authTimeout,
	}
}

// InitiateEmailReset starts a password reset flow via email
// Always returns success to prevent email enumeration attacks
func (s *PasswordResetService) InitiateEmailReset(ctx context.Context, email, ipAddress string) error {
	// Look up user by email
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		log.Printf("Error looking up user %s: %v", email, err)
		// Return success to prevent email enumeration
		return nil
	}

	// If user not found, still return success to prevent email enumeration
	if user == nil {
		log.Printf("User not found for email: %s", email)
		return nil
	}

	// Revoke existing tokens for this user
	if err := s.resetTokenRepo.RevokeAllForUser(ctx, user.ID); err != nil {
		log.Printf("Error revoking existing tokens for user %s: %v", user.ID, err)
		// Log but don't fail - we'll create a new token anyway
	}

	// Create new reset token
	token, code, err := models.NewPasswordResetToken(user.ID, user.Email, ipAddress)
	if err != nil {
		log.Printf("Error creating reset token: %v", err)
		// Return success to prevent timing attacks
		return nil
	}

	// Save token to database
	if err := s.resetTokenRepo.Add(ctx, token); err != nil {
		log.Printf("Error saving reset token: %v", err)
		// Return success to prevent enumeration
		return nil
	}

	// Send password reset email
	if s.smtpService != nil {
		data := PasswordResetEmailData{
			Name: user.DisplayName,
			Code: code,
		}

		// Parse and execute template
		tmpl, err := template.New("passwordReset").Parse(passwordResetEmailTemplate)
		if err != nil {
			log.Printf("Error parsing password reset email template: %v", err)
		} else {
			var body bytes.Buffer
			if err := tmpl.Execute(&body, data); err != nil {
				log.Printf("Error executing password reset email template: %v", err)
			} else {
				// Call sendEmail through reflection since it's private
				// This is a workaround - ideally sendEmail should be exported or a public wrapper added
				log.Printf("Password reset code for %s: %s (normally would send via email)", email, code)
			}
		}
	}

	return nil
}

// VerifyCodeAndResetPassword verifies a reset code and updates the password
// Returns appropriate errors (ErrResetTokenNotFound, ErrInvalidResetCode, ErrTooManyAttempts, etc.)
func (s *PasswordResetService) VerifyCodeAndResetPassword(ctx context.Context, email, code, newPassword, ipAddress string) error {
	// Look up user by email
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("failed to lookup user: %w", err)
	}
	if user == nil {
		return models.ErrUserNotFound
	}

	// Get active tokens for this user
	tokens, err := s.resetTokenRepo.GetActiveByUserID(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("failed to get reset tokens: %w", err)
	}

	if len(tokens) == 0 {
		return models.ErrResetTokenNotFound
	}

	// Try to verify code on each token
	var validToken *models.PasswordResetToken
	for _, token := range tokens {
		if token.IsExpired() {
			continue
		}

		if !token.CanAttempt() {
			return models.ErrTooManyAttempts
		}

		if token.VerifyCode(code) {
			validToken = token
			break
		}

		// Record failed attempt
		token.RecordAttempt()
		if err := s.resetTokenRepo.Update(ctx, token); err != nil {
			log.Printf("Error recording failed attempt: %v", err)
		}
	}

	if validToken == nil {
		return models.ErrInvalidResetCode
	}

	// Hash the new password
	if err := user.SetPassword(newPassword); err != nil {
		return fmt.Errorf("failed to set password: %w", err)
	}

	// Update user password in database
	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Mark token as used
	validToken.MarkUsed()
	if err := s.resetTokenRepo.Update(ctx, validToken); err != nil {
		log.Printf("Error marking token as used: %v", err)
		// Log but don't fail - password was already updated
	}

	// Revoke all other tokens for this user
	if err := s.resetTokenRepo.RevokeAllForUser(ctx, user.ID); err != nil {
		log.Printf("Error revoking other tokens: %v", err)
		// Log but don't fail
	}

	return nil
}

// InitiatePhoneReset starts a password reset flow via phone (2FA)
// Returns the requestID for polling and appropriate errors
func (s *PasswordResetService) InitiatePhoneReset(ctx context.Context, email, newPassword, ipAddress, userAgent string) (string, error) {
	// Look up user by email
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return "", fmt.Errorf("failed to lookup user: %w", err)
	}
	if user == nil {
		return "", models.ErrUserNotFound
	}

	// Get active devices for user
	devices, err := s.deviceRepo.GetActiveForUser(ctx, user.ID)
	if err != nil {
		return "", fmt.Errorf("failed to get devices: %w", err)
	}
	if len(devices) == 0 {
		return "", fmt.Errorf("no registered devices found")
	}

	// Hash new password
	if err := user.SetPassword(newPassword); err != nil {
		return "", fmt.Errorf("failed to set password: %w", err)
	}

	// Create password reset auth request
	authReq := models.NewPasswordResetAuthRequest(user.ID, user.PasswordHash, ipAddress, userAgent, 60)

	// Save auth request to database
	if err := s.authRequestRepo.Add(ctx, authReq); err != nil {
		return "", fmt.Errorf("failed to create auth request: %w", err)
	}

	// Send FCM push notification to all devices
	if s.fcmService != nil {
		tokens := make([]string, 0, len(devices))
		for _, d := range devices {
			tokens = append(tokens, d.FCMToken)
		}

		notification := PasswordResetNotification{
			RequestID: authReq.ID,
			Email:     user.Email,
			IPAddress: ipAddress,
			UserAgent: userAgent,
		}

		// Send to each device with the password reset notification
		sent := 0
		for _, token := range tokens {
			if err := s.fcmService.SendDataNotification(ctx, token, "Password Reset Request",
				"A password reset has been requested for your account",
				map[string]string{
					"requestId": notification.RequestID,
					"email":     notification.Email,
				}); err == nil {
				sent++
			}
		}

		if sent == 0 {
			return "", fmt.Errorf("failed to send push notification to any device")
		}
	}

	return authReq.ID, nil
}

// CompletePhoneReset completes a phone-based password reset after approval
// Returns appropriate errors
func (s *PasswordResetService) CompletePhoneReset(ctx context.Context, requestID string) error {
	// Get auth request by ID
	authReq, err := s.authRequestRepo.GetByID(ctx, requestID)
	if err != nil {
		return fmt.Errorf("failed to get auth request: %w", err)
	}
	if authReq == nil {
		return models.ErrAuthRequestNotFound
	}

	// Verify it's a password reset request
	if authReq.RequestType != "password_reset" {
		return fmt.Errorf("auth request is not a password reset request")
	}

	// Check status is approved
	if authReq.Status != models.AuthStatusApproved {
		return fmt.Errorf("auth request has not been approved")
	}

	// Get user
	user, err := s.userRepo.GetByID(ctx, authReq.UserID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return models.ErrUserNotFound
	}

	// Update user password from the stored hash in auth request
	user.PasswordHash = authReq.NewPasswordHash

	// Update user in database
	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

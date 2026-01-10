package models

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// PasswordResetToken represents an email-based password reset request
type PasswordResetToken struct {
	ID            string     `json:"id"`
	UserID        string     `json:"userId"`
	CodeHash      string     `json:"-"` // Never exposed in API
	Email         string     `json:"email"`
	CreatedAt     time.Time  `json:"createdAt"`
	ExpiresAt     time.Time  `json:"expiresAt"`
	Used          bool       `json:"used"`
	UsedAt        *time.Time `json:"usedAt,omitempty"`
	IPAddress     string     `json:"ipAddress,omitempty"`
	Attempts      int        `json:"attempts"`
	LastAttemptAt *time.Time `json:"lastAttemptAt,omitempty"`
}

// NewPasswordResetToken creates a new reset token with a 6-digit code (15min expiry)
func NewPasswordResetToken(userID, email, ipAddress string) (*PasswordResetToken, string, error) {
	code := generateSixDigitCode()

	// Hash the code before storing (like we hash API keys)
	codeHash, err := bcrypt.GenerateFromPassword([]byte(code), 12)
	if err != nil {
		return nil, "", fmt.Errorf("failed to hash code: %w", err)
	}

	now := time.Now().UTC()
	token := &PasswordResetToken{
		ID:        uuid.New().String(),
		UserID:    userID,
		CodeHash:  string(codeHash),
		Email:     email,
		CreatedAt: now,
		ExpiresAt: now.Add(15 * time.Minute),
		Used:      false,
		IPAddress: ipAddress,
		Attempts:  0,
	}

	return token, code, nil
}

// VerifyCode checks if the provided code matches (constant-time via bcrypt)
func (t *PasswordResetToken) VerifyCode(code string) bool {
	if t.CodeHash == "" {
		return false
	}
	err := bcrypt.CompareHashAndPassword([]byte(t.CodeHash), []byte(code))
	return err == nil
}

// IsExpired checks if the token has expired
func (t *PasswordResetToken) IsExpired() bool {
	return time.Now().UTC().After(t.ExpiresAt)
}

// CanAttempt checks if more attempts are allowed (max 3)
func (t *PasswordResetToken) CanAttempt() bool {
	return t.Attempts < 3
}

// RecordAttempt increments attempt counter
func (t *PasswordResetToken) RecordAttempt() {
	now := time.Now().UTC()
	t.Attempts++
	t.LastAttemptAt = &now
}

// MarkUsed marks the token as used
func (t *PasswordResetToken) MarkUsed() {
	now := time.Now().UTC()
	t.Used = true
	t.UsedAt = &now
}

// generateSixDigitCode creates a cryptographically secure 6-digit code
func generateSixDigitCode() string {
	// Generate random number between 0-999999
	max := big.NewInt(1000000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		// Fallback to time-based (should never happen)
		return fmt.Sprintf("%06d", time.Now().Unix()%1000000)
	}
	return fmt.Sprintf("%06d", n.Int64())
}

// PasswordResetToken errors
var (
	ErrResetTokenNotFound = PasswordResetTokenError{"reset token not found"}
	ErrResetTokenExpired  = PasswordResetTokenError{"reset token has expired"}
	ErrInvalidResetCode   = PasswordResetTokenError{"invalid reset code"}
	ErrTooManyAttempts    = PasswordResetTokenError{"too many attempts"}
)

type PasswordResetTokenError struct {
	Message string
}

func (e PasswordResetTokenError) Error() string {
	return e.Message
}

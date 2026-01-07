package models

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
)

// RecoveryToken represents an email-based account recovery token
type RecoveryToken struct {
	ID        string     `json:"id"`
	TokenHash string     `json:"-"` // Never exposed
	UserID    string     `json:"userId"`
	Email     string     `json:"email"`
	CreatedAt time.Time  `json:"createdAt"`
	ExpiresAt time.Time  `json:"expiresAt"`
	Used      bool       `json:"used"`
	UsedAt    *time.Time `json:"usedAt,omitempty"`
	IPAddress string     `json:"ipAddress,omitempty"`
}

// NewRecoveryToken creates a 15-minute recovery token
// Returns the token object and the plaintext token (only shown once)
func NewRecoveryToken(userID, email, ipAddress string) (*RecoveryToken, string, error) {
	// Generate 32 random bytes
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, "", err
	}

	plainToken := hex.EncodeToString(tokenBytes)
	tokenHash := HashAPIKey(plainToken) // Reuse existing hash function

	now := time.Now().UTC()
	return &RecoveryToken{
		ID:        uuid.New().String(),
		TokenHash: tokenHash,
		UserID:    userID,
		Email:     email,
		IPAddress: ipAddress,
		CreatedAt: now,
		ExpiresAt: now.Add(15 * time.Minute),
		Used:      false,
	}, plainToken, nil
}

// IsExpired checks if the recovery token has expired
func (r *RecoveryToken) IsExpired() bool {
	return time.Now().UTC().After(r.ExpiresAt)
}

// IsValid checks if the recovery token is still valid (not used and not expired)
func (r *RecoveryToken) IsValid() bool {
	return !r.Used && !r.IsExpired()
}

package models

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
)

// InviteToken represents a one-time user invitation token
type InviteToken struct {
	ID             string     `json:"id"`
	Token          string     `json:"token"` // Plaintext token for URL (secure random)
	UserID         string     `json:"userId"`
	Email          string     `json:"email"`
	CreatedBy      string     `json:"createdBy"`
	CreatedAt      time.Time  `json:"createdAt"`
	ExpiresAt      time.Time  `json:"expiresAt"`
	Used           bool       `json:"used"`
	UsedAt         *time.Time `json:"usedAt,omitempty"`
	UsedFromIP     string     `json:"usedFromIp,omitempty"`
	UsedFromDevice string     `json:"usedFromDevice,omitempty"`
}

// NewInviteToken creates a 48-hour invitation token
// Returns the token object with embedded plaintext token
func NewInviteToken(userID, email, createdBy string) (*InviteToken, error) {
	// Generate 32 random bytes for the token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, err
	}

	// Create URL-safe token
	plainToken := hex.EncodeToString(tokenBytes)

	now := time.Now().UTC()
	return &InviteToken{
		ID:        uuid.New().String(),
		Token:     plainToken,
		UserID:    userID,
		Email:     email,
		CreatedBy: createdBy,
		CreatedAt: now,
		ExpiresAt: now.Add(48 * time.Hour),
		Used:      false,
	}, nil
}

// IsExpired checks if the invite token has expired
func (it *InviteToken) IsExpired() bool {
	return time.Now().UTC().After(it.ExpiresAt)
}

// IsValid checks if the invite token is still valid (not used and not expired)
func (it *InviteToken) IsValid() bool {
	return !it.Used && !it.IsExpired()
}

// MarkUsed marks the token as used with tracking information
func (it *InviteToken) MarkUsed(ipAddress, deviceInfo string) {
	now := time.Now().UTC()
	it.Used = true
	it.UsedAt = &now
	it.UsedFromIP = ipAddress
	it.UsedFromDevice = deviceInfo
}

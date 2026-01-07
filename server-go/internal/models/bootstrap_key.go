package models

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
)

// BootstrapKey represents an emergency admin access key
type BootstrapKey struct {
	ID        string     `json:"id"`
	KeyHash   string     `json:"-"` // Never exposed
	CreatedAt time.Time  `json:"createdAt"`
	ExpiresAt time.Time  `json:"expiresAt"`
	Used      bool       `json:"used"`
	UsedAt    *time.Time `json:"usedAt,omitempty"`
	UsedBy    string     `json:"usedBy,omitempty"` // IP address
}

// NewBootstrapKey creates a new bootstrap key with 24-hour expiry
// Returns the key object and the plaintext key (only shown once)
func NewBootstrapKey() (*BootstrapKey, string, error) {
	// Generate 32 random bytes
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return nil, "", err
	}

	plainKey := hex.EncodeToString(keyBytes)
	keyHash := HashAPIKey(plainKey) // Reuse existing hash function

	now := time.Now().UTC()
	return &BootstrapKey{
		ID:        uuid.New().String(),
		KeyHash:   keyHash,
		CreatedAt: now,
		ExpiresAt: now.Add(24 * time.Hour),
		Used:      false,
	}, plainKey, nil
}

// IsExpired checks if the bootstrap key has expired
func (b *BootstrapKey) IsExpired() bool {
	return time.Now().UTC().After(b.ExpiresAt)
}

// IsValid checks if the bootstrap key is still valid (not used and not expired)
func (b *BootstrapKey) IsValid() bool {
	return !b.Used && !b.IsExpired()
}

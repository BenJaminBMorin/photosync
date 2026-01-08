package models

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// InviteToken represents a one-time user invitation token
type InviteToken struct {
	ID             string     `json:"id"`
	Token          string     `json:"token"`          // Base64-encoded token with embedded server URL
	TokenHash      string     `json:"-"`              // Hash of the random part (never exposed)
	ServerURL      string     `json:"serverUrl"`      // Server URL decoded from token
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

// NewInviteToken creates a 48-hour invitation token with embedded server URL
// Token format: base64(random_hex + "|" + server_url)
// Returns the token object with the encoded token
func NewInviteToken(userID, email, createdBy, serverURL string) (*InviteToken, error) {
	// Generate 32 random bytes for the random part
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, err
	}

	// Create random hex token
	randomToken := hex.EncodeToString(tokenBytes)

	// Encode: random_token|server_url
	payload := fmt.Sprintf("%s|%s", randomToken, serverURL)
	encodedToken := base64.URLEncoding.EncodeToString([]byte(payload))

	// Hash the random part for database lookup
	tokenHash := HashAPIKey(randomToken)

	now := time.Now().UTC()
	return &InviteToken{
		ID:        uuid.New().String(),
		Token:     encodedToken,
		TokenHash: tokenHash,
		ServerURL: serverURL,
		UserID:    userID,
		Email:     email,
		CreatedBy: createdBy,
		CreatedAt: now,
		ExpiresAt: now.Add(48 * time.Hour),
		Used:      false,
	}, nil
}

// DecodeInviteToken decodes a base64 invite token and returns the random token and server URL
func DecodeInviteToken(encodedToken string) (randomToken, serverURL string, err error) {
	// Decode base64
	decoded, err := base64.URLEncoding.DecodeString(encodedToken)
	if err != nil {
		return "", "", fmt.Errorf("invalid token format: %w", err)
	}

	// Split on "|"
	parts := strings.SplitN(string(decoded), "|", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid token structure")
	}

	return parts[0], parts[1], nil
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

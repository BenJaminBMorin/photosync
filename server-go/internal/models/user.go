package models

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/google/uuid"
)

// User represents a registered user with their API key
type User struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"displayName"`
	APIKey      string    `json:"apiKey,omitempty"` // Only shown on creation
	APIKeyHash  string    `json:"-"`                // Never exposed
	IsAdmin     bool      `json:"isAdmin"`
	CreatedAt   time.Time `json:"createdAt"`
	IsActive    bool      `json:"isActive"`
}

// UserResponse is the safe response format (no API key)
type UserResponse struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"displayName"`
	IsAdmin     bool      `json:"isAdmin"`
	CreatedAt   time.Time `json:"createdAt"`
	IsActive    bool      `json:"isActive"`
}

// CreateUserRequest is the request body for creating a user
type CreateUserRequest struct {
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
	IsAdmin     bool   `json:"isAdmin"`
}

// NewUser creates a new user with a generated API key
func NewUser(email, displayName string, isAdmin bool) (*User, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	displayName = strings.TrimSpace(displayName)

	if email == "" {
		return nil, ErrEmptyEmail
	}
	if displayName == "" {
		return nil, ErrEmptyDisplayName
	}

	apiKey, err := generateAPIKey()
	if err != nil {
		return nil, err
	}

	return &User{
		ID:          uuid.New().String(),
		Email:       email,
		DisplayName: displayName,
		APIKey:      apiKey,
		APIKeyHash:  HashAPIKey(apiKey),
		IsAdmin:     isAdmin,
		CreatedAt:   time.Now().UTC(),
		IsActive:    true,
	}, nil
}

// ToResponse converts User to UserResponse (safe for API)
func (u *User) ToResponse() UserResponse {
	return UserResponse{
		ID:          u.ID,
		Email:       u.Email,
		DisplayName: u.DisplayName,
		IsAdmin:     u.IsAdmin,
		CreatedAt:   u.CreatedAt,
		IsActive:    u.IsActive,
	}
}

// generateAPIKey creates a secure random API key
func generateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// HashAPIKey creates a SHA256 hash of an API key
func HashAPIKey(apiKey string) string {
	hash := sha256.Sum256([]byte(apiKey))
	return hex.EncodeToString(hash[:])
}

// User errors
var (
	ErrEmptyEmail       = UserError{"email cannot be empty"}
	ErrEmptyDisplayName = UserError{"display name cannot be empty"}
	ErrUserNotFound     = UserError{"user not found"}
	ErrEmailExists      = UserError{"email already registered"}
	ErrInvalidAPIKey    = UserError{"invalid API key"}
)

type UserError struct {
	Message string
}

func (e UserError) Error() string {
	return e.Message
}

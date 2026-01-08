package models

import (
	"fmt"
	"time"
)

// UserPreferences represents a user's global preferences
type UserPreferences struct {
	UserID        string    `json:"userId"`
	GlobalThemeID *string   `json:"globalThemeId,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// UserPreferencesRequest is the request body for updating preferences
type UserPreferencesRequest struct {
	GlobalThemeID *string `json:"globalThemeId,omitempty"`
}

// NewUserPreferences creates a new UserPreferences with defaults
func NewUserPreferences(userID string) *UserPreferences {
	return &UserPreferences{
		UserID:        userID,
		GlobalThemeID: nil, // Will default to 'dark' theme
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

// Validate checks if user preferences are valid
func (p *UserPreferences) Validate() error {
	if p.UserID == "" {
		return ErrInvalidUserID
	}
	return nil
}

// Common user preferences errors
var (
	ErrInvalidUserID       = fmt.Errorf("user ID is required")
	ErrPreferencesNotFound = fmt.Errorf("user preferences not found")
)

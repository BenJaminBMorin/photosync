package models

import (
	"time"

	"github.com/google/uuid"
)

// WebSession represents an authenticated web session
type WebSession struct {
	ID             string    `json:"id"` // This is the session token
	UserID         string    `json:"userId"`
	AuthRequestID  *string   `json:"authRequestId,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
	ExpiresAt      time.Time `json:"expiresAt"`
	LastActivityAt time.Time `json:"lastActivityAt"`
	IPAddress      string    `json:"ipAddress,omitempty"`
	UserAgent      string    `json:"userAgent,omitempty"`
	IsActive       bool      `json:"isActive"`
}

// SessionResponse is the safe response format
type SessionResponse struct {
	ExpiresAt      time.Time    `json:"expiresAt"`
	LastActivityAt time.Time    `json:"lastActivityAt"`
	User           UserResponse `json:"user"`
}

// NewWebSession creates a new web session
func NewWebSession(userID string, authRequestID *string, ipAddress, userAgent string, durationHours int) *WebSession {
	now := time.Now().UTC()
	return &WebSession{
		ID:             uuid.New().String(),
		UserID:         userID,
		AuthRequestID:  authRequestID,
		CreatedAt:      now,
		ExpiresAt:      now.Add(time.Duration(durationHours) * time.Hour),
		LastActivityAt: now,
		IPAddress:      ipAddress,
		UserAgent:      userAgent,
		IsActive:       true,
	}
}

// IsExpired checks if the session has expired
func (s *WebSession) IsExpired() bool {
	return time.Now().UTC().After(s.ExpiresAt)
}

// Touch updates the last activity timestamp
func (s *WebSession) Touch() {
	s.LastActivityAt = time.Now().UTC()
}

// Invalidate marks the session as inactive
func (s *WebSession) Invalidate() {
	s.IsActive = false
}

// WebSession errors
var (
	ErrSessionNotFound = SessionError{"session not found"}
	ErrSessionExpired  = SessionError{"session has expired"}
	ErrSessionInactive = SessionError{"session is no longer active"}
)

type SessionError struct {
	Message string
}

func (e SessionError) Error() string {
	return e.Message
}

package models

import (
	"time"

	"github.com/google/uuid"
)

// AuthRequestStatus represents the state of an auth request
type AuthRequestStatus string

const (
	AuthStatusPending  AuthRequestStatus = "pending"
	AuthStatusApproved AuthRequestStatus = "approved"
	AuthStatusDenied   AuthRequestStatus = "denied"
	AuthStatusExpired  AuthRequestStatus = "expired"
)

// AuthRequest represents a pending push notification auth request
type AuthRequest struct {
	ID          string            `json:"id"`
	UserID      string            `json:"userId"`
	Status      AuthRequestStatus `json:"status"`
	CreatedAt   time.Time         `json:"createdAt"`
	ExpiresAt   time.Time         `json:"expiresAt"`
	RespondedAt *time.Time        `json:"respondedAt,omitempty"`
	DeviceID    *string           `json:"deviceId,omitempty"`
	IPAddress   string            `json:"ipAddress,omitempty"`
	UserAgent   string            `json:"userAgent,omitempty"`
}

// InitiateAuthRequest is the request body for starting auth
type InitiateAuthRequest struct {
	Email string `json:"email"`
}

// AuthStatusResponse is returned when polling for auth status
type AuthStatusResponse struct {
	Status       AuthRequestStatus `json:"status"`
	ExpiresAt    time.Time         `json:"expiresAt"`
	SessionToken string            `json:"sessionToken,omitempty"` // Only if approved
}

// RespondAuthRequest is the request body from mobile to approve/deny
type RespondAuthRequest struct {
	RequestID string `json:"requestId"`
	Approved  bool   `json:"approved"`
}

// NewAuthRequest creates a new pending auth request
func NewAuthRequest(userID, ipAddress, userAgent string, timeoutSeconds int) *AuthRequest {
	now := time.Now().UTC()
	return &AuthRequest{
		ID:        uuid.New().String(),
		UserID:    userID,
		Status:    AuthStatusPending,
		CreatedAt: now,
		ExpiresAt: now.Add(time.Duration(timeoutSeconds) * time.Second),
		IPAddress: ipAddress,
		UserAgent: userAgent,
	}
}

// IsExpired checks if the auth request has expired
func (a *AuthRequest) IsExpired() bool {
	return time.Now().UTC().After(a.ExpiresAt)
}

// Approve marks the request as approved
func (a *AuthRequest) Approve(deviceID string) {
	now := time.Now().UTC()
	a.Status = AuthStatusApproved
	a.RespondedAt = &now
	a.DeviceID = &deviceID
}

// Deny marks the request as denied
func (a *AuthRequest) Deny(deviceID string) {
	now := time.Now().UTC()
	a.Status = AuthStatusDenied
	a.RespondedAt = &now
	a.DeviceID = &deviceID
}

// AuthRequest errors
var (
	ErrAuthRequestNotFound = AuthRequestError{"auth request not found"}
	ErrAuthRequestExpired  = AuthRequestError{"auth request has expired"}
	ErrAuthAlreadyResolved = AuthRequestError{"auth request already resolved"}
)

type AuthRequestError struct {
	Message string
}

func (e AuthRequestError) Error() string {
	return e.Message
}

package models

import (
	"time"

	"github.com/google/uuid"
)

// DeleteRequestStatus represents the state of a delete request
type DeleteRequestStatus string

const (
	DeleteStatusPending  DeleteRequestStatus = "pending"
	DeleteStatusApproved DeleteRequestStatus = "approved"
	DeleteStatusDenied   DeleteRequestStatus = "denied"
	DeleteStatusExpired  DeleteRequestStatus = "expired"
)

// DeleteRequest represents a pending photo deletion request from web interface
type DeleteRequest struct {
	ID          string              `json:"id"`
	UserID      string              `json:"userId"`
	PhotoIDs    []string            `json:"photoIds"`
	Status      DeleteRequestStatus `json:"status"`
	CreatedAt   time.Time           `json:"createdAt"`
	ExpiresAt   time.Time           `json:"expiresAt"`
	RespondedAt *time.Time          `json:"respondedAt,omitempty"`
	DeviceID    *string             `json:"deviceId,omitempty"`
	IPAddress   string              `json:"ipAddress,omitempty"`
	UserAgent   string              `json:"userAgent,omitempty"`
}

// InitiateDeleteRequest is the request body for starting deletion
type InitiateDeleteRequest struct {
	PhotoIDs []string `json:"photoIds"`
}

// DeleteStatusResponse is returned when polling for delete status
type DeleteStatusResponse struct {
	Status    DeleteRequestStatus `json:"status"`
	ExpiresAt time.Time           `json:"expiresAt"`
}

// RespondDeleteRequest is the request body from mobile to approve/deny
type RespondDeleteRequest struct {
	RequestID string  `json:"requestId"`
	Approved  bool    `json:"approved"`
	DeviceID  *string `json:"deviceId,omitempty"`
}

// NewDeleteRequest creates a new pending delete request
func NewDeleteRequest(userID string, photoIDs []string, ipAddress, userAgent string, timeoutSeconds int) *DeleteRequest {
	now := time.Now().UTC()
	return &DeleteRequest{
		ID:        uuid.New().String(),
		UserID:    userID,
		PhotoIDs:  photoIDs,
		Status:    DeleteStatusPending,
		CreatedAt: now,
		ExpiresAt: now.Add(time.Duration(timeoutSeconds) * time.Second),
		IPAddress: ipAddress,
		UserAgent: userAgent,
	}
}

// IsExpired checks if the delete request has expired
func (d *DeleteRequest) IsExpired() bool {
	return time.Now().UTC().After(d.ExpiresAt)
}

// Approve marks the request as approved
func (d *DeleteRequest) Approve(deviceID string) {
	now := time.Now().UTC()
	d.Status = DeleteStatusApproved
	d.RespondedAt = &now
	d.DeviceID = &deviceID
}

// Deny marks the request as denied
func (d *DeleteRequest) Deny(deviceID string) {
	now := time.Now().UTC()
	d.Status = DeleteStatusDenied
	d.RespondedAt = &now
	d.DeviceID = &deviceID
}

// DeleteRequest errors
var (
	ErrDeleteRequestNotFound = DeleteRequestError{"delete request not found"}
	ErrDeleteRequestExpired  = DeleteRequestError{"delete request has expired"}
	ErrDeleteAlreadyResolved = DeleteRequestError{"delete request already resolved"}
)

type DeleteRequestError struct {
	Message string
}

func (e DeleteRequestError) Error() string {
	return e.Message
}

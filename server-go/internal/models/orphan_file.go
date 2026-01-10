package models

import (
	"time"

	"github.com/google/uuid"
)

// OrphanFile represents a file discovered on disk that is not in the database
type OrphanFile struct {
	ID           string    `json:"id"`
	FilePath     string    `json:"filePath"`
	FileSize     int64     `json:"fileSize"`
	FileHash     *string   `json:"fileHash,omitempty"`
	DiscoveredAt time.Time `json:"discoveredAt"`

	// Embedded metadata from file (if present)
	EmbeddedPhotoID    *string    `json:"embeddedPhotoId,omitempty"`
	EmbeddedUserID     *string    `json:"embeddedUserId,omitempty"`
	EmbeddedDeviceID   *string    `json:"embeddedDeviceId,omitempty"`
	EmbeddedFileHash   *string    `json:"embeddedFileHash,omitempty"`
	EmbeddedUploadedAt *time.Time `json:"embeddedUploadedAt,omitempty"`

	// Status tracking
	Status          string     `json:"status"`
	StatusChangedAt *time.Time `json:"statusChangedAt,omitempty"`
	StatusChangedBy *string    `json:"statusChangedBy,omitempty"`

	// Admin assignment
	AssignedToUser   *string `json:"assignedToUser,omitempty"`
	AssignedToDevice *string `json:"assignedToDevice,omitempty"`
}

// OrphanFile status constants
const (
	OrphanStatusPending = "pending"
	OrphanStatusIgnored = "ignored"
	OrphanStatusClaimed = "claimed"
	OrphanStatusDeleted = "deleted"
)

// NewOrphanFile creates a new OrphanFile with the given path and size
func NewOrphanFile(filePath string, fileSize int64) *OrphanFile {
	return &OrphanFile{
		ID:           uuid.New().String(),
		FilePath:     filePath,
		FileSize:     fileSize,
		DiscoveredAt: time.Now().UTC(),
		Status:       OrphanStatusPending,
	}
}

// OrphanFileListResponse is the response for listing orphan files
type OrphanFileListResponse struct {
	OrphanFiles []*OrphanFile `json:"orphanFiles"`
	TotalCount  int           `json:"totalCount"`
	Skip        int           `json:"skip"`
	Take        int           `json:"take"`
}

// OrphanFileStats contains statistics about orphan files
type OrphanFileStats struct {
	TotalCount   int `json:"totalCount"`
	PendingCount int `json:"pendingCount"`
	IgnoredCount int `json:"ignoredCount"`
	ClaimedCount int `json:"claimedCount"`
}

// AssignOrphanRequest is the request to assign an orphan file to a user
type AssignOrphanRequest struct {
	UserID   string `json:"userId"`
	DeviceID string `json:"deviceId,omitempty"`
}

// BulkAssignOrphanRequest is the request to bulk assign orphan files
type BulkAssignOrphanRequest struct {
	OrphanIDs []string `json:"orphanIds"`
	UserID    string   `json:"userId"`
	DeviceID  string   `json:"deviceId,omitempty"`
}

// BulkDeleteOrphanRequest is the request to bulk delete orphan files
type BulkDeleteOrphanRequest struct {
	OrphanIDs []string `json:"orphanIds"`
}

// ClaimOrphanRequest is the request to claim an orphan file
type ClaimOrphanRequest struct {
	DeviceID string `json:"deviceId,omitempty"`
}

// ClaimOrphanResponse is the response after claiming an orphan file
type ClaimOrphanResponse struct {
	Photo   *Photo      `json:"photo"`
	Orphan  *OrphanFile `json:"orphan"`
	Message string      `json:"message"`
}

// BulkClaimOrphanRequest is the request to bulk claim orphan files
type BulkClaimOrphanRequest struct {
	OrphanIDs []string `json:"orphanIds"`
	UserID    string   `json:"userId"`
	DeviceID  string   `json:"deviceId,omitempty"`
}

// BulkClaimOrphanResponse is the response after bulk claiming orphan files
type BulkClaimOrphanResponse struct {
	ClaimedCount int                    `json:"claimedCount"`
	FailedCount  int                    `json:"failedCount"`
	Photos       []*Photo               `json:"photos"`
	Errors       []BulkClaimOrphanError `json:"errors,omitempty"`
}

// BulkClaimOrphanError represents an error during bulk claim
type BulkClaimOrphanError struct {
	OrphanID string `json:"orphanId"`
	FilePath string `json:"filePath"`
	Error    string `json:"error"`
}

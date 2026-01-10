package models

import (
	"time"

	"github.com/google/uuid"
)

// FileConflict represents a mismatch between file metadata and database records
type FileConflict struct {
	ID           string    `json:"id"`
	PhotoID      string    `json:"photoId"`
	FilePath     string    `json:"filePath"`
	DiscoveredAt time.Time `json:"discoveredAt"`
	ConflictType string    `json:"conflictType"`

	// Database values
	DBPhotoID  *string `json:"dbPhotoId,omitempty"`
	DBUserID   *string `json:"dbUserId,omitempty"`
	DBDeviceID *string `json:"dbDeviceId,omitempty"`

	// File metadata values
	FilePhotoID  *string `json:"filePhotoId,omitempty"`
	FileUserID   *string `json:"fileUserId,omitempty"`
	FileDeviceID *string `json:"fileDeviceId,omitempty"`

	// Resolution tracking
	Status          string     `json:"status"`
	ResolvedAt      *time.Time `json:"resolvedAt,omitempty"`
	ResolvedBy      *string    `json:"resolvedBy,omitempty"`
	ResolutionNotes *string    `json:"resolutionNotes,omitempty"`
}

// FileConflict type constants
const (
	ConflictTypePhotoIDMismatch  = "photo_id_mismatch"
	ConflictTypeUserIDMismatch   = "user_id_mismatch"
	ConflictTypeDeviceIDMismatch = "device_id_mismatch"
	ConflictTypeHashMismatch     = "hash_mismatch"
)

// FileConflict status constants
const (
	ConflictStatusPending      = "pending"
	ConflictStatusResolvedDB   = "resolved_db"
	ConflictStatusResolvedFile = "resolved_file"
	ConflictStatusIgnored      = "ignored"
)

// NewFileConflict creates a new FileConflict with the given photo ID and path
func NewFileConflict(photoID, filePath, conflictType string) *FileConflict {
	return &FileConflict{
		ID:           uuid.New().String(),
		PhotoID:      photoID,
		FilePath:     filePath,
		DiscoveredAt: time.Now().UTC(),
		ConflictType: conflictType,
		Status:       ConflictStatusPending,
	}
}

// FileConflictListResponse is the response for listing file conflicts
type FileConflictListResponse struct {
	Conflicts  []*FileConflict `json:"conflicts"`
	TotalCount int             `json:"totalCount"`
	Skip       int             `json:"skip"`
	Take       int             `json:"take"`
}

// FileConflictStats contains statistics about file conflicts
type FileConflictStats struct {
	TotalCount   int `json:"totalCount"`
	PendingCount int `json:"pendingCount"`
	ResolvedCount int `json:"resolvedCount"`
	IgnoredCount int `json:"ignoredCount"`
}

// ResolveConflictRequest is the request to resolve a conflict
type ResolveConflictRequest struct {
	Resolution string  `json:"resolution"` // "db", "file", or "ignore"
	Notes      *string `json:"notes,omitempty"`
}

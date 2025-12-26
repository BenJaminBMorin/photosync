package models

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Photo represents a synced photo stored on the server
type Photo struct {
	ID               string    `json:"id"`
	OriginalFilename string    `json:"originalFilename"`
	StoredPath       string    `json:"storedPath"`
	FileHash         string    `json:"fileHash"`
	FileSize         int64     `json:"fileSize"`
	DateTaken        time.Time `json:"dateTaken"`
	UploadedAt       time.Time `json:"uploadedAt"`
}

// NewPhoto creates a new Photo with validation and sanitization
func NewPhoto(originalFilename, storedPath, fileHash string, fileSize int64, dateTaken time.Time) (*Photo, error) {
	if strings.TrimSpace(originalFilename) == "" {
		return nil, ErrEmptyFilename
	}
	if strings.TrimSpace(storedPath) == "" {
		return nil, ErrEmptyStoredPath
	}
	if strings.TrimSpace(fileHash) == "" {
		return nil, ErrEmptyHash
	}
	if fileSize <= 0 {
		return nil, ErrInvalidFileSize
	}

	return &Photo{
		ID:               uuid.New().String(),
		OriginalFilename: sanitizeFilename(originalFilename),
		StoredPath:       storedPath,
		FileHash:         strings.ToLower(fileHash),
		FileSize:         fileSize,
		DateTaken:        dateTaken,
		UploadedAt:       time.Now().UTC(),
	}, nil
}

// sanitizeFilename removes path components and invalid characters
func sanitizeFilename(filename string) string {
	// Get just the filename, no path
	name := filepath.Base(filename)

	// Remove potentially dangerous characters
	replacer := strings.NewReplacer(
		"..", "",
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	)

	return replacer.Replace(name)
}

// Errors
type PhotoError struct {
	Message string
}

func (e PhotoError) Error() string {
	return e.Message
}

var (
	ErrEmptyFilename    = PhotoError{"original filename cannot be empty"}
	ErrEmptyStoredPath  = PhotoError{"stored path cannot be empty"}
	ErrEmptyHash        = PhotoError{"file hash cannot be empty"}
	ErrInvalidFileSize  = PhotoError{"file size must be positive"}
	ErrPhotoNotFound    = PhotoError{"photo not found"}
	ErrDuplicatePhoto   = PhotoError{"photo already exists"}
	ErrInvalidExtension = PhotoError{"file extension not allowed"}
	ErrFileTooLarge     = PhotoError{"file size exceeds maximum allowed"}
	ErrPathTraversal    = PhotoError{"invalid path - path traversal detected"}
)

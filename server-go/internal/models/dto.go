package models

import "time"

// UploadResult is returned after uploading a photo
type UploadResult struct {
	ID          string    `json:"id"`
	StoredPath  string    `json:"storedPath"`
	UploadedAt  time.Time `json:"uploadedAt"`
	IsDuplicate bool      `json:"isDuplicate"`
}

// NewUploadResult creates a result for a newly uploaded photo
func NewUploadResult(id, storedPath string, uploadedAt time.Time) UploadResult {
	return UploadResult{
		ID:          id,
		StoredPath:  storedPath,
		UploadedAt:  uploadedAt,
		IsDuplicate: false,
	}
}

// DuplicateUploadResult creates a result for a duplicate photo
func DuplicateUploadResult(id, storedPath string, uploadedAt time.Time) UploadResult {
	return UploadResult{
		ID:          id,
		StoredPath:  storedPath,
		UploadedAt:  uploadedAt,
		IsDuplicate: true,
	}
}

// CheckHashesRequest is the request body for checking hashes
type CheckHashesRequest struct {
	Hashes []string `json:"hashes"`
}

// CheckHashesResult is returned when checking which hashes exist
type CheckHashesResult struct {
	Existing []string `json:"existing"`
	Missing  []string `json:"missing"`
}

// PhotoResponse is a single photo in API responses
type PhotoResponse struct {
	ID               string    `json:"id"`
	OriginalFilename string    `json:"originalFilename"`
	StoredPath       string    `json:"storedPath"`
	FileSize         int64     `json:"fileSize"`
	DateTaken        time.Time `json:"dateTaken"`
	UploadedAt       time.Time `json:"uploadedAt"`
}

// PhotoListResponse is returned when listing photos
type PhotoListResponse struct {
	Photos     []PhotoResponse `json:"photos"`
	TotalCount int             `json:"totalCount"`
	Skip       int             `json:"skip"`
	Take       int             `json:"take"`
}

// HealthResponse is returned by health check
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

// ErrorResponse is returned on errors
type ErrorResponse struct {
	Error string `json:"error"`
}

// PhotoToResponse converts a Photo to PhotoResponse
func PhotoToResponse(p *Photo) PhotoResponse {
	return PhotoResponse{
		ID:               p.ID,
		OriginalFilename: p.OriginalFilename,
		StoredPath:       p.StoredPath,
		FileSize:         p.FileSize,
		DateTaken:        p.DateTaken,
		UploadedAt:       p.UploadedAt,
	}
}

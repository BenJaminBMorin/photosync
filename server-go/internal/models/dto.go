package models

import "time"

// UploadResult is returned after uploading a photo
// @Description Result of a photo upload operation
type UploadResult struct {
	ID          string    `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	StoredPath  string    `json:"storedPath" example:"2024/01/IMG_1234.jpg"`
	UploadedAt  time.Time `json:"uploadedAt" example:"2024-01-15T10:30:00Z"`
	IsDuplicate bool      `json:"isDuplicate" example:"false"`
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
// @Description Request to check which photo hashes already exist on the server
type CheckHashesRequest struct {
	Hashes []string `json:"hashes" example:"abc123def456,789ghi012jkl"`
}

// CheckHashesResult is returned when checking which hashes exist
// @Description Result of hash check showing which photos exist and which are missing
type CheckHashesResult struct {
	Existing []string `json:"existing" example:"abc123def456"`
	Missing  []string `json:"missing" example:"789ghi012jkl"`
}

// PhotoResponse is a single photo in API responses
// @Description Photo metadata returned by the API
type PhotoResponse struct {
	ID               string    `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	OriginalFilename string    `json:"originalFilename" example:"IMG_1234.jpg"`
	StoredPath       string    `json:"storedPath" example:"2024/01/IMG_1234.jpg"`
	FileSize         int64     `json:"fileSize" example:"2048576"`
	DateTaken        time.Time `json:"dateTaken" example:"2024-01-15T10:30:00Z"`
	UploadedAt       time.Time `json:"uploadedAt" example:"2024-01-15T12:00:00Z"`
}

// PhotoListResponse is returned when listing photos
// @Description Paginated list of photos
type PhotoListResponse struct {
	Photos     []PhotoResponse `json:"photos"`
	TotalCount int             `json:"totalCount" example:"150"`
	Skip       int             `json:"skip" example:"0"`
	Take       int             `json:"take" example:"50"`
}

// HealthResponse is returned by health check
// @Description Server health status
type HealthResponse struct {
	Status    string    `json:"status" example:"healthy"`
	Timestamp time.Time `json:"timestamp" example:"2024-01-15T12:00:00Z"`
}

// ErrorResponse is returned on errors
// @Description Error response
type ErrorResponse struct {
	Error string `json:"error" example:"Photo not found"`
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

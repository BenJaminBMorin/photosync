package models

import "time"

// SyncStatusResponse for GET /api/sync/status
type SyncStatusResponse struct {
	TotalPhotos       int        `json:"totalPhotos"`
	DevicePhotos      int        `json:"devicePhotos"`
	OtherDevicePhotos int        `json:"otherDevicePhotos"`
	LegacyPhotos      int        `json:"legacyPhotos"`
	LastSyncAt        *time.Time `json:"lastSyncAt,omitempty"`
	ServerVersion     int        `json:"serverVersion"`
	NeedsLegacyClaim  bool       `json:"needsLegacyClaim"`
}

// SyncPhotosRequest for POST /api/sync/photos
type SyncPhotosRequest struct {
	DeviceID             string     `json:"deviceId"`
	Cursor               string     `json:"cursor,omitempty"`
	Limit                int        `json:"limit"`
	IncludeThumbnailURLs bool       `json:"includeThumbnailUrls"`
	SinceTimestamp       *time.Time `json:"sinceTimestamp,omitempty"`
}

// SyncPhotoItem represents a photo in sync response
type SyncPhotoItem struct {
	ID               string            `json:"id"`
	FileHash         string            `json:"fileHash"`
	OriginalFilename string            `json:"originalFilename"`
	FileSize         int64             `json:"fileSize"`
	DateTaken        time.Time         `json:"dateTaken"`
	UploadedAt       time.Time         `json:"uploadedAt"`
	OriginDevice     *OriginDeviceInfo `json:"originDevice,omitempty"`
	ThumbnailURL     string            `json:"thumbnailUrl,omitempty"`
	Width            *int              `json:"width,omitempty"`
	Height           *int              `json:"height,omitempty"`
}

// OriginDeviceInfo embedded in SyncPhotoItem
type OriginDeviceInfo struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Platform        string `json:"platform"`
	IsCurrentDevice bool   `json:"isCurrentDevice"`
}

// SyncPhotosResponse for POST /api/sync/photos
type SyncPhotosResponse struct {
	Photos     []SyncPhotoItem `json:"photos"`
	Pagination PaginationInfo  `json:"pagination"`
	Sync       SyncInfo        `json:"sync"`
}

// PaginationInfo for cursor-based pagination
type PaginationInfo struct {
	Cursor  string `json:"cursor,omitempty"`
	HasMore bool   `json:"hasMore"`
}

// SyncInfo provides sync metadata
type SyncInfo struct {
	TotalCount    int `json:"totalCount"`
	ReturnedCount int `json:"returnedCount"`
	ServerVersion int `json:"serverVersion"`
}

// ClaimLegacyRequest for POST /api/sync/claim-legacy
type ClaimLegacyRequest struct {
	DeviceID string   `json:"deviceId"`
	PhotoIDs []string `json:"photoIds,omitempty"`
	ClaimAll bool     `json:"claimAll"`
}

// ClaimLegacyResponse for POST /api/sync/claim-legacy
type ClaimLegacyResponse struct {
	Claimed        int `json:"claimed"`
	AlreadyClaimed int `json:"alreadyClaimed"`
	Failed         int `json:"failed"`
}

// LegacyPhotosResponse for GET /api/sync/legacy-photos
type LegacyPhotosResponse struct {
	Photos     []SyncPhotoItem `json:"photos"`
	TotalCount int             `json:"totalCount"`
	Message    string          `json:"message"`
}

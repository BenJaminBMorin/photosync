package models

import "time"

// UpdateUserRequest is the request body for admin user update
type UpdateUserRequest struct {
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
	IsAdmin     bool   `json:"isAdmin"`
	IsActive    bool   `json:"isActive"`
}

// AdminUserResponse contains extended user info for admin views
type AdminUserResponse struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	DisplayName  string    `json:"displayName"`
	IsAdmin      bool      `json:"isAdmin"`
	IsActive     bool      `json:"isActive"`
	CreatedAt    time.Time `json:"createdAt"`
	DeviceCount  int       `json:"deviceCount"`
	SessionCount int       `json:"sessionCount"`
	PhotoCount   int       `json:"photoCount"`
}

// UserListResponse contains paginated user list
type UserListResponse struct {
	Users      []AdminUserResponse `json:"users"`
	TotalCount int                 `json:"totalCount"`
}

// CreateUserResponse contains the new user and their API key (shown only once)
type CreateUserResponse struct {
	User   UserResponse `json:"user"`
	APIKey string       `json:"apiKey"`
}

// ResetAPIKeyResponse contains the new API key (shown only once)
type ResetAPIKeyResponse struct {
	APIKey string `json:"apiKey"`
}

// AdminDeviceResponse contains device info for admin views
type AdminDeviceResponse struct {
	ID           string    `json:"id"`
	UserID       string    `json:"userId"`
	DeviceName   string    `json:"deviceName"`
	Platform     string    `json:"platform"`
	RegisteredAt time.Time `json:"registeredAt"`
	LastSeenAt   time.Time `json:"lastSeenAt"`
	IsActive     bool      `json:"isActive"`
}

// AdminSessionResponse contains session info for admin views
type AdminSessionResponse struct {
	ID             string    `json:"id"`
	UserID         string    `json:"userId"`
	CreatedAt      time.Time `json:"createdAt"`
	ExpiresAt      time.Time `json:"expiresAt"`
	LastActivityAt time.Time `json:"lastActivityAt"`
	IPAddress      string    `json:"ipAddress"`
	UserAgent      string    `json:"userAgent"`
	IsActive       bool      `json:"isActive"`
}

// SystemStatusResponse contains system health and statistics
type SystemStatusResponse struct {
	Version            string         `json:"version"`
	BuildVersion       string         `json:"buildVersion"`
	BuildDate          string         `json:"buildDate"`
	ServerStartTime    string         `json:"serverStartTime"`
	ContainerBuildDate string         `json:"containerBuildDate,omitempty"`
	Uptime             string         `json:"uptime"`
	Database           DatabaseStatus `json:"database"`
	Storage            StorageStatus  `json:"storage"`
	Firebase           FirebaseStatus `json:"firebase"`
	Stats              SystemStats    `json:"stats"`
}

// DatabaseStatus contains database connection info
type DatabaseStatus struct {
	Type      string `json:"type"` // "sqlite" or "postgres"
	Connected bool   `json:"connected"`
}

// StorageStatus contains photo storage info
type StorageStatus struct {
	BasePath    string `json:"basePath"`
	TotalPhotos int64  `json:"totalPhotos"`
	TotalSizeMB int64  `json:"totalSizeMB"`
}

// FirebaseStatus contains Firebase configuration status
type FirebaseStatus struct {
	Configured bool   `json:"configured"`
	ProjectID  string `json:"projectId,omitempty"`
}

// SystemStats contains system-wide statistics
type SystemStats struct {
	TotalUsers     int `json:"totalUsers"`
	TotalPhotos    int `json:"totalPhotos"`
	TotalDevices   int `json:"totalDevices"`
	ActiveSessions int `json:"activeSessions"`
}

// SystemConfigResponse contains current system configuration
type SystemConfigResponse struct {
	SetupComplete      bool     `json:"setupComplete"`
	FirebaseConfigured bool     `json:"firebaseConfigured"`
	PhotoStoragePath   string   `json:"photoStoragePath"`
	MaxFileSizeMB      int64    `json:"maxFileSizeMB"`
	AllowedExtensions  []string `json:"allowedExtensions"`
}

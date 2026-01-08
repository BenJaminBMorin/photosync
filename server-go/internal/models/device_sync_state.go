package models

import "time"

// DeviceSyncState tracks sync state per device
type DeviceSyncState struct {
	DeviceID        string     `json:"deviceId"`
	LastSyncAt      *time.Time `json:"lastSyncAt,omitempty"`
	LastSyncPhotoID string     `json:"lastSyncPhotoId,omitempty"`
	SyncVersion     int        `json:"syncVersion"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

// NewDeviceSyncState creates a new DeviceSyncState
func NewDeviceSyncState(deviceID string) *DeviceSyncState {
	now := time.Now().UTC()
	return &DeviceSyncState{
		DeviceID:    deviceID,
		SyncVersion: 0,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

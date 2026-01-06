package models

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// Device represents a registered mobile device for push notifications
type Device struct {
	ID           string    `json:"id"`
	UserID       string    `json:"userId"`
	DeviceName   string    `json:"deviceName"`
	Platform     string    `json:"platform"` // "ios" or "android"
	FCMToken     string    `json:"-"`        // Never expose FCM token
	RegisteredAt time.Time `json:"registeredAt"`
	LastSeenAt   time.Time `json:"lastSeenAt"`
	IsActive     bool      `json:"isActive"`
}

// DeviceResponse is the safe response format
type DeviceResponse struct {
	ID           string    `json:"id"`
	DeviceName   string    `json:"deviceName"`
	Platform     string    `json:"platform"`
	RegisteredAt time.Time `json:"registeredAt"`
	LastSeenAt   time.Time `json:"lastSeenAt"`
	IsActive     bool      `json:"isActive"`
}

// RegisterDeviceRequest is the request body for registering a device
type RegisterDeviceRequest struct {
	DeviceName string `json:"deviceName"`
	Platform   string `json:"platform"`
	FCMToken   string `json:"fcmToken"`
}

// UpdateTokenRequest is for updating a device's FCM token
type UpdateTokenRequest struct {
	FCMToken string `json:"fcmToken"`
}

// NewDevice creates a new device registration
func NewDevice(userID, deviceName, platform, fcmToken string) (*Device, error) {
	deviceName = strings.TrimSpace(deviceName)
	platform = strings.TrimSpace(strings.ToLower(platform))
	fcmToken = strings.TrimSpace(fcmToken)

	if deviceName == "" {
		return nil, ErrEmptyDeviceName
	}
	if platform != "ios" && platform != "android" {
		return nil, ErrInvalidPlatform
	}
	if fcmToken == "" {
		return nil, ErrEmptyFCMToken
	}

	now := time.Now().UTC()
	return &Device{
		ID:           uuid.New().String(),
		UserID:       userID,
		DeviceName:   deviceName,
		Platform:     platform,
		FCMToken:     fcmToken,
		RegisteredAt: now,
		LastSeenAt:   now,
		IsActive:     true,
	}, nil
}

// ToResponse converts Device to DeviceResponse (safe for API)
func (d *Device) ToResponse() DeviceResponse {
	return DeviceResponse{
		ID:           d.ID,
		DeviceName:   d.DeviceName,
		Platform:     d.Platform,
		RegisteredAt: d.RegisteredAt,
		LastSeenAt:   d.LastSeenAt,
		IsActive:     d.IsActive,
	}
}

// Device errors
var (
	ErrEmptyDeviceName = DeviceError{"device name cannot be empty"}
	ErrInvalidPlatform = DeviceError{"platform must be 'ios' or 'android'"}
	ErrEmptyFCMToken   = DeviceError{"FCM token cannot be empty"}
	ErrDeviceNotFound  = DeviceError{"device not found"}
)

type DeviceError struct {
	Message string
}

func (e DeviceError) Error() string {
	return e.Message
}

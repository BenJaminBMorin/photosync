package repository

import (
	"context"

	"github.com/photosync/server/internal/models"
)

// PhotoRepo defines the interface for photo persistence operations
type PhotoRepo interface {
	GetByID(ctx context.Context, id string) (*models.Photo, error)
	GetByHash(ctx context.Context, hash string) (*models.Photo, error)
	GetByHashAndUser(ctx context.Context, hash, userID string) (*models.Photo, error)
	GetExistingHashes(ctx context.Context, hashes []string) ([]string, error)
	GetExistingHashesForUser(ctx context.Context, hashes []string, userID string) ([]string, error)
	GetAll(ctx context.Context, skip, take int) ([]*models.Photo, error)
	GetAllForUser(ctx context.Context, userID string, skip, take int) ([]*models.Photo, error)
	GetCount(ctx context.Context) (int, error)
	GetCountForUser(ctx context.Context, userID string) (int, error)
	Add(ctx context.Context, photo *models.Photo) error
	AddWithUser(ctx context.Context, photo *models.Photo, userID string) error
	Delete(ctx context.Context, id string) (bool, error)
	DeleteAll(ctx context.Context) (int, error)                               // Delete all photos
	VerifyExistence(ctx context.Context, ids []string) (map[string]bool, error) // Check which IDs exist
}

// UserRepo defines the interface for user persistence operations
type UserRepo interface {
	GetByID(ctx context.Context, id string) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	GetByAPIKeyHash(ctx context.Context, apiKeyHash string) (*models.User, error)
	GetAll(ctx context.Context) ([]*models.User, error)
	GetCount(ctx context.Context) (int, error)
	Add(ctx context.Context, user *models.User) error
	Update(ctx context.Context, user *models.User) error
	Delete(ctx context.Context, id string) (bool, error)
}

// DeviceRepo defines the interface for device persistence operations
type DeviceRepo interface {
	GetByID(ctx context.Context, id string) (*models.Device, error)
	GetByFCMToken(ctx context.Context, fcmToken string) (*models.Device, error)
	GetAllForUser(ctx context.Context, userID string) ([]*models.Device, error)
	GetActiveForUser(ctx context.Context, userID string) ([]*models.Device, error)
	Add(ctx context.Context, device *models.Device) error
	UpdateToken(ctx context.Context, id, fcmToken string) error
	UpdateLastSeen(ctx context.Context, id string) error
	Deactivate(ctx context.Context, id string) error
	Delete(ctx context.Context, id string) (bool, error)
}

// AuthRequestRepo defines the interface for auth request persistence
type AuthRequestRepo interface {
	GetByID(ctx context.Context, id string) (*models.AuthRequest, error)
	GetPendingForUser(ctx context.Context, userID string) ([]*models.AuthRequest, error)
	Add(ctx context.Context, req *models.AuthRequest) error
	Update(ctx context.Context, req *models.AuthRequest) error
	ExpireOld(ctx context.Context) (int, error) // Expire all requests past their expiry time
}

// WebSessionRepo defines the interface for web session persistence
type WebSessionRepo interface {
	GetByID(ctx context.Context, id string) (*models.WebSession, error)
	GetActiveForUser(ctx context.Context, userID string) ([]*models.WebSession, error)
	Add(ctx context.Context, session *models.WebSession) error
	Touch(ctx context.Context, id string) error
	Invalidate(ctx context.Context, id string) error
	InvalidateAllForUser(ctx context.Context, userID string) error
	CleanupExpired(ctx context.Context) (int, error)
}

// SetupConfigRepo defines the interface for setup configuration
type SetupConfigRepo interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string) error
	GetAll(ctx context.Context) (map[string]string, error)
	IsSetupComplete(ctx context.Context) (bool, error)
}

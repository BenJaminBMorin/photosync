package repository

import (
	"context"
	"time"

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
	DeleteAll(ctx context.Context) (int, error)                                  // Delete all photos
	VerifyExistence(ctx context.Context, ids []string) (map[string]bool, error)  // Check which IDs exist
	GetPhotosWithoutThumbnails(ctx context.Context, limit int) ([]*models.Photo, error) // Get photos missing thumbnails
	UpdateThumbnails(ctx context.Context, photoID, smallPath, mediumPath, largePath string) error // Update thumbnail paths
	GetOrphanedPhotos(ctx context.Context, limit int) ([]*models.Photo, error) // Get photos without an owner

	// Sync-related methods
	GetAllForUserWithCursor(ctx context.Context, userID string, cursor string, limit int, sinceTimestamp *time.Time) ([]*models.Photo, string, error)
	GetCountByOriginDevice(ctx context.Context, userID, deviceID string) (int, error)
	GetLegacyPhotosForUser(ctx context.Context, userID string, limit int) ([]*models.Photo, error)
	GetLegacyPhotoCount(ctx context.Context, userID string) (int, error)
	ClaimLegacyPhotos(ctx context.Context, photoIDs []string, deviceID string) (int, error)
	ClaimAllLegacyPhotos(ctx context.Context, userID, deviceID string) (int, error)
	SetOriginDevice(ctx context.Context, photoID, deviceID string) error
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
	UpdateAPIKeyHash(ctx context.Context, id, apiKeyHash string) error
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

// BootstrapKeyRepo defines the interface for bootstrap key persistence
type BootstrapKeyRepo interface {
	Add(ctx context.Context, key *models.BootstrapKey) error
	GetByKeyHash(ctx context.Context, hash string) (*models.BootstrapKey, error)
	GetActiveKey(ctx context.Context) (*models.BootstrapKey, error)
	MarkUsed(ctx context.Context, id, usedBy string) error
	ExpireOld(ctx context.Context) (int, error)
	HasActiveKey(ctx context.Context) (bool, error)
}

// RecoveryTokenRepo defines the interface for recovery token persistence
type RecoveryTokenRepo interface {
	Add(ctx context.Context, token *models.RecoveryToken) error
	GetByTokenHash(ctx context.Context, hash string) (*models.RecoveryToken, error)
	MarkUsed(ctx context.Context, id, ipAddress string) error
	GetRecentCountForEmail(ctx context.Context, email string, since time.Time) (int, error)
	RecordRateLimit(ctx context.Context, email string) error
	CheckRateLimit(ctx context.Context, email string) (bool, error)
	ExpireOld(ctx context.Context) (int, error)
}

// ConfigOverrideRepo defines the interface for config override persistence
type ConfigOverrideRepo interface {
	Get(ctx context.Context, key string) (*models.ConfigItem, error)
	GetAll(ctx context.Context) ([]*models.ConfigItem, error)
	GetByCategory(ctx context.Context, category models.ConfigCategory) ([]*models.ConfigItem, error)
	Set(ctx context.Context, key, value, valueType string, category models.ConfigCategory, requiresRestart, isSensitive bool, updatedBy string) error
	Delete(ctx context.Context, key string) error
	HasRestartRequired(ctx context.Context) (bool, error)
}

// SMTPConfigRepo defines the interface for SMTP configuration persistence
type SMTPConfigRepo interface {
	Get(ctx context.Context) (*models.SMTPConfig, error)
	Set(ctx context.Context, config *models.SMTPConfig, updatedBy string) error
	IsConfigured(ctx context.Context) (bool, error)
}

// CollectionRepo defines the interface for collection persistence
type CollectionRepo interface {
	GetByID(ctx context.Context, id string) (*models.Collection, error)
	GetBySlug(ctx context.Context, slug string) (*models.Collection, error)
	GetBySecretToken(ctx context.Context, token string) (*models.Collection, error)
	GetAllForUser(ctx context.Context, userID string) ([]*models.Collection, error)
	GetSharedWithUser(ctx context.Context, userID string) ([]*models.Collection, error)
	Add(ctx context.Context, collection *models.Collection) error
	Update(ctx context.Context, collection *models.Collection) error
	Delete(ctx context.Context, id string) error
	SlugExists(ctx context.Context, slug string, excludeID string) (bool, error)
}

// CollectionPhotoRepo defines the interface for collection-photo associations
type CollectionPhotoRepo interface {
	GetByCollectionID(ctx context.Context, collectionID string) ([]*models.CollectionPhoto, error)
	GetPhotosForCollection(ctx context.Context, collectionID string) ([]*models.Photo, error)
	GetPhotoCountForCollection(ctx context.Context, collectionID string) (int, error)
	Add(ctx context.Context, cp *models.CollectionPhoto) error
	AddMultiple(ctx context.Context, collectionID string, photoIDs []string) error
	Remove(ctx context.Context, collectionID, photoID string) error
	RemoveMultiple(ctx context.Context, collectionID string, photoIDs []string) error
	Reorder(ctx context.Context, collectionID string, photoIDs []string) error
	IsPhotoInCollection(ctx context.Context, collectionID, photoID string) (bool, error)
	GetMaxPosition(ctx context.Context, collectionID string) (int, error)
}

// CollectionShareRepo defines the interface for collection sharing
type CollectionShareRepo interface {
	GetByCollectionID(ctx context.Context, collectionID string) ([]*models.CollectionShare, error)
	GetSharesWithUsers(ctx context.Context, collectionID string) ([]*models.CollectionShareWithUser, error)
	IsSharedWithUser(ctx context.Context, collectionID, userID string) (bool, error)
	Add(ctx context.Context, share *models.CollectionShare) error
	Remove(ctx context.Context, collectionID, userID string) error
	RemoveAll(ctx context.Context, collectionID string) error
}

// DeviceSyncStateRepo defines the interface for device sync state tracking
type DeviceSyncStateRepo interface {
	Get(ctx context.Context, deviceID string) (*models.DeviceSyncState, error)
	Upsert(ctx context.Context, state *models.DeviceSyncState) error
	GetSyncVersion(ctx context.Context, userID string) (int, error)
	IncrementSyncVersion(ctx context.Context, userID string) error
	UpdateLastSync(ctx context.Context, deviceID string, lastPhotoID string) error
}

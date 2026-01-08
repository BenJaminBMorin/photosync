package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/photosync/server/internal/models"
)

// DeviceSyncStateRepository handles device sync state persistence
type DeviceSyncStateRepository struct {
	db *sql.DB
}

// NewDeviceSyncStateRepository creates a new DeviceSyncStateRepository
func NewDeviceSyncStateRepository(db *sql.DB) *DeviceSyncStateRepository {
	return &DeviceSyncStateRepository{db: db}
}

// Get retrieves sync state for a device
func (r *DeviceSyncStateRepository) Get(ctx context.Context, deviceID string) (*models.DeviceSyncState, error) {
	query := `SELECT device_id, last_sync_at, last_sync_photo_id, sync_version, created_at, updated_at
		FROM device_sync_state WHERE device_id = $1`

	var state models.DeviceSyncState
	err := r.db.QueryRowContext(ctx, query, deviceID).Scan(
		&state.DeviceID,
		&state.LastSyncAt,
		&state.LastSyncPhotoID,
		&state.SyncVersion,
		&state.CreatedAt,
		&state.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &state, nil
}

// Upsert creates or updates sync state for a device
func (r *DeviceSyncStateRepository) Upsert(ctx context.Context, state *models.DeviceSyncState) error {
	query := `INSERT INTO device_sync_state (device_id, last_sync_at, last_sync_photo_id, sync_version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (device_id) DO UPDATE SET
			last_sync_at = EXCLUDED.last_sync_at,
			last_sync_photo_id = EXCLUDED.last_sync_photo_id,
			sync_version = EXCLUDED.sync_version,
			updated_at = EXCLUDED.updated_at`

	_, err := r.db.ExecContext(ctx, query,
		state.DeviceID,
		state.LastSyncAt,
		state.LastSyncPhotoID,
		state.SyncVersion,
		state.CreatedAt,
		state.UpdatedAt,
	)
	return err
}

// GetSyncVersion returns the current sync version for a user
// This is used to detect changes since last sync
func (r *DeviceSyncStateRepository) GetSyncVersion(ctx context.Context, userID string) (int, error) {
	// Sync version is tracked per user - we use a simple counter based on photo count
	// More sophisticated implementations could use a sequence or version table
	var count int
	err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM photos WHERE user_id = $1",
		userID,
	).Scan(&count)
	return count, err
}

// IncrementSyncVersion is a no-op for this simple implementation
// Version is derived from photo count
func (r *DeviceSyncStateRepository) IncrementSyncVersion(ctx context.Context, userID string) error {
	// No-op - version is derived from photo count
	return nil
}

// UpdateLastSync updates the last sync timestamp and photo ID for a device
func (r *DeviceSyncStateRepository) UpdateLastSync(ctx context.Context, deviceID string, lastPhotoID string) error {
	now := time.Now().UTC()

	// First, try to get existing state
	existing, err := r.Get(ctx, deviceID)
	if err != nil {
		return err
	}

	if existing == nil {
		// Create new state
		state := &models.DeviceSyncState{
			DeviceID:        deviceID,
			LastSyncAt:      &now,
			LastSyncPhotoID: lastPhotoID,
			SyncVersion:     0,
			CreatedAt:       now,
			UpdatedAt:       now,
		}
		return r.Upsert(ctx, state)
	}

	// Update existing
	query := `UPDATE device_sync_state
		SET last_sync_at = $1, last_sync_photo_id = $2, updated_at = $3
		WHERE device_id = $4`

	_, err = r.db.ExecContext(ctx, query, now, lastPhotoID, now, deviceID)
	return err
}

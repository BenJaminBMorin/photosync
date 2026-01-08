package repository

import (
	"context"
	"strings"
	"time"

	"github.com/photosync/server/internal/models"
)

// DeleteAll deletes all photos from the database
// Returns the number of photos deleted
func (r *PhotoRepository) DeleteAll(ctx context.Context) (int, error) {
	result, err := r.db.ExecContext(ctx, "DELETE FROM photos")
	if err != nil {
		return 0, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return int(affected), nil
}

// VerifyExistence checks which photo IDs exist in the database
// Returns a map where keys are photo IDs and values indicate existence (true/false)
func (r *PhotoRepository) VerifyExistence(ctx context.Context, ids []string) (map[string]bool, error) {
	if len(ids) == 0 {
		return map[string]bool{}, nil
	}

	// Initialize result map with all IDs as false
	result := make(map[string]bool, len(ids))
	for _, id := range ids {
		result[id] = false
	}

	// Build query with placeholders
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}

	query := `SELECT id FROM photos WHERE id IN (` + strings.Join(placeholders, ",") + `)`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Mark found IDs as true
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		result[id] = true
	}

	return result, rows.Err()
}

// GetPhotosWithoutThumbnails returns photos that don't have thumbnails generated
// Note: SQLite version may not have thumbnail columns, returns empty slice
func (r *PhotoRepository) GetPhotosWithoutThumbnails(ctx context.Context, limit int) ([]*models.Photo, error) {
	// SQLite schema may not have thumbnail columns, so return empty
	return []*models.Photo{}, nil
}

// UpdateThumbnails updates the thumbnail paths for a photo
// Note: SQLite version may not have thumbnail columns, this is a no-op
func (r *PhotoRepository) UpdateThumbnails(ctx context.Context, photoID, smallPath, mediumPath, largePath string) error {
	// SQLite schema may not have thumbnail columns, so no-op
	return nil
}

// GetOrphanedPhotos returns photos that don't have an owner
// Note: SQLite version returns empty slice
func (r *PhotoRepository) GetOrphanedPhotos(ctx context.Context, limit int) ([]*models.Photo, error) {
	return []*models.Photo{}, nil
}

// Sync-related methods for SQLite (stubs for development/testing)

// GetAllForUserWithCursor returns photos with cursor-based pagination
func (r *PhotoRepository) GetAllForUserWithCursor(ctx context.Context, userID string, cursor string, limit int, sinceTimestamp *time.Time) ([]*models.Photo, string, error) {
	// SQLite stub - returns all photos without cursor support
	photos, err := r.GetAllForUser(ctx, userID, 0, limit)
	if err != nil {
		return nil, "", err
	}
	return photos, "", nil
}

// GetCountByOriginDevice returns count of photos from a specific device
func (r *PhotoRepository) GetCountByOriginDevice(ctx context.Context, userID, deviceID string) (int, error) {
	// SQLite stub - returns 0
	return 0, nil
}

// GetLegacyPhotosForUser returns photos without an origin device
func (r *PhotoRepository) GetLegacyPhotosForUser(ctx context.Context, userID string, limit int) ([]*models.Photo, error) {
	// SQLite stub - returns empty
	return []*models.Photo{}, nil
}

// GetLegacyPhotoCount returns count of photos without an origin device
func (r *PhotoRepository) GetLegacyPhotoCount(ctx context.Context, userID string) (int, error) {
	// SQLite stub - returns 0
	return 0, nil
}

// ClaimLegacyPhotos sets the origin device for specific photos
func (r *PhotoRepository) ClaimLegacyPhotos(ctx context.Context, photoIDs []string, deviceID string) (int, error) {
	// SQLite stub - no-op
	return 0, nil
}

// ClaimAllLegacyPhotos sets the origin device for all legacy photos for a user
func (r *PhotoRepository) ClaimAllLegacyPhotos(ctx context.Context, userID, deviceID string) (int, error) {
	// SQLite stub - no-op
	return 0, nil
}

// SetOriginDevice sets the origin device for a photo
func (r *PhotoRepository) SetOriginDevice(ctx context.Context, photoID, deviceID string) error {
	// SQLite stub - no-op
	return nil
}

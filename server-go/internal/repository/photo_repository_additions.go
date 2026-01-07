package repository

import (
	"context"
	"strings"

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

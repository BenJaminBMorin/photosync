package repository

import (
	"context"
	"database/sql"
	"strings"

	"github.com/google/uuid"
	"github.com/photosync/server/internal/models"
)

// CollectionPhotoRepository implements CollectionPhotoRepo for PostgreSQL/SQLite
type CollectionPhotoRepository struct {
	db *sql.DB
}

// NewCollectionPhotoRepository creates a new CollectionPhotoRepository
func NewCollectionPhotoRepository(db *sql.DB) *CollectionPhotoRepository {
	return &CollectionPhotoRepository{db: db}
}

func (r *CollectionPhotoRepository) GetByCollectionID(ctx context.Context, collectionID string) ([]*models.CollectionPhoto, error) {
	query := `SELECT id, collection_id, photo_id, position, added_at
			  FROM collection_photos WHERE collection_id = $1 ORDER BY position ASC`

	rows, err := r.db.QueryContext(ctx, query, collectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var photos []*models.CollectionPhoto
	for rows.Next() {
		var cp models.CollectionPhoto
		if err := rows.Scan(&cp.ID, &cp.CollectionID, &cp.PhotoID, &cp.Position, &cp.AddedAt); err != nil {
			return nil, err
		}
		photos = append(photos, &cp)
	}
	return photos, rows.Err()
}

func (r *CollectionPhotoRepository) GetPhotosForCollection(ctx context.Context, collectionID string) ([]*models.Photo, error) {
	query := `SELECT p.id, p.user_id, p.original_filename, p.stored_path, p.file_hash, p.file_size,
			  p.date_taken, p.uploaded_at, p.thumb_small, p.thumb_medium, p.thumb_large,
			  p.camera_make, p.camera_model, p.lens_model, p.focal_length, p.aperture,
			  p.shutter_speed, p.iso, p.orientation, p.latitude, p.longitude, p.altitude,
			  p.width, p.height
			  FROM photos p
			  INNER JOIN collection_photos cp ON cp.photo_id = p.id
			  WHERE cp.collection_id = $1 ORDER BY cp.position ASC`

	rows, err := r.db.QueryContext(ctx, query, collectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var photos []*models.Photo
	for rows.Next() {
		var p models.Photo
		if err := rows.Scan(
			&p.ID, &p.UserID, &p.OriginalFilename, &p.StoredPath, &p.FileHash, &p.FileSize,
			&p.DateTaken, &p.UploadedAt, &p.ThumbSmall, &p.ThumbMedium, &p.ThumbLarge,
			&p.CameraMake, &p.CameraModel, &p.LensModel, &p.FocalLength, &p.Aperture,
			&p.ShutterSpeed, &p.ISO, &p.Orientation, &p.Latitude, &p.Longitude, &p.Altitude,
			&p.Width, &p.Height,
		); err != nil {
			return nil, err
		}
		photos = append(photos, &p)
	}
	return photos, rows.Err()
}

func (r *CollectionPhotoRepository) GetPhotoCountForCollection(ctx context.Context, collectionID string) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM collection_photos WHERE collection_id = $1`, collectionID).Scan(&count)
	return count, err
}

func (r *CollectionPhotoRepository) Add(ctx context.Context, cp *models.CollectionPhoto) error {
	query := `INSERT INTO collection_photos (id, collection_id, photo_id, position, added_at)
			  VALUES ($1, $2, $3, $4, $5)
			  ON CONFLICT (collection_id, photo_id) DO NOTHING`

	_, err := r.db.ExecContext(ctx, query, cp.ID, cp.CollectionID, cp.PhotoID, cp.Position, cp.AddedAt)
	return err
}

func (r *CollectionPhotoRepository) AddMultiple(ctx context.Context, collectionID string, photoIDs []string) error {
	if len(photoIDs) == 0 {
		return nil
	}

	// Get current max position
	maxPos, err := r.GetMaxPosition(ctx, collectionID)
	if err != nil {
		return err
	}

	// Build batch insert
	valueStrings := make([]string, 0, len(photoIDs))
	valueArgs := make([]interface{}, 0, len(photoIDs)*5)

	for i, photoID := range photoIDs {
		idx := i * 5
		valueStrings = append(valueStrings, "($"+string(rune('1'+idx))+", $"+string(rune('2'+idx))+", $"+string(rune('3'+idx))+", $"+string(rune('4'+idx))+", NOW())")
		valueArgs = append(valueArgs, uuid.New().String(), collectionID, photoID, maxPos+i+1)
	}

	// Use a simpler approach with individual inserts to avoid placeholder issues
	for i, photoID := range photoIDs {
		query := `INSERT INTO collection_photos (id, collection_id, photo_id, position, added_at)
				  VALUES ($1, $2, $3, $4, NOW())
				  ON CONFLICT (collection_id, photo_id) DO NOTHING`
		_, err := r.db.ExecContext(ctx, query, uuid.New().String(), collectionID, photoID, maxPos+i+1)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *CollectionPhotoRepository) Remove(ctx context.Context, collectionID, photoID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM collection_photos WHERE collection_id = $1 AND photo_id = $2`, collectionID, photoID)
	return err
}

func (r *CollectionPhotoRepository) RemoveMultiple(ctx context.Context, collectionID string, photoIDs []string) error {
	if len(photoIDs) == 0 {
		return nil
	}

	// Build placeholders for IN clause
	placeholders := make([]string, len(photoIDs))
	args := make([]interface{}, len(photoIDs)+1)
	args[0] = collectionID
	for i, id := range photoIDs {
		placeholders[i] = "$" + string(rune('2'+i))
		args[i+1] = id
	}

	// Use simpler approach with multiple deletes
	for _, photoID := range photoIDs {
		_, err := r.db.ExecContext(ctx, `DELETE FROM collection_photos WHERE collection_id = $1 AND photo_id = $2`, collectionID, photoID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *CollectionPhotoRepository) Reorder(ctx context.Context, collectionID string, photoIDs []string) error {
	// Update position for each photo in the given order
	for i, photoID := range photoIDs {
		query := `UPDATE collection_photos SET position = $3 WHERE collection_id = $1 AND photo_id = $2`
		_, err := r.db.ExecContext(ctx, query, collectionID, photoID, i)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *CollectionPhotoRepository) IsPhotoInCollection(ctx context.Context, collectionID, photoID string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM collection_photos WHERE collection_id = $1 AND photo_id = $2)`
	err := r.db.QueryRowContext(ctx, query, collectionID, photoID).Scan(&exists)
	return exists, err
}

func (r *CollectionPhotoRepository) GetMaxPosition(ctx context.Context, collectionID string) (int, error) {
	var maxPos sql.NullInt64
	query := `SELECT MAX(position) FROM collection_photos WHERE collection_id = $1`
	err := r.db.QueryRowContext(ctx, query, collectionID).Scan(&maxPos)
	if err != nil {
		return 0, err
	}
	if !maxPos.Valid {
		return 0, nil
	}
	return int(maxPos.Int64), nil
}

// GetCollectionsForPhoto returns all collection IDs that contain this photo
func (r *CollectionPhotoRepository) GetCollectionsForPhoto(ctx context.Context, photoID string) ([]string, error) {
	query := `SELECT collection_id FROM collection_photos WHERE photo_id = $1`

	rows, err := r.db.QueryContext(ctx, query, photoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var collectionIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		collectionIDs = append(collectionIDs, id)
	}
	return collectionIDs, rows.Err()
}

// Ensure strings import is used
var _ = strings.Join

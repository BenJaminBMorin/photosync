package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/photosync/server/internal/models"
)

// PhotoRepositoryPostgres handles photo persistence for PostgreSQL
type PhotoRepositoryPostgres struct {
	db *sql.DB
}

// NewPhotoRepositoryPostgres creates a new PhotoRepositoryPostgres
func NewPhotoRepositoryPostgres(db *sql.DB) *PhotoRepositoryPostgres {
	return &PhotoRepositoryPostgres{db: db}
}

// allColumns lists all columns for SELECT queries
const photoSelectColumns = `id, original_filename, stored_path, file_hash, file_size, date_taken, uploaded_at, user_id,
	thumb_small, thumb_medium, thumb_large,
	camera_make, camera_model, lens_model, focal_length, aperture, shutter_speed, iso, orientation,
	latitude, longitude, altitude, width, height, origin_device_id`

// scanPhoto scans a row into a Photo struct
func scanPhoto(scanner interface{ Scan(...interface{}) error }) (*models.Photo, error) {
	var photo models.Photo
	err := scanner.Scan(
		&photo.ID,
		&photo.OriginalFilename,
		&photo.StoredPath,
		&photo.FileHash,
		&photo.FileSize,
		&photo.DateTaken,
		&photo.UploadedAt,
		&photo.UserID,
		&photo.ThumbSmall,
		&photo.ThumbMedium,
		&photo.ThumbLarge,
		&photo.CameraMake,
		&photo.CameraModel,
		&photo.LensModel,
		&photo.FocalLength,
		&photo.Aperture,
		&photo.ShutterSpeed,
		&photo.ISO,
		&photo.Orientation,
		&photo.Latitude,
		&photo.Longitude,
		&photo.Altitude,
		&photo.Width,
		&photo.Height,
		&photo.OriginDeviceID,
	)
	return &photo, err
}

// GetByID retrieves a photo by its ID
func (r *PhotoRepositoryPostgres) GetByID(ctx context.Context, id string) (*models.Photo, error) {
	query := `SELECT ` + photoSelectColumns + ` FROM photos WHERE id = $1`

	photo, err := scanPhoto(r.db.QueryRowContext(ctx, query, id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return photo, nil
}

// GetByHash retrieves a photo by its file hash
func (r *PhotoRepositoryPostgres) GetByHash(ctx context.Context, hash string) (*models.Photo, error) {
	normalizedHash := strings.ToLower(hash)
	query := `SELECT ` + photoSelectColumns + ` FROM photos WHERE file_hash = $1`

	photo, err := scanPhoto(r.db.QueryRowContext(ctx, query, normalizedHash))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return photo, nil
}

// GetByHashAndUser retrieves a photo by hash for a specific user
func (r *PhotoRepositoryPostgres) GetByHashAndUser(ctx context.Context, hash, userID string) (*models.Photo, error) {
	normalizedHash := strings.ToLower(hash)
	query := `SELECT ` + photoSelectColumns + ` FROM photos WHERE file_hash = $1 AND user_id = $2`

	photo, err := scanPhoto(r.db.QueryRowContext(ctx, query, normalizedHash, userID))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return photo, nil
}

// GetExistingHashes returns which hashes from the list already exist
func (r *PhotoRepositoryPostgres) GetExistingHashes(ctx context.Context, hashes []string) ([]string, error) {
	if len(hashes) == 0 {
		return []string{}, nil
	}

	normalized := make([]string, len(hashes))
	for i, h := range hashes {
		normalized[i] = strings.ToLower(h)
	}

	placeholders := make([]string, len(normalized))
	args := make([]interface{}, len(normalized))
	for i, h := range normalized {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = h
	}

	query := `SELECT file_hash FROM photos WHERE file_hash IN (` + strings.Join(placeholders, ",") + `)`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var existing []string
	for rows.Next() {
		var hash string
		if err := rows.Scan(&hash); err != nil {
			return nil, err
		}
		existing = append(existing, hash)
	}

	if existing == nil {
		existing = []string{}
	}
	return existing, rows.Err()
}

// GetExistingHashesForUser returns which hashes from the list already exist for a user
func (r *PhotoRepositoryPostgres) GetExistingHashesForUser(ctx context.Context, hashes []string, userID string) ([]string, error) {
	if len(hashes) == 0 {
		return []string{}, nil
	}

	normalized := make([]string, len(hashes))
	for i, h := range hashes {
		normalized[i] = strings.ToLower(h)
	}

	placeholders := make([]string, len(normalized))
	args := make([]interface{}, len(normalized)+1)
	args[0] = userID
	for i, h := range normalized {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args[i+1] = h
	}

	query := `SELECT file_hash FROM photos WHERE user_id = $1 AND file_hash IN (` + strings.Join(placeholders, ",") + `)`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var existing []string
	for rows.Next() {
		var hash string
		if err := rows.Scan(&hash); err != nil {
			return nil, err
		}
		existing = append(existing, hash)
	}

	if existing == nil {
		existing = []string{}
	}
	return existing, rows.Err()
}

// GetAll retrieves photos with pagination
func (r *PhotoRepositoryPostgres) GetAll(ctx context.Context, skip, take int) ([]*models.Photo, error) {
	query := `SELECT ` + photoSelectColumns + ` FROM photos ORDER BY date_taken DESC LIMIT $1 OFFSET $2`

	rows, err := r.db.QueryContext(ctx, query, take, skip)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var photos []*models.Photo
	for rows.Next() {
		photo, err := scanPhoto(rows)
		if err != nil {
			return nil, err
		}
		photos = append(photos, photo)
	}

	if photos == nil {
		photos = []*models.Photo{}
	}
	return photos, rows.Err()
}

// GetAllForUser retrieves photos for a specific user with pagination
func (r *PhotoRepositoryPostgres) GetAllForUser(ctx context.Context, userID string, skip, take int) ([]*models.Photo, error) {
	query := `SELECT ` + photoSelectColumns + ` FROM photos WHERE user_id = $1 ORDER BY date_taken DESC LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, query, userID, take, skip)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var photos []*models.Photo
	for rows.Next() {
		photo, err := scanPhoto(rows)
		if err != nil {
			return nil, err
		}
		photos = append(photos, photo)
	}

	if photos == nil {
		photos = []*models.Photo{}
	}
	return photos, rows.Err()
}

// GetCount returns the total number of photos
func (r *PhotoRepositoryPostgres) GetCount(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM photos").Scan(&count)
	return count, err
}

// GetCountForUser returns the total number of photos for a specific user
func (r *PhotoRepositoryPostgres) GetCountForUser(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM photos WHERE user_id = $1", userID).Scan(&count)
	return count, err
}

// Add inserts a new photo with all metadata
func (r *PhotoRepositoryPostgres) Add(ctx context.Context, photo *models.Photo) error {
	query := `
		INSERT INTO photos (
			id, original_filename, stored_path, file_hash, file_size, date_taken, uploaded_at, user_id,
			thumb_small, thumb_medium, thumb_large,
			camera_make, camera_model, lens_model, focal_length, aperture, shutter_speed, iso, orientation,
			latitude, longitude, altitude, width, height, origin_device_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25)
	`

	_, err := r.db.ExecContext(ctx, query,
		photo.ID,
		photo.OriginalFilename,
		photo.StoredPath,
		photo.FileHash,
		photo.FileSize,
		photo.DateTaken,
		photo.UploadedAt,
		photo.UserID,
		photo.ThumbSmall,
		photo.ThumbMedium,
		photo.ThumbLarge,
		photo.CameraMake,
		photo.CameraModel,
		photo.LensModel,
		photo.FocalLength,
		photo.Aperture,
		photo.ShutterSpeed,
		photo.ISO,
		photo.Orientation,
		photo.Latitude,
		photo.Longitude,
		photo.Altitude,
		photo.Width,
		photo.Height,
		photo.OriginDeviceID,
	)

	return err
}

// AddWithUser inserts a new photo with user association
func (r *PhotoRepositoryPostgres) AddWithUser(ctx context.Context, photo *models.Photo, userID string) error {
	photo.UserID = &userID
	return r.Add(ctx, photo)
}

// Update updates an existing photo record
func (r *PhotoRepositoryPostgres) Update(ctx context.Context, photo *models.Photo) error {
	query := `
		UPDATE photos SET
			original_filename = $1,
			stored_path = $2,
			file_hash = $3,
			file_size = $4,
			date_taken = $5,
			user_id = $6,
			origin_device_id = $7,
			thumb_small = $8,
			thumb_medium = $9,
			thumb_large = $10,
			width = $11,
			height = $12,
			camera_make = $13,
			camera_model = $14,
			lens_model = $15,
			focal_length = $16,
			aperture = $17,
			shutter_speed = $18,
			iso = $19,
			orientation = $20,
			latitude = $21,
			longitude = $22,
			altitude = $23
		WHERE id = $24
	`

	_, err := r.db.ExecContext(ctx, query,
		photo.OriginalFilename,
		photo.StoredPath,
		photo.FileHash,
		photo.FileSize,
		photo.DateTaken,
		photo.UserID,
		photo.OriginDeviceID,
		photo.ThumbSmall,
		photo.ThumbMedium,
		photo.ThumbLarge,
		photo.Width,
		photo.Height,
		photo.CameraMake,
		photo.CameraModel,
		photo.LensModel,
		photo.FocalLength,
		photo.Aperture,
		photo.ShutterSpeed,
		photo.ISO,
		photo.Orientation,
		photo.Latitude,
		photo.Longitude,
		photo.Altitude,
		photo.ID,
	)

	return err
}

// Delete removes a photo by ID
func (r *PhotoRepositoryPostgres) Delete(ctx context.Context, id string) (bool, error) {
	result, err := r.db.ExecContext(ctx, "DELETE FROM photos WHERE id = $1", id)
	if err != nil {
		return false, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}

	return affected > 0, nil
}

// GetPhotosWithLocation returns photos that have GPS coordinates (for map view)
func (r *PhotoRepositoryPostgres) GetPhotosWithLocation(ctx context.Context, skip, take int) ([]*models.Photo, error) {
	query := `SELECT ` + photoSelectColumns + ` FROM photos WHERE latitude IS NOT NULL AND longitude IS NOT NULL ORDER BY date_taken DESC LIMIT $1 OFFSET $2`

	rows, err := r.db.QueryContext(ctx, query, take, skip)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var photos []*models.Photo
	for rows.Next() {
		photo, err := scanPhoto(rows)
		if err != nil {
			return nil, err
		}
		photos = append(photos, photo)
	}

	if photos == nil {
		photos = []*models.Photo{}
	}
	return photos, rows.Err()
}

// GetPhotosWithLocationForUser returns photos with GPS for a specific user
func (r *PhotoRepositoryPostgres) GetPhotosWithLocationForUser(ctx context.Context, userID string, skip, take int) ([]*models.Photo, error) {
	query := `SELECT ` + photoSelectColumns + ` FROM photos WHERE user_id = $1 AND latitude IS NOT NULL AND longitude IS NOT NULL ORDER BY date_taken DESC LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, query, userID, take, skip)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var photos []*models.Photo
	for rows.Next() {
		photo, err := scanPhoto(rows)
		if err != nil {
			return nil, err
		}
		photos = append(photos, photo)
	}

	if photos == nil {
		photos = []*models.Photo{}
	}
	return photos, rows.Err()
}

// GetLocationCount returns count of photos with GPS coordinates
func (r *PhotoRepositoryPostgres) GetLocationCount(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM photos WHERE latitude IS NOT NULL AND longitude IS NOT NULL").Scan(&count)
	return count, err
}

// GetLocationCountForUser returns count of photos with GPS for a specific user
func (r *PhotoRepositoryPostgres) GetLocationCountForUser(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM photos WHERE user_id = $1 AND latitude IS NOT NULL AND longitude IS NOT NULL", userID).Scan(&count)
	return count, err
}

// GetPhotosWithoutThumbnails returns photos that don't have thumbnails generated
func (r *PhotoRepositoryPostgres) GetPhotosWithoutThumbnails(ctx context.Context, limit int) ([]*models.Photo, error) {
	query := `SELECT ` + photoSelectColumns + ` FROM photos
		WHERE thumb_small IS NULL
		LIMIT $1`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var photos []*models.Photo
	for rows.Next() {
		photo, err := scanPhoto(rows)
		if err != nil {
			return nil, err
		}
		photos = append(photos, photo)
	}

	if photos == nil {
		photos = []*models.Photo{}
	}
	return photos, rows.Err()
}

// UpdateThumbnails updates the thumbnail paths for a photo
func (r *PhotoRepositoryPostgres) UpdateThumbnails(ctx context.Context, photoID, smallPath, mediumPath, largePath string) error {
	query := `UPDATE photos SET thumb_small = $1, thumb_medium = $2, thumb_large = $3 WHERE id = $4`
	_, err := r.db.ExecContext(ctx, query, smallPath, mediumPath, largePath, photoID)
	return err
}

// GetOrphanedPhotos returns photos that don't have an owner (user_id IS NULL)
func (r *PhotoRepositoryPostgres) GetOrphanedPhotos(ctx context.Context, limit int) ([]*models.Photo, error) {
	query := `SELECT ` + photoSelectColumns + ` FROM photos WHERE user_id IS NULL LIMIT $1`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var photos []*models.Photo
	for rows.Next() {
		photo, err := scanPhoto(rows)
		if err != nil {
			return nil, err
		}
		photos = append(photos, photo)
	}

	if photos == nil {
		photos = []*models.Photo{}
	}
	return photos, rows.Err()
}

// ============================================================================
// Sync-related methods
// ============================================================================

// GetAllForUserWithCursor returns photos for a user using cursor-based pagination
// Returns photos, next cursor, and error
func (r *PhotoRepositoryPostgres) GetAllForUserWithCursor(ctx context.Context, userID string, cursor string, limit int, sinceTimestamp *time.Time) ([]*models.Photo, string, error) {
	var query string
	var args []interface{}

	if cursor != "" {
		// Cursor is the last photo ID from previous page
		// We use uploaded_at + id for stable ordering
		if sinceTimestamp != nil {
			query = `SELECT ` + photoSelectColumns + ` FROM photos
				WHERE user_id = $1 AND uploaded_at >= $2 AND (uploaded_at, id) > (
					SELECT uploaded_at, id FROM photos WHERE id = $3
				)
				ORDER BY uploaded_at ASC, id ASC
				LIMIT $4`
			args = []interface{}{userID, *sinceTimestamp, cursor, limit + 1}
		} else {
			query = `SELECT ` + photoSelectColumns + ` FROM photos
				WHERE user_id = $1 AND (uploaded_at, id) > (
					SELECT uploaded_at, id FROM photos WHERE id = $2
				)
				ORDER BY uploaded_at ASC, id ASC
				LIMIT $3`
			args = []interface{}{userID, cursor, limit + 1}
		}
	} else {
		// First page
		if sinceTimestamp != nil {
			query = `SELECT ` + photoSelectColumns + ` FROM photos
				WHERE user_id = $1 AND uploaded_at >= $2
				ORDER BY uploaded_at ASC, id ASC
				LIMIT $3`
			args = []interface{}{userID, *sinceTimestamp, limit + 1}
		} else {
			query = `SELECT ` + photoSelectColumns + ` FROM photos
				WHERE user_id = $1
				ORDER BY uploaded_at ASC, id ASC
				LIMIT $2`
			args = []interface{}{userID, limit + 1}
		}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	var photos []*models.Photo
	for rows.Next() {
		photo, err := scanPhoto(rows)
		if err != nil {
			return nil, "", err
		}
		photos = append(photos, photo)
	}

	if err := rows.Err(); err != nil {
		return nil, "", err
	}

	// Check if there's a next page
	var nextCursor string
	if len(photos) > limit {
		// Has more pages, return cursor as the last item's ID
		nextCursor = photos[limit-1].ID
		photos = photos[:limit] // Trim the extra item
	}

	if photos == nil {
		photos = []*models.Photo{}
	}
	return photos, nextCursor, nil
}

// GetCountByOriginDevice returns count of photos from a specific device
func (r *PhotoRepositoryPostgres) GetCountByOriginDevice(ctx context.Context, userID, deviceID string) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM photos WHERE user_id = $1 AND origin_device_id = $2",
		userID, deviceID,
	).Scan(&count)
	return count, err
}

// GetLegacyPhotosForUser returns photos without an origin device (legacy photos)
func (r *PhotoRepositoryPostgres) GetLegacyPhotosForUser(ctx context.Context, userID string, limit int) ([]*models.Photo, error) {
	query := `SELECT ` + photoSelectColumns + ` FROM photos
		WHERE user_id = $1 AND origin_device_id IS NULL
		ORDER BY date_taken DESC
		LIMIT $2`

	rows, err := r.db.QueryContext(ctx, query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var photos []*models.Photo
	for rows.Next() {
		photo, err := scanPhoto(rows)
		if err != nil {
			return nil, err
		}
		photos = append(photos, photo)
	}

	if photos == nil {
		photos = []*models.Photo{}
	}
	return photos, rows.Err()
}

// GetLegacyPhotoCount returns count of photos without origin device
func (r *PhotoRepositoryPostgres) GetLegacyPhotoCount(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM photos WHERE user_id = $1 AND origin_device_id IS NULL",
		userID,
	).Scan(&count)
	return count, err
}

// ClaimLegacyPhotos sets origin_device_id for specific photos
func (r *PhotoRepositoryPostgres) ClaimLegacyPhotos(ctx context.Context, photoIDs []string, deviceID string) (int, error) {
	if len(photoIDs) == 0 {
		return 0, nil
	}

	// Build placeholder list
	placeholders := make([]string, len(photoIDs))
	args := make([]interface{}, len(photoIDs)+1)
	args[0] = deviceID
	for i, id := range photoIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args[i+1] = id
	}

	query := fmt.Sprintf(`UPDATE photos SET origin_device_id = $1
		WHERE id IN (%s) AND origin_device_id IS NULL`,
		strings.Join(placeholders, ", "))

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}

	affected, err := result.RowsAffected()
	return int(affected), err
}

// ClaimAllLegacyPhotos sets origin_device_id for all user's legacy photos
func (r *PhotoRepositoryPostgres) ClaimAllLegacyPhotos(ctx context.Context, userID, deviceID string) (int, error) {
	result, err := r.db.ExecContext(ctx,
		`UPDATE photos SET origin_device_id = $1
		WHERE user_id = $2 AND origin_device_id IS NULL`,
		deviceID, userID,
	)
	if err != nil {
		return 0, err
	}

	affected, err := result.RowsAffected()
	return int(affected), err
}

// SetOriginDevice sets the origin device for a photo
func (r *PhotoRepositoryPostgres) SetOriginDevice(ctx context.Context, photoID, deviceID string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE photos SET origin_device_id = $1 WHERE id = $2",
		deviceID, photoID,
	)
	return err
}

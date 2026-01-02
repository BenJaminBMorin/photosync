package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

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

// GetByID retrieves a photo by its ID
func (r *PhotoRepositoryPostgres) GetByID(ctx context.Context, id string) (*models.Photo, error) {
	query := `
		SELECT id, original_filename, stored_path, file_hash, file_size, date_taken, uploaded_at
		FROM photos WHERE id = $1
	`

	var photo models.Photo
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&photo.ID,
		&photo.OriginalFilename,
		&photo.StoredPath,
		&photo.FileHash,
		&photo.FileSize,
		&photo.DateTaken,
		&photo.UploadedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &photo, nil
}

// GetByHash retrieves a photo by its file hash
func (r *PhotoRepositoryPostgres) GetByHash(ctx context.Context, hash string) (*models.Photo, error) {
	normalizedHash := strings.ToLower(hash)
	query := `
		SELECT id, original_filename, stored_path, file_hash, file_size, date_taken, uploaded_at
		FROM photos WHERE file_hash = $1
	`

	var photo models.Photo
	err := r.db.QueryRowContext(ctx, query, normalizedHash).Scan(
		&photo.ID,
		&photo.OriginalFilename,
		&photo.StoredPath,
		&photo.FileHash,
		&photo.FileSize,
		&photo.DateTaken,
		&photo.UploadedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &photo, nil
}

// GetExistingHashes returns which hashes from the list already exist
func (r *PhotoRepositoryPostgres) GetExistingHashes(ctx context.Context, hashes []string) ([]string, error) {
	if len(hashes) == 0 {
		return []string{}, nil
	}

	// Normalize hashes
	normalized := make([]string, len(hashes))
	for i, h := range hashes {
		normalized[i] = strings.ToLower(h)
	}

	// Build query with numbered placeholders
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

// GetAll retrieves photos with pagination
func (r *PhotoRepositoryPostgres) GetAll(ctx context.Context, skip, take int) ([]*models.Photo, error) {
	query := `
		SELECT id, original_filename, stored_path, file_hash, file_size, date_taken, uploaded_at
		FROM photos
		ORDER BY date_taken DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.QueryContext(ctx, query, take, skip)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var photos []*models.Photo
	for rows.Next() {
		var photo models.Photo
		if err := rows.Scan(
			&photo.ID,
			&photo.OriginalFilename,
			&photo.StoredPath,
			&photo.FileHash,
			&photo.FileSize,
			&photo.DateTaken,
			&photo.UploadedAt,
		); err != nil {
			return nil, err
		}
		photos = append(photos, &photo)
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

// Add inserts a new photo
func (r *PhotoRepositoryPostgres) Add(ctx context.Context, photo *models.Photo) error {
	query := `
		INSERT INTO photos (id, original_filename, stored_path, file_hash, file_size, date_taken, uploaded_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.db.ExecContext(ctx, query,
		photo.ID,
		photo.OriginalFilename,
		photo.StoredPath,
		photo.FileHash,
		photo.FileSize,
		photo.DateTaken,
		photo.UploadedAt,
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

package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/photosync/server/internal/models"
)

// FileConflictRepository implements FileConflictRepo
type FileConflictRepository struct {
	db *sql.DB
}

// NewFileConflictRepository creates a new file conflict repository
func NewFileConflictRepository(db *sql.DB) *FileConflictRepository {
	return &FileConflictRepository{db: db}
}

// Add adds a new file conflict record
func (r *FileConflictRepository) Add(ctx context.Context, conflict *models.FileConflict) error {
	query := `
		INSERT INTO file_conflicts (
			id, photo_id, file_path, discovered_at, conflict_type,
			db_photo_id, db_user_id, db_device_id,
			file_photo_id, file_user_id, file_device_id,
			status, resolved_at, resolved_by, resolution_notes
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query,
		conflict.ID,
		conflict.PhotoID,
		conflict.FilePath,
		conflict.DiscoveredAt,
		conflict.ConflictType,
		conflict.DBPhotoID,
		conflict.DBUserID,
		conflict.DBDeviceID,
		conflict.FilePhotoID,
		conflict.FileUserID,
		conflict.FileDeviceID,
		conflict.Status,
		conflict.ResolvedAt,
		conflict.ResolvedBy,
		conflict.ResolutionNotes,
	)
	return err
}

// GetByID retrieves a file conflict by its ID
func (r *FileConflictRepository) GetByID(ctx context.Context, id string) (*models.FileConflict, error) {
	query := `
		SELECT id, photo_id, file_path, discovered_at, conflict_type,
			db_photo_id, db_user_id, db_device_id,
			file_photo_id, file_user_id, file_device_id,
			status, resolved_at, resolved_by, resolution_notes
		FROM file_conflicts
		WHERE id = ?
	`
	return r.scanFileConflict(r.db.QueryRowContext(ctx, query, id))
}

// GetByPhotoID retrieves all conflicts for a specific photo
func (r *FileConflictRepository) GetByPhotoID(ctx context.Context, photoID string) ([]*models.FileConflict, error) {
	query := `
		SELECT id, photo_id, file_path, discovered_at, conflict_type,
			db_photo_id, db_user_id, db_device_id,
			file_photo_id, file_user_id, file_device_id,
			status, resolved_at, resolved_by, resolution_notes
		FROM file_conflicts
		WHERE photo_id = ?
		ORDER BY discovered_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, photoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanFileConflicts(rows)
}

// Delete removes a file conflict record by ID
func (r *FileConflictRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM file_conflicts WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// GetAll retrieves all file conflicts with optional status filter
func (r *FileConflictRepository) GetAll(ctx context.Context, status string, skip, take int) ([]*models.FileConflict, int, error) {
	countQuery := `SELECT COUNT(*) FROM file_conflicts`
	dataQuery := `
		SELECT id, photo_id, file_path, discovered_at, conflict_type,
			db_photo_id, db_user_id, db_device_id,
			file_photo_id, file_user_id, file_device_id,
			status, resolved_at, resolved_by, resolution_notes
		FROM file_conflicts
	`

	args := []interface{}{}
	if status != "" {
		countQuery += ` WHERE status = ?`
		dataQuery += ` WHERE status = ?`
		args = append(args, status)
	}

	dataQuery += ` ORDER BY discovered_at DESC LIMIT ? OFFSET ?`

	// Get count
	var total int
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get data
	args = append(args, take, skip)
	rows, err := r.db.QueryContext(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	conflicts, err := r.scanFileConflicts(rows)
	return conflicts, total, err
}

// GetPending retrieves all pending file conflicts
func (r *FileConflictRepository) GetPending(ctx context.Context, skip, take int) ([]*models.FileConflict, int, error) {
	return r.GetAll(ctx, models.ConflictStatusPending, skip, take)
}

// Resolve marks a file conflict as resolved
func (r *FileConflictRepository) Resolve(ctx context.Context, id, status, resolvedBy string, notes *string) error {
	query := `
		UPDATE file_conflicts
		SET status = ?, resolved_at = ?, resolved_by = ?, resolution_notes = ?
		WHERE id = ?
	`
	_, err := r.db.ExecContext(ctx, query, status, time.Now().UTC(), resolvedBy, notes, id)
	return err
}

// GetStats returns statistics about file conflicts
func (r *FileConflictRepository) GetStats(ctx context.Context) (*models.FileConflictStats, error) {
	query := `
		SELECT
			COUNT(*) as total,
			SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) as pending,
			SUM(CASE WHEN status IN ('resolved_db', 'resolved_file') THEN 1 ELSE 0 END) as resolved,
			SUM(CASE WHEN status = 'ignored' THEN 1 ELSE 0 END) as ignored
		FROM file_conflicts
	`

	stats := &models.FileConflictStats{}
	err := r.db.QueryRowContext(ctx, query).Scan(
		&stats.TotalCount,
		&stats.PendingCount,
		&stats.ResolvedCount,
		&stats.IgnoredCount,
	)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

// scanFileConflict scans a single row into a FileConflict
func (r *FileConflictRepository) scanFileConflict(row *sql.Row) (*models.FileConflict, error) {
	conflict := &models.FileConflict{}
	var dbPhotoID, dbUserID, dbDeviceID sql.NullString
	var filePhotoID, fileUserID, fileDeviceID sql.NullString
	var resolvedAt sql.NullTime
	var resolvedBy, resolutionNotes sql.NullString

	err := row.Scan(
		&conflict.ID,
		&conflict.PhotoID,
		&conflict.FilePath,
		&conflict.DiscoveredAt,
		&conflict.ConflictType,
		&dbPhotoID,
		&dbUserID,
		&dbDeviceID,
		&filePhotoID,
		&fileUserID,
		&fileDeviceID,
		&conflict.Status,
		&resolvedAt,
		&resolvedBy,
		&resolutionNotes,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if dbPhotoID.Valid {
		conflict.DBPhotoID = &dbPhotoID.String
	}
	if dbUserID.Valid {
		conflict.DBUserID = &dbUserID.String
	}
	if dbDeviceID.Valid {
		conflict.DBDeviceID = &dbDeviceID.String
	}
	if filePhotoID.Valid {
		conflict.FilePhotoID = &filePhotoID.String
	}
	if fileUserID.Valid {
		conflict.FileUserID = &fileUserID.String
	}
	if fileDeviceID.Valid {
		conflict.FileDeviceID = &fileDeviceID.String
	}
	if resolvedAt.Valid {
		conflict.ResolvedAt = &resolvedAt.Time
	}
	if resolvedBy.Valid {
		conflict.ResolvedBy = &resolvedBy.String
	}
	if resolutionNotes.Valid {
		conflict.ResolutionNotes = &resolutionNotes.String
	}

	return conflict, nil
}

// scanFileConflicts scans multiple rows into FileConflict slice
func (r *FileConflictRepository) scanFileConflicts(rows *sql.Rows) ([]*models.FileConflict, error) {
	var conflicts []*models.FileConflict

	for rows.Next() {
		conflict := &models.FileConflict{}
		var dbPhotoID, dbUserID, dbDeviceID sql.NullString
		var filePhotoID, fileUserID, fileDeviceID sql.NullString
		var resolvedAt sql.NullTime
		var resolvedBy, resolutionNotes sql.NullString

		err := rows.Scan(
			&conflict.ID,
			&conflict.PhotoID,
			&conflict.FilePath,
			&conflict.DiscoveredAt,
			&conflict.ConflictType,
			&dbPhotoID,
			&dbUserID,
			&dbDeviceID,
			&filePhotoID,
			&fileUserID,
			&fileDeviceID,
			&conflict.Status,
			&resolvedAt,
			&resolvedBy,
			&resolutionNotes,
		)

		if err != nil {
			return nil, err
		}

		if dbPhotoID.Valid {
			conflict.DBPhotoID = &dbPhotoID.String
		}
		if dbUserID.Valid {
			conflict.DBUserID = &dbUserID.String
		}
		if dbDeviceID.Valid {
			conflict.DBDeviceID = &dbDeviceID.String
		}
		if filePhotoID.Valid {
			conflict.FilePhotoID = &filePhotoID.String
		}
		if fileUserID.Valid {
			conflict.FileUserID = &fileUserID.String
		}
		if fileDeviceID.Valid {
			conflict.FileDeviceID = &fileDeviceID.String
		}
		if resolvedAt.Valid {
			conflict.ResolvedAt = &resolvedAt.Time
		}
		if resolvedBy.Valid {
			conflict.ResolvedBy = &resolvedBy.String
		}
		if resolutionNotes.Valid {
			conflict.ResolutionNotes = &resolutionNotes.String
		}

		conflicts = append(conflicts, conflict)
	}

	return conflicts, rows.Err()
}

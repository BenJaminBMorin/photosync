package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/photosync/server/internal/models"
)

// OrphanFileRepository implements OrphanFileRepo
type OrphanFileRepository struct {
	db *sql.DB
}

// NewOrphanFileRepository creates a new orphan file repository
func NewOrphanFileRepository(db *sql.DB) *OrphanFileRepository {
	return &OrphanFileRepository{db: db}
}

// Add adds a new orphan file record
func (r *OrphanFileRepository) Add(ctx context.Context, orphan *models.OrphanFile) error {
	query := `
		INSERT INTO orphan_files (
			id, file_path, file_size, file_hash, discovered_at,
			embedded_photo_id, embedded_user_id, embedded_device_id, embedded_file_hash, embedded_uploaded_at,
			status, status_changed_at, status_changed_by, assigned_to_user, assigned_to_device
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query,
		orphan.ID,
		orphan.FilePath,
		orphan.FileSize,
		orphan.FileHash,
		orphan.DiscoveredAt,
		orphan.EmbeddedPhotoID,
		orphan.EmbeddedUserID,
		orphan.EmbeddedDeviceID,
		orphan.EmbeddedFileHash,
		orphan.EmbeddedUploadedAt,
		orphan.Status,
		orphan.StatusChangedAt,
		orphan.StatusChangedBy,
		orphan.AssignedToUser,
		orphan.AssignedToDevice,
	)
	return err
}

// GetByID retrieves an orphan file by its ID
func (r *OrphanFileRepository) GetByID(ctx context.Context, id string) (*models.OrphanFile, error) {
	query := `
		SELECT id, file_path, file_size, file_hash, discovered_at,
			embedded_photo_id, embedded_user_id, embedded_device_id, embedded_file_hash, embedded_uploaded_at,
			status, status_changed_at, status_changed_by, assigned_to_user, assigned_to_device
		FROM orphan_files
		WHERE id = ?
	`
	return r.scanOrphanFile(r.db.QueryRowContext(ctx, query, id))
}

// GetByPath retrieves an orphan file by its file path
func (r *OrphanFileRepository) GetByPath(ctx context.Context, path string) (*models.OrphanFile, error) {
	query := `
		SELECT id, file_path, file_size, file_hash, discovered_at,
			embedded_photo_id, embedded_user_id, embedded_device_id, embedded_file_hash, embedded_uploaded_at,
			status, status_changed_at, status_changed_by, assigned_to_user, assigned_to_device
		FROM orphan_files
		WHERE file_path = ?
	`
	return r.scanOrphanFile(r.db.QueryRowContext(ctx, query, path))
}

// Delete removes an orphan file record by ID
func (r *OrphanFileRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM orphan_files WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// GetAll retrieves all orphan files with optional status filter
func (r *OrphanFileRepository) GetAll(ctx context.Context, status string, skip, take int) ([]*models.OrphanFile, int, error) {
	countQuery := `SELECT COUNT(*) FROM orphan_files`
	dataQuery := `
		SELECT id, file_path, file_size, file_hash, discovered_at,
			embedded_photo_id, embedded_user_id, embedded_device_id, embedded_file_hash, embedded_uploaded_at,
			status, status_changed_at, status_changed_by, assigned_to_user, assigned_to_device
		FROM orphan_files
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

	orphans, err := r.scanOrphanFiles(rows)
	return orphans, total, err
}

// GetForUser retrieves orphan files for a specific user (based on embedded_user_id)
func (r *OrphanFileRepository) GetForUser(ctx context.Context, userID string, status string, skip, take int) ([]*models.OrphanFile, int, error) {
	countQuery := `SELECT COUNT(*) FROM orphan_files WHERE embedded_user_id = ?`
	dataQuery := `
		SELECT id, file_path, file_size, file_hash, discovered_at,
			embedded_photo_id, embedded_user_id, embedded_device_id, embedded_file_hash, embedded_uploaded_at,
			status, status_changed_at, status_changed_by, assigned_to_user, assigned_to_device
		FROM orphan_files
		WHERE embedded_user_id = ?
	`

	args := []interface{}{userID}
	if status != "" {
		countQuery += ` AND status = ?`
		dataQuery += ` AND status = ?`
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

	orphans, err := r.scanOrphanFiles(rows)
	return orphans, total, err
}

// GetUnassigned retrieves orphan files with no embedded user ID
func (r *OrphanFileRepository) GetUnassigned(ctx context.Context, skip, take int) ([]*models.OrphanFile, int, error) {
	countQuery := `SELECT COUNT(*) FROM orphan_files WHERE embedded_user_id IS NULL AND status = 'pending'`
	dataQuery := `
		SELECT id, file_path, file_size, file_hash, discovered_at,
			embedded_photo_id, embedded_user_id, embedded_device_id, embedded_file_hash, embedded_uploaded_at,
			status, status_changed_at, status_changed_by, assigned_to_user, assigned_to_device
		FROM orphan_files
		WHERE embedded_user_id IS NULL AND status = 'pending'
		ORDER BY discovered_at DESC
		LIMIT ? OFFSET ?
	`

	// Get count
	var total int
	err := r.db.QueryRowContext(ctx, countQuery).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get data
	rows, err := r.db.QueryContext(ctx, dataQuery, take, skip)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	orphans, err := r.scanOrphanFiles(rows)
	return orphans, total, err
}

// UpdateStatus updates the status of an orphan file
func (r *OrphanFileRepository) UpdateStatus(ctx context.Context, id, status, changedBy string) error {
	query := `
		UPDATE orphan_files
		SET status = ?, status_changed_at = ?, status_changed_by = ?
		WHERE id = ?
	`
	_, err := r.db.ExecContext(ctx, query, status, time.Now().UTC(), changedBy, id)
	return err
}

// AssignToUser assigns an orphan file to a user and device
func (r *OrphanFileRepository) AssignToUser(ctx context.Context, id, userID, deviceID, assignedBy string) error {
	query := `
		UPDATE orphan_files
		SET assigned_to_user = ?, assigned_to_device = ?, status = ?, status_changed_at = ?, status_changed_by = ?
		WHERE id = ?
	`
	var device interface{}
	if deviceID == "" {
		device = nil
	} else {
		device = deviceID
	}
	_, err := r.db.ExecContext(ctx, query, userID, device, models.OrphanStatusClaimed, time.Now().UTC(), assignedBy, id)
	return err
}

// BulkUpdateStatus updates the status of multiple orphan files
func (r *OrphanFileRepository) BulkUpdateStatus(ctx context.Context, ids []string, status, changedBy string) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids)+3)
	args[0] = status
	args[1] = time.Now().UTC()
	args[2] = changedBy
	for i, id := range ids {
		placeholders[i] = "?"
		args[i+3] = id
	}

	query := fmt.Sprintf(`
		UPDATE orphan_files
		SET status = ?, status_changed_at = ?, status_changed_by = ?
		WHERE id IN (%s)
	`, strings.Join(placeholders, ","))

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}

	rows, err := result.RowsAffected()
	return int(rows), err
}

// BulkAssign assigns multiple orphan files to a user
func (r *OrphanFileRepository) BulkAssign(ctx context.Context, ids []string, userID, deviceID, assignedBy string) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids)+5)
	args[0] = userID
	if deviceID == "" {
		args[1] = nil
	} else {
		args[1] = deviceID
	}
	args[2] = models.OrphanStatusClaimed
	args[3] = time.Now().UTC()
	args[4] = assignedBy
	for i, id := range ids {
		placeholders[i] = "?"
		args[i+5] = id
	}

	query := fmt.Sprintf(`
		UPDATE orphan_files
		SET assigned_to_user = ?, assigned_to_device = ?, status = ?, status_changed_at = ?, status_changed_by = ?
		WHERE id IN (%s)
	`, strings.Join(placeholders, ","))

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}

	rows, err := result.RowsAffected()
	return int(rows), err
}

// BulkDelete deletes multiple orphan file records
func (r *OrphanFileRepository) BulkDelete(ctx context.Context, ids []string) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf(`DELETE FROM orphan_files WHERE id IN (%s)`, strings.Join(placeholders, ","))

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}

	rows, err := result.RowsAffected()
	return int(rows), err
}

// GetStats returns statistics about orphan files
func (r *OrphanFileRepository) GetStats(ctx context.Context) (*models.OrphanFileStats, error) {
	query := `
		SELECT
			COUNT(*) as total,
			SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) as pending,
			SUM(CASE WHEN status = 'ignored' THEN 1 ELSE 0 END) as ignored,
			SUM(CASE WHEN status = 'claimed' THEN 1 ELSE 0 END) as claimed
		FROM orphan_files
	`

	stats := &models.OrphanFileStats{}
	err := r.db.QueryRowContext(ctx, query).Scan(
		&stats.TotalCount,
		&stats.PendingCount,
		&stats.IgnoredCount,
		&stats.ClaimedCount,
	)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

// scanOrphanFile scans a single row into an OrphanFile
func (r *OrphanFileRepository) scanOrphanFile(row *sql.Row) (*models.OrphanFile, error) {
	orphan := &models.OrphanFile{}
	var fileHash, embeddedPhotoID, embeddedUserID, embeddedDeviceID, embeddedFileHash sql.NullString
	var embeddedUploadedAt, statusChangedAt sql.NullTime
	var statusChangedBy, assignedToUser, assignedToDevice sql.NullString

	err := row.Scan(
		&orphan.ID,
		&orphan.FilePath,
		&orphan.FileSize,
		&fileHash,
		&orphan.DiscoveredAt,
		&embeddedPhotoID,
		&embeddedUserID,
		&embeddedDeviceID,
		&embeddedFileHash,
		&embeddedUploadedAt,
		&orphan.Status,
		&statusChangedAt,
		&statusChangedBy,
		&assignedToUser,
		&assignedToDevice,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if fileHash.Valid {
		orphan.FileHash = &fileHash.String
	}
	if embeddedPhotoID.Valid {
		orphan.EmbeddedPhotoID = &embeddedPhotoID.String
	}
	if embeddedUserID.Valid {
		orphan.EmbeddedUserID = &embeddedUserID.String
	}
	if embeddedDeviceID.Valid {
		orphan.EmbeddedDeviceID = &embeddedDeviceID.String
	}
	if embeddedFileHash.Valid {
		orphan.EmbeddedFileHash = &embeddedFileHash.String
	}
	if embeddedUploadedAt.Valid {
		orphan.EmbeddedUploadedAt = &embeddedUploadedAt.Time
	}
	if statusChangedAt.Valid {
		orphan.StatusChangedAt = &statusChangedAt.Time
	}
	if statusChangedBy.Valid {
		orphan.StatusChangedBy = &statusChangedBy.String
	}
	if assignedToUser.Valid {
		orphan.AssignedToUser = &assignedToUser.String
	}
	if assignedToDevice.Valid {
		orphan.AssignedToDevice = &assignedToDevice.String
	}

	return orphan, nil
}

// scanOrphanFiles scans multiple rows into OrphanFile slice
func (r *OrphanFileRepository) scanOrphanFiles(rows *sql.Rows) ([]*models.OrphanFile, error) {
	var orphans []*models.OrphanFile

	for rows.Next() {
		orphan := &models.OrphanFile{}
		var fileHash, embeddedPhotoID, embeddedUserID, embeddedDeviceID, embeddedFileHash sql.NullString
		var embeddedUploadedAt, statusChangedAt sql.NullTime
		var statusChangedBy, assignedToUser, assignedToDevice sql.NullString

		err := rows.Scan(
			&orphan.ID,
			&orphan.FilePath,
			&orphan.FileSize,
			&fileHash,
			&orphan.DiscoveredAt,
			&embeddedPhotoID,
			&embeddedUserID,
			&embeddedDeviceID,
			&embeddedFileHash,
			&embeddedUploadedAt,
			&orphan.Status,
			&statusChangedAt,
			&statusChangedBy,
			&assignedToUser,
			&assignedToDevice,
		)

		if err != nil {
			return nil, err
		}

		if fileHash.Valid {
			orphan.FileHash = &fileHash.String
		}
		if embeddedPhotoID.Valid {
			orphan.EmbeddedPhotoID = &embeddedPhotoID.String
		}
		if embeddedUserID.Valid {
			orphan.EmbeddedUserID = &embeddedUserID.String
		}
		if embeddedDeviceID.Valid {
			orphan.EmbeddedDeviceID = &embeddedDeviceID.String
		}
		if embeddedFileHash.Valid {
			orphan.EmbeddedFileHash = &embeddedFileHash.String
		}
		if embeddedUploadedAt.Valid {
			orphan.EmbeddedUploadedAt = &embeddedUploadedAt.Time
		}
		if statusChangedAt.Valid {
			orphan.StatusChangedAt = &statusChangedAt.Time
		}
		if statusChangedBy.Valid {
			orphan.StatusChangedBy = &statusChangedBy.String
		}
		if assignedToUser.Valid {
			orphan.AssignedToUser = &assignedToUser.String
		}
		if assignedToDevice.Valid {
			orphan.AssignedToDevice = &assignedToDevice.String
		}

		orphans = append(orphans, orphan)
	}

	return orphans, rows.Err()
}

package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/photosync/server/internal/models"
)

// DeviceRepository implements DeviceRepo for PostgreSQL/SQLite
type DeviceRepository struct {
	db *sql.DB
}

// NewDeviceRepository creates a new DeviceRepository
func NewDeviceRepository(db *sql.DB) *DeviceRepository {
	return &DeviceRepository{db: db}
}

func (r *DeviceRepository) GetByID(ctx context.Context, id string) (*models.Device, error) {
	query := `SELECT id, user_id, device_name, platform, fcm_token, registered_at, last_seen_at, is_active
			  FROM devices WHERE id = $1`

	var device models.Device
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&device.ID, &device.UserID, &device.DeviceName, &device.Platform,
		&device.FCMToken, &device.RegisteredAt, &device.LastSeenAt, &device.IsActive,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &device, nil
}

func (r *DeviceRepository) GetByFCMToken(ctx context.Context, fcmToken string) (*models.Device, error) {
	query := `SELECT id, user_id, device_name, platform, fcm_token, registered_at, last_seen_at, is_active
			  FROM devices WHERE fcm_token = $1`

	var device models.Device
	err := r.db.QueryRowContext(ctx, query, fcmToken).Scan(
		&device.ID, &device.UserID, &device.DeviceName, &device.Platform,
		&device.FCMToken, &device.RegisteredAt, &device.LastSeenAt, &device.IsActive,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &device, nil
}

func (r *DeviceRepository) GetAllForUser(ctx context.Context, userID string) ([]*models.Device, error) {
	query := `SELECT id, user_id, device_name, platform, fcm_token, registered_at, last_seen_at, is_active
			  FROM devices WHERE user_id = $1 ORDER BY last_seen_at DESC`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []*models.Device
	for rows.Next() {
		var device models.Device
		if err := rows.Scan(&device.ID, &device.UserID, &device.DeviceName, &device.Platform,
			&device.FCMToken, &device.RegisteredAt, &device.LastSeenAt, &device.IsActive); err != nil {
			return nil, err
		}
		devices = append(devices, &device)
	}
	return devices, rows.Err()
}

func (r *DeviceRepository) GetActiveForUser(ctx context.Context, userID string) ([]*models.Device, error) {
	query := `SELECT id, user_id, device_name, platform, fcm_token, registered_at, last_seen_at, is_active
			  FROM devices WHERE user_id = $1 AND is_active = true ORDER BY last_seen_at DESC`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []*models.Device
	for rows.Next() {
		var device models.Device
		if err := rows.Scan(&device.ID, &device.UserID, &device.DeviceName, &device.Platform,
			&device.FCMToken, &device.RegisteredAt, &device.LastSeenAt, &device.IsActive); err != nil {
			return nil, err
		}
		devices = append(devices, &device)
	}
	return devices, rows.Err()
}

func (r *DeviceRepository) Add(ctx context.Context, device *models.Device) error {
	query := `INSERT INTO devices (id, user_id, device_name, platform, fcm_token, registered_at, last_seen_at, is_active)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := r.db.ExecContext(ctx, query,
		device.ID, device.UserID, device.DeviceName, device.Platform,
		device.FCMToken, device.RegisteredAt, device.LastSeenAt, device.IsActive,
	)
	return err
}

func (r *DeviceRepository) UpdateToken(ctx context.Context, id, fcmToken string) error {
	query := `UPDATE devices SET fcm_token = $2, last_seen_at = $3 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, fcmToken, time.Now().UTC())
	return err
}

func (r *DeviceRepository) UpdateLastSeen(ctx context.Context, id string) error {
	query := `UPDATE devices SET last_seen_at = $2 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, time.Now().UTC())
	return err
}

func (r *DeviceRepository) Deactivate(ctx context.Context, id string) error {
	query := `UPDATE devices SET is_active = false WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *DeviceRepository) Delete(ctx context.Context, id string) (bool, error) {
	result, err := r.db.ExecContext(ctx, `DELETE FROM devices WHERE id = $1`, id)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	return rows > 0, err
}

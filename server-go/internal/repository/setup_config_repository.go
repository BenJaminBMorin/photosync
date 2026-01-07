package repository

import (
	"context"
	"database/sql"
	"time"
)

const (
	SetupKeyComplete       = "setup_complete"
	SetupKeyFirebaseConfig = "firebase_configured"
	SetupKeyAdminCreated   = "admin_created"
	SetupKeyAppName        = "app_name"
)

// SetupConfigRepository implements SetupConfigRepo
type SetupConfigRepository struct {
	db *sql.DB
}

// NewSetupConfigRepository creates a new SetupConfigRepository
func NewSetupConfigRepository(db *sql.DB) *SetupConfigRepository {
	return &SetupConfigRepository{db: db}
}

func (r *SetupConfigRepository) Get(ctx context.Context, key string) (string, error) {
	var value string
	err := r.db.QueryRowContext(ctx, `SELECT value FROM setup_config WHERE key = $1`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

func (r *SetupConfigRepository) Set(ctx context.Context, key, value string) error {
	query := `
		INSERT INTO setup_config (key, value, updated_at) VALUES ($1, $2, $3)
		ON CONFLICT (key) DO UPDATE SET value = $2, updated_at = $3
	`
	_, err := r.db.ExecContext(ctx, query, key, value, time.Now().UTC())
	return err
}

func (r *SetupConfigRepository) GetAll(ctx context.Context) (map[string]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT key, value FROM setup_config`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		result[key] = value
	}
	return result, rows.Err()
}

func (r *SetupConfigRepository) IsSetupComplete(ctx context.Context) (bool, error) {
	value, err := r.Get(ctx, SetupKeyComplete)
	if err != nil {
		return false, err
	}
	return value == "true", nil
}

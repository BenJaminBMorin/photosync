package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/photosync/server/internal/models"
)

// ConfigOverrideRepository implements ConfigOverrideRepo
type ConfigOverrideRepository struct {
	db *sql.DB
}

// NewConfigOverrideRepository creates a new config override repository
func NewConfigOverrideRepository(db *sql.DB) *ConfigOverrideRepository {
	return &ConfigOverrideRepository{db: db}
}

// Get retrieves a single config item by key
func (r *ConfigOverrideRepository) Get(ctx context.Context, key string) (*models.ConfigItem, error) {
	query := `
		SELECT key, value, value_type, category, requires_restart, is_sensitive, updated_at
		FROM config_overrides
		WHERE key = ?
	`

	item := &models.ConfigItem{}
	err := r.db.QueryRowContext(ctx, query, key).Scan(
		&item.Key,
		&item.Value,
		&item.ValueType,
		&item.Category,
		&item.RequiresRestart,
		&item.IsSensitive,
		&item.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return item, nil
}

// GetAll retrieves all config items
func (r *ConfigOverrideRepository) GetAll(ctx context.Context) ([]*models.ConfigItem, error) {
	query := `
		SELECT key, value, value_type, category, requires_restart, is_sensitive, updated_at
		FROM config_overrides
		ORDER BY category, key
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*models.ConfigItem
	for rows.Next() {
		item := &models.ConfigItem{}
		err := rows.Scan(
			&item.Key,
			&item.Value,
			&item.ValueType,
			&item.Category,
			&item.RequiresRestart,
			&item.IsSensitive,
			&item.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

// GetByCategory retrieves all config items in a specific category
func (r *ConfigOverrideRepository) GetByCategory(ctx context.Context, category models.ConfigCategory) ([]*models.ConfigItem, error) {
	query := `
		SELECT key, value, value_type, category, requires_restart, is_sensitive, updated_at
		FROM config_overrides
		WHERE category = ?
		ORDER BY key
	`

	rows, err := r.db.QueryContext(ctx, query, string(category))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*models.ConfigItem
	for rows.Next() {
		item := &models.ConfigItem{}
		err := rows.Scan(
			&item.Key,
			&item.Value,
			&item.ValueType,
			&item.Category,
			&item.RequiresRestart,
			&item.IsSensitive,
			&item.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

// Set creates or updates a config item
func (r *ConfigOverrideRepository) Set(ctx context.Context, key, value, valueType string, category models.ConfigCategory, requiresRestart, isSensitive bool, updatedBy string) error {
	query := `
		INSERT INTO config_overrides (key, value, value_type, category, requires_restart, is_sensitive, updated_at, updated_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET
			value = ?,
			value_type = ?,
			category = ?,
			requires_restart = ?,
			is_sensitive = ?,
			updated_at = ?,
			updated_by = ?
	`

	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, query,
		key, value, valueType, string(category), requiresRestart, isSensitive, now, updatedBy,
		value, valueType, string(category), requiresRestart, isSensitive, now, updatedBy,
	)
	return err
}

// Delete deletes a config item
func (r *ConfigOverrideRepository) Delete(ctx context.Context, key string) error {
	query := `DELETE FROM config_overrides WHERE key = ?`
	_, err := r.db.ExecContext(ctx, query, key)
	return err
}

// HasRestartRequired checks if any config item has requires_restart = true
func (r *ConfigOverrideRepository) HasRestartRequired(ctx context.Context) (bool, error) {
	query := `
		SELECT COUNT(*) FROM config_overrides
		WHERE requires_restart = 1
	`

	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

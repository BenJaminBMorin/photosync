package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/photosync/server/internal/models"
)

// UserPreferencesRepository defines operations for user preferences persistence
type UserPreferencesRepository interface {
	Get(ctx context.Context, userID string) (*models.UserPreferences, error)
	CreateOrUpdate(ctx context.Context, prefs *models.UserPreferences) error
	Delete(ctx context.Context, userID string) error
}

type userPreferencesRepository struct {
	db *sql.DB
}

// NewUserPreferencesRepository creates a new user preferences repository
func NewUserPreferencesRepository(db *sql.DB) UserPreferencesRepository {
	return &userPreferencesRepository{db: db}
}

// Get retrieves user preferences by user ID
func (r *userPreferencesRepository) Get(ctx context.Context, userID string) (*models.UserPreferences, error) {
	query := `
		SELECT user_id, global_theme_id, created_at, updated_at
		FROM user_preferences
		WHERE user_id = $1
	`

	var prefs models.UserPreferences
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&prefs.UserID,
		&prefs.GlobalThemeID,
		&prefs.CreatedAt,
		&prefs.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, models.ErrPreferencesNotFound
	}

	if err != nil {
		return nil, err
	}

	return &prefs, nil
}

// CreateOrUpdate creates or updates user preferences (upsert)
func (r *userPreferencesRepository) CreateOrUpdate(ctx context.Context, prefs *models.UserPreferences) error {
	prefs.UpdatedAt = time.Now()

	// Try PostgreSQL upsert syntax first
	query := `
		INSERT INTO user_preferences (user_id, global_theme_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id) DO UPDATE
		SET global_theme_id = EXCLUDED.global_theme_id,
		    updated_at = EXCLUDED.updated_at
	`

	_, err := r.db.ExecContext(ctx, query,
		prefs.UserID,
		prefs.GlobalThemeID,
		prefs.CreatedAt,
		prefs.UpdatedAt,
	)

	// If PostgreSQL upsert fails, try SQLite upsert syntax
	if err != nil {
		query = `
			INSERT INTO user_preferences (user_id, global_theme_id, created_at, updated_at)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (user_id) DO UPDATE
			SET global_theme_id = excluded.global_theme_id,
			    updated_at = excluded.updated_at
		`

		_, err = r.db.ExecContext(ctx, query,
			prefs.UserID,
			prefs.GlobalThemeID,
			prefs.CreatedAt,
			prefs.UpdatedAt,
		)
	}

	return err
}

// Delete deletes user preferences
func (r *userPreferencesRepository) Delete(ctx context.Context, userID string) error {
	query := `DELETE FROM user_preferences WHERE user_id = $1`

	result, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return models.ErrPreferencesNotFound
	}

	return nil
}

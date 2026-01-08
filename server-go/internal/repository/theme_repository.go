package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/photosync/server/internal/models"
)

// ThemeRepository defines operations for theme persistence
type ThemeRepository interface {
	GetAll(ctx context.Context) ([]*models.Theme, error)
	GetByID(ctx context.Context, id string) (*models.Theme, error)
	Create(ctx context.Context, theme *models.Theme) error
	Update(ctx context.Context, theme *models.Theme) error
	Delete(ctx context.Context, id string) error
	GetSystemThemes(ctx context.Context) ([]*models.Theme, error)
}

type themeRepository struct {
	db *sql.DB
}

// NewThemeRepository creates a new theme repository
func NewThemeRepository(db *sql.DB) ThemeRepository {
	return &themeRepository{db: db}
}

// GetAll retrieves all themes
func (r *themeRepository) GetAll(ctx context.Context) ([]*models.Theme, error) {
	query := `
		SELECT id, name, description, is_system, created_by, properties, created_at, updated_at
		FROM themes
		ORDER BY is_system DESC, name ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var themes []*models.Theme
	for rows.Next() {
		theme, err := r.scanTheme(rows)
		if err != nil {
			return nil, err
		}
		themes = append(themes, theme)
	}

	return themes, rows.Err()
}

// GetByID retrieves a theme by its ID
func (r *themeRepository) GetByID(ctx context.Context, id string) (*models.Theme, error) {
	query := `
		SELECT id, name, description, is_system, created_by, properties, created_at, updated_at
		FROM themes
		WHERE id = $1
	`

	row := r.db.QueryRowContext(ctx, query, id)
	return r.scanTheme(row)
}

// Create creates a new theme
func (r *themeRepository) Create(ctx context.Context, theme *models.Theme) error {
	propertiesJSON, err := json.Marshal(theme.Properties)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO themes (id, name, description, is_system, created_by, properties, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err = r.db.ExecContext(ctx, query,
		theme.ID,
		theme.Name,
		theme.Description,
		theme.IsSystem,
		theme.CreatedBy,
		string(propertiesJSON),
		theme.CreatedAt,
		theme.UpdatedAt,
	)

	return err
}

// Update updates an existing theme
func (r *themeRepository) Update(ctx context.Context, theme *models.Theme) error {
	// Check if it's a system theme (cannot be modified)
	existing, err := r.GetByID(ctx, theme.ID)
	if err != nil {
		return err
	}
	if existing.IsSystem {
		return models.ErrSystemThemeEdit
	}

	propertiesJSON, err := json.Marshal(theme.Properties)
	if err != nil {
		return err
	}

	query := `
		UPDATE themes
		SET name = $1, description = $2, properties = $3, updated_at = $4
		WHERE id = $5 AND is_system = FALSE
	`

	theme.UpdatedAt = time.Now()

	result, err := r.db.ExecContext(ctx, query,
		theme.Name,
		theme.Description,
		string(propertiesJSON),
		theme.UpdatedAt,
		theme.ID,
	)

	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return models.ErrThemeNotFound
	}

	return nil
}

// Delete deletes a theme by ID (only non-system themes)
func (r *themeRepository) Delete(ctx context.Context, id string) error {
	// Check if it's a system theme (cannot be deleted)
	existing, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if existing.IsSystem {
		return models.ErrSystemThemeEdit
	}

	query := `DELETE FROM themes WHERE id = $1 AND is_system = FALSE`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return models.ErrThemeNotFound
	}

	return nil
}

// GetSystemThemes retrieves only system themes
func (r *themeRepository) GetSystemThemes(ctx context.Context) ([]*models.Theme, error) {
	query := `
		SELECT id, name, description, is_system, created_by, properties, created_at, updated_at
		FROM themes
		WHERE is_system = TRUE
		ORDER BY name ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var themes []*models.Theme
	for rows.Next() {
		theme, err := r.scanTheme(rows)
		if err != nil {
			return nil, err
		}
		themes = append(themes, theme)
	}

	return themes, rows.Err()
}

// scanTheme scans a row into a Theme struct
func (r *themeRepository) scanTheme(scanner interface {
	Scan(dest ...interface{}) error
}) (*models.Theme, error) {
	var theme models.Theme
	var propertiesJSON string
	var isSystemInt interface{} // Handle both BOOLEAN (postgres) and INTEGER (sqlite)

	err := scanner.Scan(
		&theme.ID,
		&theme.Name,
		&theme.Description,
		&isSystemInt,
		&theme.CreatedBy,
		&propertiesJSON,
		&theme.CreatedAt,
		&theme.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, models.ErrThemeNotFound
		}
		return nil, err
	}

	// Handle is_system field (boolean in postgres, integer in sqlite)
	switch v := isSystemInt.(type) {
	case bool:
		theme.IsSystem = v
	case int64:
		theme.IsSystem = v != 0
	}

	// Parse properties JSON
	if err := json.Unmarshal([]byte(propertiesJSON), &theme.Properties); err != nil {
		return nil, err
	}

	return &theme, nil
}

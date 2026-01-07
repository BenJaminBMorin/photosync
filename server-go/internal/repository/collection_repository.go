package repository

import (
	"context"
	"database/sql"

	"github.com/photosync/server/internal/models"
)

// CollectionRepository implements CollectionRepo for PostgreSQL/SQLite
type CollectionRepository struct {
	db *sql.DB
}

// NewCollectionRepository creates a new CollectionRepository
func NewCollectionRepository(db *sql.DB) *CollectionRepository {
	return &CollectionRepository{db: db}
}

func (r *CollectionRepository) GetByID(ctx context.Context, id string) (*models.Collection, error) {
	query := `SELECT id, user_id, name, description, slug, theme, custom_css, visibility,
			  secret_token, cover_photo_id, created_at, updated_at
			  FROM collections WHERE id = $1`

	var c models.Collection
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&c.ID, &c.UserID, &c.Name, &c.Description, &c.Slug, &c.Theme, &c.CustomCSS,
		&c.Visibility, &c.SecretToken, &c.CoverPhotoID, &c.CreatedAt, &c.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *CollectionRepository) GetBySlug(ctx context.Context, slug string) (*models.Collection, error) {
	query := `SELECT id, user_id, name, description, slug, theme, custom_css, visibility,
			  secret_token, cover_photo_id, created_at, updated_at
			  FROM collections WHERE slug = $1`

	var c models.Collection
	err := r.db.QueryRowContext(ctx, query, slug).Scan(
		&c.ID, &c.UserID, &c.Name, &c.Description, &c.Slug, &c.Theme, &c.CustomCSS,
		&c.Visibility, &c.SecretToken, &c.CoverPhotoID, &c.CreatedAt, &c.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *CollectionRepository) GetBySecretToken(ctx context.Context, token string) (*models.Collection, error) {
	query := `SELECT id, user_id, name, description, slug, theme, custom_css, visibility,
			  secret_token, cover_photo_id, created_at, updated_at
			  FROM collections WHERE secret_token = $1 AND visibility = 'secret_link'`

	var c models.Collection
	err := r.db.QueryRowContext(ctx, query, token).Scan(
		&c.ID, &c.UserID, &c.Name, &c.Description, &c.Slug, &c.Theme, &c.CustomCSS,
		&c.Visibility, &c.SecretToken, &c.CoverPhotoID, &c.CreatedAt, &c.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *CollectionRepository) GetAllForUser(ctx context.Context, userID string) ([]*models.Collection, error) {
	query := `SELECT c.id, c.user_id, c.name, c.description, c.slug, c.theme, c.custom_css,
			  c.visibility, c.secret_token, c.cover_photo_id, c.created_at, c.updated_at,
			  (SELECT COUNT(*) FROM collection_photos WHERE collection_id = c.id) as photo_count
			  FROM collections c WHERE c.user_id = $1 ORDER BY c.updated_at DESC`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var collections []*models.Collection
	for rows.Next() {
		var c models.Collection
		if err := rows.Scan(&c.ID, &c.UserID, &c.Name, &c.Description, &c.Slug, &c.Theme,
			&c.CustomCSS, &c.Visibility, &c.SecretToken, &c.CoverPhotoID, &c.CreatedAt,
			&c.UpdatedAt, &c.PhotoCount); err != nil {
			return nil, err
		}
		c.IsOwner = true
		collections = append(collections, &c)
	}
	return collections, rows.Err()
}

func (r *CollectionRepository) GetSharedWithUser(ctx context.Context, userID string) ([]*models.Collection, error) {
	query := `SELECT c.id, c.user_id, c.name, c.description, c.slug, c.theme, c.custom_css,
			  c.visibility, c.secret_token, c.cover_photo_id, c.created_at, c.updated_at,
			  (SELECT COUNT(*) FROM collection_photos WHERE collection_id = c.id) as photo_count
			  FROM collections c
			  INNER JOIN collection_shares cs ON cs.collection_id = c.id
			  WHERE cs.user_id = $1 ORDER BY c.updated_at DESC`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var collections []*models.Collection
	for rows.Next() {
		var c models.Collection
		if err := rows.Scan(&c.ID, &c.UserID, &c.Name, &c.Description, &c.Slug, &c.Theme,
			&c.CustomCSS, &c.Visibility, &c.SecretToken, &c.CoverPhotoID, &c.CreatedAt,
			&c.UpdatedAt, &c.PhotoCount); err != nil {
			return nil, err
		}
		c.IsOwner = false
		collections = append(collections, &c)
	}
	return collections, rows.Err()
}

func (r *CollectionRepository) Add(ctx context.Context, collection *models.Collection) error {
	query := `INSERT INTO collections (id, user_id, name, description, slug, theme, custom_css,
			  visibility, secret_token, cover_photo_id, created_at, updated_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	_, err := r.db.ExecContext(ctx, query,
		collection.ID, collection.UserID, collection.Name, collection.Description,
		collection.Slug, collection.Theme, collection.CustomCSS, collection.Visibility,
		collection.SecretToken, collection.CoverPhotoID, collection.CreatedAt, collection.UpdatedAt,
	)
	return err
}

func (r *CollectionRepository) Update(ctx context.Context, collection *models.Collection) error {
	query := `UPDATE collections SET name = $2, description = $3, slug = $4, theme = $5,
			  custom_css = $6, visibility = $7, secret_token = $8, cover_photo_id = $9, updated_at = $10
			  WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query,
		collection.ID, collection.Name, collection.Description, collection.Slug,
		collection.Theme, collection.CustomCSS, collection.Visibility, collection.SecretToken,
		collection.CoverPhotoID, collection.UpdatedAt,
	)
	return err
}

func (r *CollectionRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM collections WHERE id = $1`, id)
	return err
}

func (r *CollectionRepository) SlugExists(ctx context.Context, slug string, excludeID string) (bool, error) {
	var query string
	var args []interface{}

	if excludeID == "" {
		query = `SELECT EXISTS(SELECT 1 FROM collections WHERE slug = $1)`
		args = []interface{}{slug}
	} else {
		query = `SELECT EXISTS(SELECT 1 FROM collections WHERE slug = $1 AND id != $2)`
		args = []interface{}{slug, excludeID}
	}

	var exists bool
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&exists)
	return exists, err
}

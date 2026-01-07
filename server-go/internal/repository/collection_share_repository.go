package repository

import (
	"context"
	"database/sql"

	"github.com/photosync/server/internal/models"
)

// CollectionShareRepository implements CollectionShareRepo for PostgreSQL/SQLite
type CollectionShareRepository struct {
	db *sql.DB
}

// NewCollectionShareRepository creates a new CollectionShareRepository
func NewCollectionShareRepository(db *sql.DB) *CollectionShareRepository {
	return &CollectionShareRepository{db: db}
}

func (r *CollectionShareRepository) GetByCollectionID(ctx context.Context, collectionID string) ([]*models.CollectionShare, error) {
	query := `SELECT id, collection_id, user_id, created_at
			  FROM collection_shares WHERE collection_id = $1 ORDER BY created_at ASC`

	rows, err := r.db.QueryContext(ctx, query, collectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shares []*models.CollectionShare
	for rows.Next() {
		var s models.CollectionShare
		if err := rows.Scan(&s.ID, &s.CollectionID, &s.UserID, &s.CreatedAt); err != nil {
			return nil, err
		}
		shares = append(shares, &s)
	}
	return shares, rows.Err()
}

func (r *CollectionShareRepository) GetSharesWithUsers(ctx context.Context, collectionID string) ([]*models.CollectionShareWithUser, error) {
	query := `SELECT cs.id, cs.collection_id, cs.user_id, cs.created_at,
			  u.id, u.email, u.display_name
			  FROM collection_shares cs
			  INNER JOIN users u ON u.id = cs.user_id
			  WHERE cs.collection_id = $1 ORDER BY cs.created_at ASC`

	rows, err := r.db.QueryContext(ctx, query, collectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shares []*models.CollectionShareWithUser
	for rows.Next() {
		var s models.CollectionShareWithUser
		var user models.User
		if err := rows.Scan(&s.ID, &s.CollectionID, &s.UserID, &s.CreatedAt,
			&user.ID, &user.Email, &user.DisplayName); err != nil {
			return nil, err
		}
		s.User = &user
		shares = append(shares, &s)
	}
	return shares, rows.Err()
}

func (r *CollectionShareRepository) IsSharedWithUser(ctx context.Context, collectionID, userID string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM collection_shares WHERE collection_id = $1 AND user_id = $2)`
	err := r.db.QueryRowContext(ctx, query, collectionID, userID).Scan(&exists)
	return exists, err
}

func (r *CollectionShareRepository) Add(ctx context.Context, share *models.CollectionShare) error {
	query := `INSERT INTO collection_shares (id, collection_id, user_id, created_at)
			  VALUES ($1, $2, $3, $4)
			  ON CONFLICT (collection_id, user_id) DO NOTHING`

	_, err := r.db.ExecContext(ctx, query, share.ID, share.CollectionID, share.UserID, share.CreatedAt)
	return err
}

func (r *CollectionShareRepository) Remove(ctx context.Context, collectionID, userID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM collection_shares WHERE collection_id = $1 AND user_id = $2`, collectionID, userID)
	return err
}

func (r *CollectionShareRepository) RemoveAll(ctx context.Context, collectionID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM collection_shares WHERE collection_id = $1`, collectionID)
	return err
}

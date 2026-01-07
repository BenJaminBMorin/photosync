package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/photosync/server/internal/models"
)

// BootstrapKeyRepository implements BootstrapKeyRepo
type BootstrapKeyRepository struct {
	db *sql.DB
}

// NewBootstrapKeyRepository creates a new bootstrap key repository
func NewBootstrapKeyRepository(db *sql.DB) *BootstrapKeyRepository {
	return &BootstrapKeyRepository{db: db}
}

// Add adds a new bootstrap key
func (r *BootstrapKeyRepository) Add(ctx context.Context, key *models.BootstrapKey) error {
	query := `
		INSERT INTO bootstrap_keys (id, key_hash, created_at, expires_at, used, used_at, used_by)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query,
		key.ID,
		key.KeyHash,
		key.CreatedAt,
		key.ExpiresAt,
		key.Used,
		key.UsedAt,
		key.UsedBy,
	)
	return err
}

// GetByKeyHash retrieves a bootstrap key by its hash
func (r *BootstrapKeyRepository) GetByKeyHash(ctx context.Context, hash string) (*models.BootstrapKey, error) {
	query := `
		SELECT id, key_hash, created_at, expires_at, used, used_at, used_by
		FROM bootstrap_keys
		WHERE key_hash = ?
	`

	key := &models.BootstrapKey{}
	var usedAt sql.NullTime
	var usedBy sql.NullString

	err := r.db.QueryRowContext(ctx, query, hash).Scan(
		&key.ID,
		&key.KeyHash,
		&key.CreatedAt,
		&key.ExpiresAt,
		&key.Used,
		&usedAt,
		&usedBy,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if usedAt.Valid {
		t := usedAt.Time
		key.UsedAt = &t
	}
	if usedBy.Valid {
		key.UsedBy = usedBy.String
	}

	return key, nil
}

// GetActiveKey retrieves the most recent active (unused and not expired) bootstrap key
func (r *BootstrapKeyRepository) GetActiveKey(ctx context.Context) (*models.BootstrapKey, error) {
	query := `
		SELECT id, key_hash, created_at, expires_at, used, used_at, used_by
		FROM bootstrap_keys
		WHERE used = 0 AND expires_at > ?
		ORDER BY created_at DESC
		LIMIT 1
	`

	key := &models.BootstrapKey{}
	var usedAt sql.NullTime
	var usedBy sql.NullString

	err := r.db.QueryRowContext(ctx, query, time.Now().UTC()).Scan(
		&key.ID,
		&key.KeyHash,
		&key.CreatedAt,
		&key.ExpiresAt,
		&key.Used,
		&usedAt,
		&usedBy,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if usedAt.Valid {
		t := usedAt.Time
		key.UsedAt = &t
	}
	if usedBy.Valid {
		key.UsedBy = usedBy.String
	}

	return key, nil
}

// MarkUsed marks a bootstrap key as used
func (r *BootstrapKeyRepository) MarkUsed(ctx context.Context, id, usedBy string) error {
	query := `
		UPDATE bootstrap_keys
		SET used = 1, used_at = ?, used_by = ?
		WHERE id = ?
	`
	_, err := r.db.ExecContext(ctx, query, time.Now().UTC(), usedBy, id)
	return err
}

// ExpireOld deletes bootstrap keys that have expired
func (r *BootstrapKeyRepository) ExpireOld(ctx context.Context) (int, error) {
	query := `DELETE FROM bootstrap_keys WHERE expires_at < ?`
	result, err := r.db.ExecContext(ctx, query, time.Now().UTC())
	if err != nil {
		return 0, err
	}

	rows, err := result.RowsAffected()
	return int(rows), err
}

// HasActiveKey checks if there is an active bootstrap key
func (r *BootstrapKeyRepository) HasActiveKey(ctx context.Context) (bool, error) {
	query := `
		SELECT COUNT(*) FROM bootstrap_keys
		WHERE used = 0 AND expires_at > ?
	`

	var count int
	err := r.db.QueryRowContext(ctx, query, time.Now().UTC()).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

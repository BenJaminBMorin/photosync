package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/photosync/server/internal/models"
)

// InviteTokenRepository implements invite token data access
type InviteTokenRepository struct {
	db *sql.DB
}

// NewInviteTokenRepository creates a new invite token repository
func NewInviteTokenRepository(db *sql.DB) *InviteTokenRepository {
	return &InviteTokenRepository{db: db}
}

// Add adds a new invite token
func (r *InviteTokenRepository) Add(ctx context.Context, token *models.InviteToken) error {
	query := `
		INSERT INTO invite_tokens (id, token, token_hash, server_url, user_id, email, created_by, created_at, expires_at, used, used_at, used_from_ip, used_from_device)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query,
		token.ID,
		token.Token,
		token.TokenHash,
		token.ServerURL,
		token.UserID,
		token.Email,
		token.CreatedBy,
		token.CreatedAt,
		token.ExpiresAt,
		token.Used,
		token.UsedAt,
		token.UsedFromIP,
		token.UsedFromDevice,
	)
	return err
}

// GetByTokenHash retrieves an invite token by its hash
func (r *InviteTokenRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*models.InviteToken, error) {
	query := `
		SELECT id, token, token_hash, server_url, user_id, email, created_by, created_at, expires_at, used, used_at, used_from_ip, used_from_device
		FROM invite_tokens
		WHERE token_hash = ?
	`

	it := &models.InviteToken{}
	var usedAt sql.NullTime
	var usedFromIP sql.NullString
	var usedFromDevice sql.NullString

	err := r.db.QueryRowContext(ctx, query, tokenHash).Scan(
		&it.ID,
		&it.Token,
		&it.TokenHash,
		&it.ServerURL,
		&it.UserID,
		&it.Email,
		&it.CreatedBy,
		&it.CreatedAt,
		&it.ExpiresAt,
		&it.Used,
		&usedAt,
		&usedFromIP,
		&usedFromDevice,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if usedAt.Valid {
		t := usedAt.Time
		it.UsedAt = &t
	}
	if usedFromIP.Valid {
		it.UsedFromIP = usedFromIP.String
	}
	if usedFromDevice.Valid {
		it.UsedFromDevice = usedFromDevice.String
	}

	return it, nil
}

// GetByUserID retrieves all invite tokens for a user
func (r *InviteTokenRepository) GetByUserID(ctx context.Context, userID string) ([]*models.InviteToken, error) {
	query := `
		SELECT id, token, token_hash, server_url, user_id, email, created_by, created_at, expires_at, used, used_at, used_from_ip, used_from_device
		FROM invite_tokens
		WHERE user_id = ?
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []*models.InviteToken
	for rows.Next() {
		it := &models.InviteToken{}
		var usedAt sql.NullTime
		var usedFromIP sql.NullString
		var usedFromDevice sql.NullString

		err := rows.Scan(
			&it.ID,
			&it.Token,
			&it.TokenHash,
			&it.ServerURL,
			&it.UserID,
			&it.Email,
			&it.CreatedBy,
			&it.CreatedAt,
			&it.ExpiresAt,
			&it.Used,
			&usedAt,
			&usedFromIP,
			&usedFromDevice,
		)
		if err != nil {
			return nil, err
		}

		if usedAt.Valid {
			t := usedAt.Time
			it.UsedAt = &t
		}
		if usedFromIP.Valid {
			it.UsedFromIP = usedFromIP.String
		}
		if usedFromDevice.Valid {
			it.UsedFromDevice = usedFromDevice.String
		}

		tokens = append(tokens, it)
	}

	return tokens, rows.Err()
}

// MarkUsed marks an invite token as used with tracking information
func (r *InviteTokenRepository) MarkUsed(ctx context.Context, id, ipAddress, deviceInfo string) error {
	query := `
		UPDATE invite_tokens
		SET used = 1, used_at = ?, used_from_ip = ?, used_from_device = ?
		WHERE id = ?
	`
	_, err := r.db.ExecContext(ctx, query, time.Now().UTC(), ipAddress, deviceInfo, id)
	return err
}

// HasPendingInvite checks if a user has any unused, non-expired invite tokens
func (r *InviteTokenRepository) HasPendingInvite(ctx context.Context, userID string) (bool, error) {
	query := `
		SELECT COUNT(*) FROM invite_tokens
		WHERE user_id = ? AND used = 0 AND expires_at > ?
	`

	var count int
	err := r.db.QueryRowContext(ctx, query, userID, time.Now().UTC()).Scan(&count)
	return count > 0, err
}

// ExpireOld deletes invite tokens that have expired
func (r *InviteTokenRepository) ExpireOld(ctx context.Context) (int, error) {
	query := `DELETE FROM invite_tokens WHERE expires_at < ?`
	result, err := r.db.ExecContext(ctx, query, time.Now().UTC())
	if err != nil {
		return 0, err
	}

	rows, err := result.RowsAffected()
	return int(rows), err
}

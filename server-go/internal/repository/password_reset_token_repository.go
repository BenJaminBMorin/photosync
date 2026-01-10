package repository

import (
	"context"
	"database/sql"
	"github.com/photosync/server/internal/models"
)

// PasswordResetTokenRepo defines the interface for password reset token persistence
type PasswordResetTokenRepo interface {
	Add(ctx context.Context, token *models.PasswordResetToken) error
	GetActiveByUserID(ctx context.Context, userID string) ([]*models.PasswordResetToken, error)
	Update(ctx context.Context, token *models.PasswordResetToken) error
	RevokeAllForUser(ctx context.Context, userID string) error
}

// PasswordResetTokenRepository implements PasswordResetTokenRepo for SQLite
type PasswordResetTokenRepository struct {
	db *sql.DB
}

// NewPasswordResetTokenRepository creates a new repository
func NewPasswordResetTokenRepository(db *sql.DB) *PasswordResetTokenRepository {
	return &PasswordResetTokenRepository{db: db}
}

// Add inserts a new password reset token
func (r *PasswordResetTokenRepository) Add(ctx context.Context, token *models.PasswordResetToken) error {
	query := `INSERT INTO password_reset_tokens
	          (id, user_id, code_hash, email, created_at, expires_at, used, ip_address, attempts)
	          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.ExecContext(ctx, query,
		token.ID, token.UserID, token.CodeHash, token.Email,
		token.CreatedAt, token.ExpiresAt, token.Used, token.IPAddress, token.Attempts,
	)
	return err
}

// GetActiveByUserID retrieves active (unused and not expired) tokens for a user
func (r *PasswordResetTokenRepository) GetActiveByUserID(ctx context.Context, userID string) ([]*models.PasswordResetToken, error) {
	query := `SELECT id, user_id, code_hash, email, created_at, expires_at,
	          used, used_at, ip_address, attempts, last_attempt_at
	          FROM password_reset_tokens
	          WHERE user_id = ? AND used = 0 AND expires_at > datetime('now')
	          ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []*models.PasswordResetToken
	for rows.Next() {
		token := &models.PasswordResetToken{}
		var usedAt sql.NullTime
		var lastAttemptAt sql.NullTime

		if err := rows.Scan(
			&token.ID, &token.UserID, &token.CodeHash, &token.Email,
			&token.CreatedAt, &token.ExpiresAt, &token.Used, &usedAt,
			&token.IPAddress, &token.Attempts, &lastAttemptAt,
		); err != nil {
			return nil, err
		}

		if usedAt.Valid {
			token.UsedAt = &usedAt.Time
		}
		if lastAttemptAt.Valid {
			token.LastAttemptAt = &lastAttemptAt.Time
		}

		tokens = append(tokens, token)
	}

	return tokens, rows.Err()
}

// Update updates an existing password reset token
func (r *PasswordResetTokenRepository) Update(ctx context.Context, token *models.PasswordResetToken) error {
	query := `UPDATE password_reset_tokens
	          SET used = ?, used_at = ?, attempts = ?, last_attempt_at = ?
	          WHERE id = ?`

	_, err := r.db.ExecContext(ctx, query,
		token.Used, token.UsedAt, token.Attempts, token.LastAttemptAt, token.ID,
	)
	return err
}

// RevokeAllForUser marks all tokens for a user as used
func (r *PasswordResetTokenRepository) RevokeAllForUser(ctx context.Context, userID string) error {
	query := `UPDATE password_reset_tokens SET used = 1 WHERE user_id = ? AND used = 0`
	_, err := r.db.ExecContext(ctx, query, userID)
	return err
}

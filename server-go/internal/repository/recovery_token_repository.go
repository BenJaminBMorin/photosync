package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/photosync/server/internal/models"
)

// RecoveryTokenRepository implements RecoveryTokenRepo
type RecoveryTokenRepository struct {
	db *sql.DB
}

// NewRecoveryTokenRepository creates a new recovery token repository
func NewRecoveryTokenRepository(db *sql.DB) *RecoveryTokenRepository {
	return &RecoveryTokenRepository{db: db}
}

// Add adds a new recovery token
func (r *RecoveryTokenRepository) Add(ctx context.Context, token *models.RecoveryToken) error {
	query := `
		INSERT INTO recovery_tokens (id, token_hash, user_id, email, created_at, expires_at, used, used_at, ip_address)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query,
		token.ID,
		token.TokenHash,
		token.UserID,
		token.Email,
		token.CreatedAt,
		token.ExpiresAt,
		token.Used,
		token.UsedAt,
		token.IPAddress,
	)
	return err
}

// GetByTokenHash retrieves a recovery token by its hash
func (r *RecoveryTokenRepository) GetByTokenHash(ctx context.Context, hash string) (*models.RecoveryToken, error) {
	query := `
		SELECT id, token_hash, user_id, email, created_at, expires_at, used, used_at, ip_address
		FROM recovery_tokens
		WHERE token_hash = ?
	`

	token := &models.RecoveryToken{}
	var usedAt sql.NullTime
	var ipAddress sql.NullString

	err := r.db.QueryRowContext(ctx, query, hash).Scan(
		&token.ID,
		&token.TokenHash,
		&token.UserID,
		&token.Email,
		&token.CreatedAt,
		&token.ExpiresAt,
		&token.Used,
		&usedAt,
		&ipAddress,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if usedAt.Valid {
		t := usedAt.Time
		token.UsedAt = &t
	}
	if ipAddress.Valid {
		token.IPAddress = ipAddress.String
	}

	return token, nil
}

// MarkUsed marks a recovery token as used
func (r *RecoveryTokenRepository) MarkUsed(ctx context.Context, id, ipAddress string) error {
	query := `
		UPDATE recovery_tokens
		SET used = 1, used_at = ?, ip_address = ?
		WHERE id = ?
	`
	_, err := r.db.ExecContext(ctx, query, time.Now().UTC(), ipAddress, id)
	return err
}

// GetRecentCountForEmail counts recovery tokens created for an email since a given time
func (r *RecoveryTokenRepository) GetRecentCountForEmail(ctx context.Context, email string, since time.Time) (int, error) {
	query := `
		SELECT COUNT(*) FROM recovery_tokens
		WHERE email = ? AND created_at > ?
	`

	var count int
	err := r.db.QueryRowContext(ctx, query, email, since).Scan(&count)
	return count, err
}

// RecordRateLimit records a rate limit event for an email
func (r *RecoveryTokenRepository) RecordRateLimit(ctx context.Context, email string) error {
	query := `
		INSERT INTO recovery_rate_limits (email, last_request_at, request_count)
		VALUES (?, ?, 1)
		ON CONFLICT(email) DO UPDATE SET
			last_request_at = ?,
			request_count = request_count + 1
	`
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, query, email, now, now)
	return err
}

// CheckRateLimit checks if an email is rate limited (more than 1 request in 5 minutes)
func (r *RecoveryTokenRepository) CheckRateLimit(ctx context.Context, email string) (bool, error) {
	query := `
		SELECT last_request_at, request_count
		FROM recovery_rate_limits
		WHERE email = ?
	`

	var lastRequestAt time.Time
	var requestCount int

	err := r.db.QueryRowContext(ctx, query, email).Scan(&lastRequestAt, &requestCount)
	if err == sql.ErrNoRows {
		// No rate limit record exists, so not rate limited
		return false, nil
	}
	if err != nil {
		return false, err
	}

	// Check if last request was within 5 minutes
	fiveMinutesAgo := time.Now().UTC().Add(-5 * time.Minute)
	if lastRequestAt.After(fiveMinutesAgo) {
		// Still within rate limit window
		return true, nil
	}

	// Rate limit window has passed, reset count
	resetQuery := `
		UPDATE recovery_rate_limits
		SET request_count = 0
		WHERE email = ?
	`
	_, err = r.db.ExecContext(ctx, resetQuery, email)
	return false, err
}

// ExpireOld deletes recovery tokens that have expired
func (r *RecoveryTokenRepository) ExpireOld(ctx context.Context) (int, error) {
	query := `DELETE FROM recovery_tokens WHERE expires_at < ?`
	result, err := r.db.ExecContext(ctx, query, time.Now().UTC())
	if err != nil {
		return 0, err
	}

	rows, err := result.RowsAffected()
	return int(rows), err
}

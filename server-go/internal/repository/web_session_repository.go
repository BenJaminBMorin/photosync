package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/photosync/server/internal/models"
)

// WebSessionRepository implements WebSessionRepo for PostgreSQL/SQLite
type WebSessionRepository struct {
	db *sql.DB
}

// NewWebSessionRepository creates a new WebSessionRepository
func NewWebSessionRepository(db *sql.DB) *WebSessionRepository {
	return &WebSessionRepository{db: db}
}

func (r *WebSessionRepository) GetByID(ctx context.Context, id string) (*models.WebSession, error) {
	query := `SELECT id, user_id, auth_request_id, created_at, expires_at, last_activity_at, ip_address, user_agent, is_active
			  FROM web_sessions WHERE id = $1`

	var session models.WebSession
	var authRequestID sql.NullString
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&session.ID, &session.UserID, &authRequestID, &session.CreatedAt,
		&session.ExpiresAt, &session.LastActivityAt, &session.IPAddress,
		&session.UserAgent, &session.IsActive,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if authRequestID.Valid {
		session.AuthRequestID = &authRequestID.String
	}
	return &session, nil
}

func (r *WebSessionRepository) GetActiveForUser(ctx context.Context, userID string) ([]*models.WebSession, error) {
	query := `SELECT id, user_id, auth_request_id, created_at, expires_at, last_activity_at, ip_address, user_agent, is_active
			  FROM web_sessions WHERE user_id = $1 AND is_active = true AND expires_at > $2
			  ORDER BY last_activity_at DESC`

	rows, err := r.db.QueryContext(ctx, query, userID, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*models.WebSession
	for rows.Next() {
		var session models.WebSession
		var authRequestID sql.NullString
		if err := rows.Scan(&session.ID, &session.UserID, &authRequestID, &session.CreatedAt,
			&session.ExpiresAt, &session.LastActivityAt, &session.IPAddress,
			&session.UserAgent, &session.IsActive); err != nil {
			return nil, err
		}
		if authRequestID.Valid {
			session.AuthRequestID = &authRequestID.String
		}
		sessions = append(sessions, &session)
	}
	return sessions, rows.Err()
}

func (r *WebSessionRepository) Add(ctx context.Context, session *models.WebSession) error {
	query := `INSERT INTO web_sessions (id, user_id, auth_request_id, created_at, expires_at, last_activity_at, ip_address, user_agent, is_active)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	var authRequestID interface{}
	if session.AuthRequestID != nil {
		authRequestID = *session.AuthRequestID
	}

	_, err := r.db.ExecContext(ctx, query,
		session.ID, session.UserID, authRequestID, session.CreatedAt,
		session.ExpiresAt, session.LastActivityAt, session.IPAddress,
		session.UserAgent, session.IsActive,
	)
	return err
}

func (r *WebSessionRepository) Touch(ctx context.Context, id string) error {
	query := `UPDATE web_sessions SET last_activity_at = $2 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, time.Now().UTC())
	return err
}

func (r *WebSessionRepository) Invalidate(ctx context.Context, id string) error {
	query := `UPDATE web_sessions SET is_active = false WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *WebSessionRepository) InvalidateAllForUser(ctx context.Context, userID string) error {
	query := `UPDATE web_sessions SET is_active = false WHERE user_id = $1`
	_, err := r.db.ExecContext(ctx, query, userID)
	return err
}

func (r *WebSessionRepository) CleanupExpired(ctx context.Context) (int, error) {
	query := `DELETE FROM web_sessions WHERE expires_at <= $1 OR is_active = false`

	result, err := r.db.ExecContext(ctx, query, time.Now().UTC())
	if err != nil {
		return 0, err
	}
	rows, err := result.RowsAffected()
	return int(rows), err
}

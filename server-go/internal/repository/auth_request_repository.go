package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/photosync/server/internal/models"
)

// AuthRequestRepository implements AuthRequestRepo for PostgreSQL/SQLite
type AuthRequestRepository struct {
	db *sql.DB
}

// NewAuthRequestRepository creates a new AuthRequestRepository
func NewAuthRequestRepository(db *sql.DB) *AuthRequestRepository {
	return &AuthRequestRepository{db: db}
}

func (r *AuthRequestRepository) GetByID(ctx context.Context, id string) (*models.AuthRequest, error) {
	query := `SELECT id, user_id, status, request_type, new_password_hash, created_at, expires_at, responded_at, device_id, ip_address, user_agent
			  FROM auth_requests WHERE id = $1`

	var req models.AuthRequest
	var respondedAt sql.NullTime
	var deviceID sql.NullString
	var requestType sql.NullString
	var newPasswordHash sql.NullString
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&req.ID, &req.UserID, &req.Status, &requestType, &newPasswordHash, &req.CreatedAt, &req.ExpiresAt,
		&respondedAt, &deviceID, &req.IPAddress, &req.UserAgent,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if requestType.Valid {
		req.RequestType = requestType.String
	} else {
		req.RequestType = "web_login"
	}
	if newPasswordHash.Valid {
		req.NewPasswordHash = newPasswordHash.String
	}
	if respondedAt.Valid {
		req.RespondedAt = &respondedAt.Time
	}
	if deviceID.Valid {
		req.DeviceID = &deviceID.String
	}
	return &req, nil
}

func (r *AuthRequestRepository) GetPendingForUser(ctx context.Context, userID string) ([]*models.AuthRequest, error) {
	query := `SELECT id, user_id, status, created_at, expires_at, responded_at, device_id, ip_address, user_agent
			  FROM auth_requests WHERE user_id = $1 AND status = 'pending' AND expires_at > $2
			  ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, userID, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []*models.AuthRequest
	for rows.Next() {
		var req models.AuthRequest
		var respondedAt sql.NullTime
		var deviceID sql.NullString
		if err := rows.Scan(&req.ID, &req.UserID, &req.Status, &req.CreatedAt, &req.ExpiresAt,
			&respondedAt, &deviceID, &req.IPAddress, &req.UserAgent); err != nil {
			return nil, err
		}
		if respondedAt.Valid {
			req.RespondedAt = &respondedAt.Time
		}
		if deviceID.Valid {
			req.DeviceID = &deviceID.String
		}
		requests = append(requests, &req)
	}
	return requests, rows.Err()
}

func (r *AuthRequestRepository) Add(ctx context.Context, req *models.AuthRequest) error {
	query := `INSERT INTO auth_requests (id, user_id, status, request_type, new_password_hash, created_at, expires_at, ip_address, user_agent)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	requestType := req.RequestType
	if requestType == "" {
		requestType = "web_login"
	}

	_, err := r.db.ExecContext(ctx, query,
		req.ID, req.UserID, req.Status, requestType, req.NewPasswordHash,
		req.CreatedAt, req.ExpiresAt, req.IPAddress, req.UserAgent,
	)
	return err
}

func (r *AuthRequestRepository) Update(ctx context.Context, req *models.AuthRequest) error {
	query := `UPDATE auth_requests SET status = $2, responded_at = $3, device_id = $4 WHERE id = $1`

	var respondedAt interface{}
	if req.RespondedAt != nil {
		respondedAt = *req.RespondedAt
	}
	var deviceID interface{}
	if req.DeviceID != nil {
		deviceID = *req.DeviceID
	}

	_, err := r.db.ExecContext(ctx, query, req.ID, req.Status, respondedAt, deviceID)
	return err
}

func (r *AuthRequestRepository) ExpireOld(ctx context.Context) (int, error) {
	query := `UPDATE auth_requests SET status = 'expired' WHERE status = 'pending' AND expires_at <= $1`

	result, err := r.db.ExecContext(ctx, query, time.Now().UTC())
	if err != nil {
		return 0, err
	}
	rows, err := result.RowsAffected()
	return int(rows), err
}

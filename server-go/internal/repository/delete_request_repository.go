package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/photosync/server/internal/models"
)

// DeleteRequestRepository implements delete request storage for PostgreSQL/SQLite
type DeleteRequestRepository struct {
	db *sql.DB
}

// NewDeleteRequestRepository creates a new DeleteRequestRepository
func NewDeleteRequestRepository(db *sql.DB) *DeleteRequestRepository {
	return &DeleteRequestRepository{db: db}
}

func (r *DeleteRequestRepository) GetByID(ctx context.Context, id string) (*models.DeleteRequest, error) {
	query := `SELECT id, user_id, photo_ids, status, created_at, expires_at, responded_at, device_id, ip_address, user_agent
			  FROM delete_requests WHERE id = $1`

	var req models.DeleteRequest
	var respondedAt sql.NullTime
	var deviceID sql.NullString
	var photoIDsJSON string
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&req.ID, &req.UserID, &photoIDsJSON, &req.Status, &req.CreatedAt, &req.ExpiresAt,
		&respondedAt, &deviceID, &req.IPAddress, &req.UserAgent,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Parse photo IDs from JSON
	if err := json.Unmarshal([]byte(photoIDsJSON), &req.PhotoIDs); err != nil {
		return nil, err
	}

	if respondedAt.Valid {
		req.RespondedAt = &respondedAt.Time
	}
	if deviceID.Valid {
		req.DeviceID = &deviceID.String
	}
	return &req, nil
}

func (r *DeleteRequestRepository) GetPendingForUser(ctx context.Context, userID string) ([]*models.DeleteRequest, error) {
	query := `SELECT id, user_id, photo_ids, status, created_at, expires_at, responded_at, device_id, ip_address, user_agent
			  FROM delete_requests WHERE user_id = $1 AND status = 'pending' AND expires_at > $2
			  ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, userID, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []*models.DeleteRequest
	for rows.Next() {
		var req models.DeleteRequest
		var respondedAt sql.NullTime
		var deviceID sql.NullString
		var photoIDsJSON string
		if err := rows.Scan(&req.ID, &req.UserID, &photoIDsJSON, &req.Status, &req.CreatedAt, &req.ExpiresAt,
			&respondedAt, &deviceID, &req.IPAddress, &req.UserAgent); err != nil {
			return nil, err
		}

		// Parse photo IDs from JSON
		if err := json.Unmarshal([]byte(photoIDsJSON), &req.PhotoIDs); err != nil {
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

func (r *DeleteRequestRepository) Add(ctx context.Context, req *models.DeleteRequest) error {
	query := `INSERT INTO delete_requests (id, user_id, photo_ids, status, created_at, expires_at, ip_address, user_agent)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	// Convert photo IDs to JSON
	photoIDsJSON, err := json.Marshal(req.PhotoIDs)
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(ctx, query,
		req.ID, req.UserID, string(photoIDsJSON), req.Status, req.CreatedAt, req.ExpiresAt,
		req.IPAddress, req.UserAgent,
	)
	return err
}

func (r *DeleteRequestRepository) Update(ctx context.Context, req *models.DeleteRequest) error {
	query := `UPDATE delete_requests SET status = $2, responded_at = $3, device_id = $4 WHERE id = $1`

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

func (r *DeleteRequestRepository) ExpireOld(ctx context.Context) (int, error) {
	query := `UPDATE delete_requests SET status = 'expired' WHERE status = 'pending' AND expires_at <= $1`

	result, err := r.db.ExecContext(ctx, query, time.Now().UTC())
	if err != nil {
		return 0, err
	}
	rows, err := result.RowsAffected()
	return int(rows), err
}

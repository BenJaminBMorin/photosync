package repository

import (
	"context"
	"database/sql"

	"github.com/photosync/server/internal/models"
)

// UserRepository implements UserRepo for PostgreSQL/SQLite
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository creates a new UserRepository
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	query := `SELECT id, email, display_name, api_key, api_key_hash, password_hash, is_admin, created_at, is_active
			  FROM users WHERE id = $1`

	var user models.User
	var passwordHash sql.NullString
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.DisplayName, &user.APIKey, &user.APIKeyHash,
		&passwordHash, &user.IsAdmin, &user.CreatedAt, &user.IsActive,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if passwordHash.Valid {
		user.PasswordHash = passwordHash.String
	}
	user.APIKey = "" // Never return API key after creation
	return &user, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `SELECT id, email, display_name, api_key, api_key_hash, password_hash, is_admin, created_at, is_active
			  FROM users WHERE email = $1`

	var user models.User
	var passwordHash sql.NullString
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.DisplayName, &user.APIKey, &user.APIKeyHash,
		&passwordHash, &user.IsAdmin, &user.CreatedAt, &user.IsActive,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if passwordHash.Valid {
		user.PasswordHash = passwordHash.String
	}
	user.APIKey = "" // Never return API key after creation
	return &user, nil
}

func (r *UserRepository) GetByAPIKeyHash(ctx context.Context, apiKeyHash string) (*models.User, error) {
	query := `SELECT id, email, display_name, api_key, api_key_hash, password_hash, is_admin, created_at, is_active
			  FROM users WHERE api_key_hash = $1 AND is_active = true`

	var user models.User
	var passwordHash sql.NullString
	err := r.db.QueryRowContext(ctx, query, apiKeyHash).Scan(
		&user.ID, &user.Email, &user.DisplayName, &user.APIKey, &user.APIKeyHash,
		&passwordHash, &user.IsAdmin, &user.CreatedAt, &user.IsActive,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if passwordHash.Valid {
		user.PasswordHash = passwordHash.String
	}
	user.APIKey = "" // Never return API key after creation
	return &user, nil
}

func (r *UserRepository) GetAll(ctx context.Context) ([]*models.User, error) {
	query := `SELECT id, email, display_name, api_key_hash, is_admin, created_at, is_active
			  FROM users ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		var user models.User
		if err := rows.Scan(&user.ID, &user.Email, &user.DisplayName, &user.APIKeyHash,
			&user.IsAdmin, &user.CreatedAt, &user.IsActive); err != nil {
			return nil, err
		}
		users = append(users, &user)
	}
	return users, rows.Err()
}

func (r *UserRepository) GetCount(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&count)
	return count, err
}

func (r *UserRepository) Add(ctx context.Context, user *models.User) error {
	query := `INSERT INTO users (id, email, display_name, api_key, api_key_hash, is_admin, created_at, is_active)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := r.db.ExecContext(ctx, query,
		user.ID, user.Email, user.DisplayName, user.APIKey, user.APIKeyHash,
		user.IsAdmin, user.CreatedAt, user.IsActive,
	)
	return err
}

func (r *UserRepository) Update(ctx context.Context, user *models.User) error {
	query := `UPDATE users SET email = $2, display_name = $3, is_admin = $4, is_active = $5
			  WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, user.ID, user.Email, user.DisplayName, user.IsAdmin, user.IsActive)
	return err
}

func (r *UserRepository) Delete(ctx context.Context, id string) (bool, error) {
	result, err := r.db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	return rows > 0, err
}

// UpdateAPIKeyHash updates a user's API key hash (used for API key reset)
func (r *UserRepository) UpdateAPIKeyHash(ctx context.Context, id, apiKeyHash string) error {
	query := `UPDATE users SET api_key_hash = $2, api_key = '' WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, apiKeyHash)
	return err
}

// UpdatePasswordHash updates a user's password hash
func (r *UserRepository) UpdatePasswordHash(ctx context.Context, userID, passwordHash string) error {
	query := `UPDATE users SET password_hash = $2 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, userID, passwordHash)
	return err
}

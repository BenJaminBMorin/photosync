package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/photosync/server/internal/models"
)

// SMTPConfigRepository implements SMTPConfigRepo
type SMTPConfigRepository struct {
	db *sql.DB
}

// NewSMTPConfigRepository creates a new SMTP config repository
func NewSMTPConfigRepository(db *sql.DB) *SMTPConfigRepository {
	return &SMTPConfigRepository{db: db}
}

// Get retrieves the SMTP configuration
func (r *SMTPConfigRepository) Get(ctx context.Context) (*models.SMTPConfig, error) {
	query := `
		SELECT host, port, username, password_encrypted, from_address, from_name, use_tls, skip_verify
		FROM smtp_config
		WHERE id = 1
	`

	config := &models.SMTPConfig{}
	err := r.db.QueryRowContext(ctx, query).Scan(
		&config.Host,
		&config.Port,
		&config.Username,
		&config.Password, // This will be the encrypted password
		&config.FromAddress,
		&config.FromName,
		&config.UseTLS,
		&config.SkipVerify,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return config, nil
}

// Set creates or updates the SMTP configuration
func (r *SMTPConfigRepository) Set(ctx context.Context, config *models.SMTPConfig, updatedBy string) error {
	query := `
		INSERT INTO smtp_config (id, host, port, username, password_encrypted, from_address, from_name, use_tls, skip_verify, updated_at, updated_by)
		VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			host = ?,
			port = ?,
			username = ?,
			password_encrypted = ?,
			from_address = ?,
			from_name = ?,
			use_tls = ?,
			skip_verify = ?,
			updated_at = ?,
			updated_by = ?
	`

	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, query,
		// INSERT values
		config.Host,
		config.Port,
		config.Username,
		config.Password, // This should be encrypted before calling Set
		config.FromAddress,
		config.FromName,
		config.UseTLS,
		config.SkipVerify,
		now,
		updatedBy,
		// UPDATE values
		config.Host,
		config.Port,
		config.Username,
		config.Password,
		config.FromAddress,
		config.FromName,
		config.UseTLS,
		config.SkipVerify,
		now,
		updatedBy,
	)
	return err
}

// IsConfigured checks if SMTP is configured
func (r *SMTPConfigRepository) IsConfigured(ctx context.Context) (bool, error) {
	query := `SELECT COUNT(*) FROM smtp_config WHERE id = 1`

	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

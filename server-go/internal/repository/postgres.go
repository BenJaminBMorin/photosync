package repository

import (
	"database/sql"

	_ "github.com/lib/pq"
)

// NewPostgresDB creates and initializes a PostgreSQL database connection
func NewPostgresDB(connStr string) (*sql.DB, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	// Create tables
	if err := createPostgresTables(db); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func createPostgresTables(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS photos (
		id TEXT PRIMARY KEY,
		original_filename TEXT NOT NULL,
		stored_path TEXT NOT NULL,
		file_hash TEXT NOT NULL,
		file_size BIGINT NOT NULL,
		date_taken TIMESTAMP NOT NULL,
		uploaded_at TIMESTAMP NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_photos_hash ON photos(file_hash);
	CREATE INDEX IF NOT EXISTS idx_photos_date ON photos(date_taken);
	`

	_, err := db.Exec(schema)
	return err
}

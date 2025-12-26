package repository

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

// NewSQLiteDB creates and initializes a SQLite database
func NewSQLiteDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, err
	}

	// Create tables
	if err := createTables(db); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func createTables(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS photos (
		id TEXT PRIMARY KEY,
		original_filename TEXT NOT NULL,
		stored_path TEXT NOT NULL,
		file_hash TEXT NOT NULL,
		file_size INTEGER NOT NULL,
		date_taken DATETIME NOT NULL,
		uploaded_at DATETIME NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_photos_hash ON photos(file_hash);
	CREATE INDEX IF NOT EXISTS idx_photos_date ON photos(date_taken);
	`

	_, err := db.Exec(schema)
	return err
}

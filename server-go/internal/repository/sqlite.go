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
	-- Users table
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		email TEXT UNIQUE NOT NULL,
		display_name TEXT NOT NULL,
		api_key TEXT UNIQUE NOT NULL,
		api_key_hash TEXT NOT NULL,
		is_admin INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		is_active INTEGER NOT NULL DEFAULT 1
	);

	CREATE INDEX IF NOT EXISTS idx_users_api_key_hash ON users(api_key_hash);
	CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

	-- Devices table (for push notifications)
	CREATE TABLE IF NOT EXISTS devices (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		device_name TEXT NOT NULL,
		platform TEXT NOT NULL,
		fcm_token TEXT NOT NULL,
		registered_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		last_seen_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		is_active INTEGER NOT NULL DEFAULT 1
	);

	CREATE INDEX IF NOT EXISTS idx_devices_user_id ON devices(user_id);
	CREATE INDEX IF NOT EXISTS idx_devices_fcm_token ON devices(fcm_token);

	-- Auth requests (pending push approvals)
	CREATE TABLE IF NOT EXISTS auth_requests (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		status TEXT NOT NULL DEFAULT 'pending',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		expires_at DATETIME NOT NULL,
		responded_at DATETIME,
		device_id TEXT REFERENCES devices(id),
		ip_address TEXT,
		user_agent TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_auth_requests_user_id ON auth_requests(user_id);
	CREATE INDEX IF NOT EXISTS idx_auth_requests_status ON auth_requests(status);

	-- Delete requests (pending photo deletion approvals)
	CREATE TABLE IF NOT EXISTS delete_requests (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		photo_ids TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		expires_at DATETIME NOT NULL,
		responded_at DATETIME,
		device_id TEXT REFERENCES devices(id),
		ip_address TEXT,
		user_agent TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_delete_requests_user_id ON delete_requests(user_id);
	CREATE INDEX IF NOT EXISTS idx_delete_requests_status ON delete_requests(status);

	-- Web sessions
	CREATE TABLE IF NOT EXISTS web_sessions (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		auth_request_id TEXT REFERENCES auth_requests(id),
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		expires_at DATETIME NOT NULL,
		last_activity_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		ip_address TEXT,
		user_agent TEXT,
		is_active INTEGER NOT NULL DEFAULT 1
	);

	CREATE INDEX IF NOT EXISTS idx_web_sessions_user_id ON web_sessions(user_id);

	-- Photos table (with user_id)
	CREATE TABLE IF NOT EXISTS photos (
		id TEXT PRIMARY KEY,
		user_id TEXT REFERENCES users(id),
		original_filename TEXT NOT NULL,
		stored_path TEXT NOT NULL,
		file_hash TEXT NOT NULL,
		file_size INTEGER NOT NULL,
		date_taken DATETIME NOT NULL,
		uploaded_at DATETIME NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_photos_hash ON photos(file_hash);
	CREATE INDEX IF NOT EXISTS idx_photos_date ON photos(date_taken);
	CREATE INDEX IF NOT EXISTS idx_photos_user_id ON photos(user_id);

	-- Setup config table
	CREATE TABLE IF NOT EXISTS setup_config (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	`

	_, err := db.Exec(schema)
	return err
}

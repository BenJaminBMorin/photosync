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

	-- Bootstrap keys (emergency admin access)
	CREATE TABLE IF NOT EXISTS bootstrap_keys (
		id TEXT PRIMARY KEY,
		key_hash TEXT NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		expires_at DATETIME NOT NULL,
		used INTEGER NOT NULL DEFAULT 0,
		used_at DATETIME,
		used_by TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_bootstrap_expires ON bootstrap_keys(expires_at);

	-- Recovery tokens (email-based account recovery)
	CREATE TABLE IF NOT EXISTS recovery_tokens (
		id TEXT PRIMARY KEY,
		token_hash TEXT NOT NULL,
		user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		email TEXT NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		expires_at DATETIME NOT NULL,
		used INTEGER NOT NULL DEFAULT 0,
		used_at DATETIME,
		ip_address TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_recovery_expires ON recovery_tokens(expires_at);
	CREATE INDEX IF NOT EXISTS idx_recovery_user ON recovery_tokens(user_id);

	-- Config overrides (runtime-editable configuration)
	CREATE TABLE IF NOT EXISTS config_overrides (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		value_type TEXT NOT NULL,
		category TEXT NOT NULL,
		requires_restart INTEGER NOT NULL DEFAULT 0,
		is_sensitive INTEGER NOT NULL DEFAULT 0,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_by TEXT NOT NULL REFERENCES users(id)
	);

	CREATE INDEX IF NOT EXISTS idx_config_category ON config_overrides(category);

	-- SMTP configuration
	CREATE TABLE IF NOT EXISTS smtp_config (
		id INTEGER PRIMARY KEY DEFAULT 1,
		host TEXT NOT NULL,
		port INTEGER NOT NULL DEFAULT 587,
		username TEXT NOT NULL,
		password_encrypted TEXT NOT NULL,
		from_address TEXT NOT NULL,
		from_name TEXT NOT NULL DEFAULT 'PhotoSync',
		use_tls INTEGER NOT NULL DEFAULT 1,
		skip_verify INTEGER NOT NULL DEFAULT 0,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_by TEXT NOT NULL REFERENCES users(id),
		CHECK (id = 1)
	);

	-- Recovery rate limits
	CREATE TABLE IF NOT EXISTS recovery_rate_limits (
		email TEXT PRIMARY KEY,
		last_request_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		request_count INTEGER NOT NULL DEFAULT 1
	);

	CREATE INDEX IF NOT EXISTS idx_rate_limit_time ON recovery_rate_limits(last_request_at);

	-- Collections table
	CREATE TABLE IF NOT EXISTS collections (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		name TEXT NOT NULL,
		description TEXT,
		slug TEXT NOT NULL UNIQUE,
		theme TEXT NOT NULL DEFAULT 'dark',
		custom_css TEXT,
		visibility TEXT NOT NULL DEFAULT 'private',
		secret_token TEXT,
		cover_photo_id TEXT REFERENCES photos(id) ON DELETE SET NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_collections_user_id ON collections(user_id);
	CREATE INDEX IF NOT EXISTS idx_collections_slug ON collections(slug);
	CREATE INDEX IF NOT EXISTS idx_collections_secret_token ON collections(secret_token);

	-- Collection photos (junction table)
	CREATE TABLE IF NOT EXISTS collection_photos (
		id TEXT PRIMARY KEY,
		collection_id TEXT NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
		photo_id TEXT NOT NULL REFERENCES photos(id) ON DELETE CASCADE,
		position INTEGER NOT NULL DEFAULT 0,
		added_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(collection_id, photo_id)
	);

	CREATE INDEX IF NOT EXISTS idx_collection_photos_collection_id ON collection_photos(collection_id);
	CREATE INDEX IF NOT EXISTS idx_collection_photos_photo_id ON collection_photos(photo_id);

	-- Collection shares (for registered users)
	CREATE TABLE IF NOT EXISTS collection_shares (
		id TEXT PRIMARY KEY,
		collection_id TEXT NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
		user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(collection_id, user_id)
	);

	CREATE INDEX IF NOT EXISTS idx_collection_shares_collection_id ON collection_shares(collection_id);
	CREATE INDEX IF NOT EXISTS idx_collection_shares_user_id ON collection_shares(user_id);
	`

	_, err := db.Exec(schema)
	return err
}

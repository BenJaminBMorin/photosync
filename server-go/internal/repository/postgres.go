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
	// First, run migrations for existing tables
	if err := runPostgresMigrations(db); err != nil {
		return err
	}

	schema := `
	-- Users table
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		email TEXT UNIQUE NOT NULL,
		display_name TEXT NOT NULL,
		api_key TEXT UNIQUE NOT NULL,
		api_key_hash TEXT NOT NULL,
		is_admin BOOLEAN NOT NULL DEFAULT FALSE,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		is_active BOOLEAN NOT NULL DEFAULT TRUE
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
		registered_at TIMESTAMP NOT NULL DEFAULT NOW(),
		last_seen_at TIMESTAMP NOT NULL DEFAULT NOW(),
		is_active BOOLEAN NOT NULL DEFAULT TRUE
	);

	CREATE INDEX IF NOT EXISTS idx_devices_user_id ON devices(user_id);
	CREATE INDEX IF NOT EXISTS idx_devices_fcm_token ON devices(fcm_token);

	-- Auth requests (pending push approvals)
	CREATE TABLE IF NOT EXISTS auth_requests (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		status TEXT NOT NULL DEFAULT 'pending',
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		expires_at TIMESTAMP NOT NULL,
		responded_at TIMESTAMP,
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
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		expires_at TIMESTAMP NOT NULL,
		responded_at TIMESTAMP,
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
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		expires_at TIMESTAMP NOT NULL,
		last_activity_at TIMESTAMP NOT NULL DEFAULT NOW(),
		ip_address TEXT,
		user_agent TEXT,
		is_active BOOLEAN NOT NULL DEFAULT TRUE
	);

	CREATE INDEX IF NOT EXISTS idx_web_sessions_user_id ON web_sessions(user_id);

	-- Photos table (with user_id, thumbnails, EXIF metadata, GPS)
	CREATE TABLE IF NOT EXISTS photos (
		id TEXT PRIMARY KEY,
		user_id TEXT REFERENCES users(id),
		original_filename TEXT NOT NULL,
		stored_path TEXT NOT NULL,
		file_hash TEXT NOT NULL,
		file_size BIGINT NOT NULL,
		date_taken TIMESTAMP NOT NULL,
		uploaded_at TIMESTAMP NOT NULL,

		-- Thumbnail paths (relative to storage base)
		thumb_small TEXT,
		thumb_medium TEXT,
		thumb_large TEXT,

		-- EXIF Metadata
		camera_make TEXT,
		camera_model TEXT,
		lens_model TEXT,
		focal_length TEXT,
		aperture TEXT,
		shutter_speed TEXT,
		iso INTEGER,
		orientation INTEGER DEFAULT 1,

		-- GPS Location
		latitude DOUBLE PRECISION,
		longitude DOUBLE PRECISION,
		altitude DOUBLE PRECISION,

		-- Image dimensions
		width INTEGER,
		height INTEGER
	);

	CREATE INDEX IF NOT EXISTS idx_photos_hash ON photos(file_hash);
	CREATE INDEX IF NOT EXISTS idx_photos_date ON photos(date_taken);
	CREATE INDEX IF NOT EXISTS idx_photos_user_id ON photos(user_id);

	-- Setup config table
	CREATE TABLE IF NOT EXISTS setup_config (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		updated_at TIMESTAMP NOT NULL DEFAULT NOW()
	);

	-- Bootstrap keys (emergency admin access)
	CREATE TABLE IF NOT EXISTS bootstrap_keys (
		id TEXT PRIMARY KEY,
		key_hash TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		expires_at TIMESTAMP NOT NULL,
		used BOOLEAN NOT NULL DEFAULT FALSE,
		used_at TIMESTAMP,
		used_by TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_bootstrap_expires ON bootstrap_keys(expires_at);

	-- Recovery tokens (email-based account recovery)
	CREATE TABLE IF NOT EXISTS recovery_tokens (
		id TEXT PRIMARY KEY,
		token_hash TEXT NOT NULL,
		user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		email TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		expires_at TIMESTAMP NOT NULL,
		used BOOLEAN NOT NULL DEFAULT FALSE,
		used_at TIMESTAMP,
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
		requires_restart BOOLEAN NOT NULL DEFAULT FALSE,
		is_sensitive BOOLEAN NOT NULL DEFAULT FALSE,
		updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
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
		use_tls BOOLEAN NOT NULL DEFAULT TRUE,
		skip_verify BOOLEAN NOT NULL DEFAULT FALSE,
		updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_by TEXT NOT NULL REFERENCES users(id),
		CHECK (id = 1)
	);

	-- Recovery rate limits
	CREATE TABLE IF NOT EXISTS recovery_rate_limits (
		email TEXT PRIMARY KEY,
		last_request_at TIMESTAMP NOT NULL DEFAULT NOW(),
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
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW()
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
		added_at TIMESTAMP NOT NULL DEFAULT NOW(),
		UNIQUE(collection_id, photo_id)
	);

	CREATE INDEX IF NOT EXISTS idx_collection_photos_collection_id ON collection_photos(collection_id);
	CREATE INDEX IF NOT EXISTS idx_collection_photos_photo_id ON collection_photos(photo_id);

	-- Collection shares (for registered users)
	CREATE TABLE IF NOT EXISTS collection_shares (
		id TEXT PRIMARY KEY,
		collection_id TEXT NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
		user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		UNIQUE(collection_id, user_id)
	);

	CREATE INDEX IF NOT EXISTS idx_collection_shares_collection_id ON collection_shares(collection_id);
	CREATE INDEX IF NOT EXISTS idx_collection_shares_user_id ON collection_shares(user_id);
	`

	_, err := db.Exec(schema)
	return err
}

// runPostgresMigrations handles incremental migrations for existing tables
func runPostgresMigrations(db *sql.DB) error {
	// Check if photos table exists
	var tableExists bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_name = 'photos'
		)
	`).Scan(&tableExists)

	if err != nil || !tableExists {
		return nil
	}

	// Add new columns if they don't exist (for migration from older schema)
	migrations := []string{
		`ALTER TABLE photos ADD COLUMN IF NOT EXISTS user_id TEXT`,
		`ALTER TABLE photos ADD COLUMN IF NOT EXISTS thumb_small TEXT`,
		`ALTER TABLE photos ADD COLUMN IF NOT EXISTS thumb_medium TEXT`,
		`ALTER TABLE photos ADD COLUMN IF NOT EXISTS thumb_large TEXT`,
		`ALTER TABLE photos ADD COLUMN IF NOT EXISTS camera_make TEXT`,
		`ALTER TABLE photos ADD COLUMN IF NOT EXISTS camera_model TEXT`,
		`ALTER TABLE photos ADD COLUMN IF NOT EXISTS lens_model TEXT`,
		`ALTER TABLE photos ADD COLUMN IF NOT EXISTS focal_length TEXT`,
		`ALTER TABLE photos ADD COLUMN IF NOT EXISTS aperture TEXT`,
		`ALTER TABLE photos ADD COLUMN IF NOT EXISTS shutter_speed TEXT`,
		`ALTER TABLE photos ADD COLUMN IF NOT EXISTS iso INTEGER`,
		`ALTER TABLE photos ADD COLUMN IF NOT EXISTS orientation INTEGER DEFAULT 1`,
		`ALTER TABLE photos ADD COLUMN IF NOT EXISTS latitude DOUBLE PRECISION`,
		`ALTER TABLE photos ADD COLUMN IF NOT EXISTS longitude DOUBLE PRECISION`,
		`ALTER TABLE photos ADD COLUMN IF NOT EXISTS altitude DOUBLE PRECISION`,
		`ALTER TABLE photos ADD COLUMN IF NOT EXISTS width INTEGER`,
		`ALTER TABLE photos ADD COLUMN IF NOT EXISTS height INTEGER`,
	}

	for _, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			return err
		}
	}

	// Create partial index for photos with GPS coordinates (for map view)
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_photos_location ON photos(latitude, longitude) WHERE latitude IS NOT NULL`)
	if err != nil {
		// Index might already exist with different definition, ignore
	}

	return nil
}

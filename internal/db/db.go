package db

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps the database connection
type DB struct {
	*sql.DB
	dbPath string
}

// Init initializes the database connection and runs migrations
func Init(dbPath string) (*DB, error) {
	// Ensure data directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	// Open database connection
	sqlDB, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
	if err != nil {
		return nil, err
	}

	db := &DB{sqlDB, dbPath}

	// Run migrations
	if err := db.migrate(); err != nil {
		sqlDB.Close()
		return nil, err
	}

	return db, nil
}

// GetDBPath returns the database file path
func (db *DB) GetDBPath() string {
	return db.dbPath
}

// // Init initializes the database connection and runs migrations
// func Init(dbPath string) (*DB, error) {
// 	// Ensure data directory exists
// 	dir := filepath.Dir(dbPath)
// 	if err := os.MkdirAll(dir, 0755); err != nil {
// 		return nil, err
// 	}

// 	// Open database connection
// 	sqlDB, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
// 	if err != nil {
// 		return nil, err
// 	}

// 	db := &DB{sqlDB}

// 	// Run migrations
// 	if err := db.migrate(); err != nil {
// 		sqlDB.Close()
// 		return nil, err
// 	}

// 	return db, nil
// }

// migrate runs database migrations
func (db *DB) migrate() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS apps (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			description TEXT,
			compose_content TEXT NOT NULL,
			tunnel_token TEXT,
			tunnel_id TEXT,
			tunnel_domain TEXT,
			public_url TEXT,
			status TEXT NOT NULL DEFAULT 'stopped',
			error_message TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			password TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS settings (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			cloudflare_api_token TEXT,
			cloudflare_account_id TEXT,
			auth_enabled INTEGER NOT NULL DEFAULT 0,
			auto_start_apps INTEGER NOT NULL DEFAULT 0,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS cloudflare_tunnels (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			app_id INTEGER NOT NULL,
			tunnel_id TEXT NOT NULL,
			tunnel_name TEXT NOT NULL,
			tunnel_token TEXT NOT NULL,
			account_id TEXT NOT NULL,
			is_active INTEGER NOT NULL DEFAULT 1,
			status TEXT NOT NULL DEFAULT 'active',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			last_synced_at DATETIME,
			error_details TEXT,
			UNIQUE(app_id),
			FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE
		)`,
		// Add missing columns if they don't exist (for existing databases)
		`ALTER TABLE settings ADD COLUMN auto_start_apps INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE apps ADD COLUMN error_message TEXT`,
	}

	for _, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			// Ignore error if column already exists
			if !isDuplicateColumnError(err) {
				return err
			}
		}
	}

	// Create default settings row if it doesn't exist
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM settings").Scan(&count); err != nil {
		log.Printf("Error checking settings: %v", err)
	} else if count == 0 {
		db.Exec("INSERT INTO settings (auth_enabled, auto_start_apps) VALUES (0, 0)")
	}

	return nil
}

// isDuplicateColumnError checks if error is about duplicate column
func isDuplicateColumnError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "duplicate column name") ||
		strings.Contains(errStr, "already exists")
}

// CreateApp creates a new app
func (db *DB) CreateApp(app *App) error {
	var errorMessage interface{}
	if app.ErrorMessage != nil {
		errorMessage = *app.ErrorMessage
	} else {
		errorMessage = nil
	}

	result, err := db.Exec(
		"INSERT INTO apps (name, description, compose_content, tunnel_token, tunnel_id, tunnel_domain, public_url, status, error_message, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		app.Name, app.Description, app.ComposeContent, app.TunnelToken, app.TunnelID, app.TunnelDomain, app.PublicURL, app.Status, errorMessage, time.Now(), time.Now(),
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	app.ID = id
	return nil
}

// GetAllApps retrieves all apps
func (db *DB) GetAllApps() ([]*App, error) {
	rows, err := db.Query("SELECT id, name, description, compose_content, tunnel_token, tunnel_id, tunnel_domain, public_url, status, error_message, created_at, updated_at FROM apps ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apps []*App
	for rows.Next() {
		app := &App{}
		err := rows.Scan(&app.ID, &app.Name, &app.Description, &app.ComposeContent, &app.TunnelToken, &app.TunnelID, &app.TunnelDomain, &app.PublicURL, &app.Status, &app.ErrorMessage, &app.CreatedAt, &app.UpdatedAt)
		if err != nil {
			return nil, err
		}
		apps = append(apps, app)
	}

	return apps, nil
}

// GetApp retrieves an app by ID
func (db *DB) GetApp(id int64) (*App, error) {
	app := &App{}
	var errorMessage sql.NullString
	err := db.QueryRow(
		"SELECT id, name, description, compose_content, tunnel_token, tunnel_id, tunnel_domain, public_url, status, error_message, created_at, updated_at FROM apps WHERE id = ?",
		id,
	).Scan(&app.ID, &app.Name, &app.Description, &app.ComposeContent, &app.TunnelToken, &app.TunnelID, &app.TunnelDomain, &app.PublicURL, &app.Status, &errorMessage, &app.CreatedAt, &app.UpdatedAt)

	if err == nil {
		if errorMessage.Valid {
			app.ErrorMessage = &errorMessage.String
		} else {
			app.ErrorMessage = nil
		}
	}
	return app, err
}

// ListApps retrieves all apps
func (db *DB) ListApps() ([]*App, error) {
	rows, err := db.Query("SELECT id, name, description, compose_content, tunnel_token, tunnel_id, tunnel_domain, public_url, status, error_message, created_at, updated_at FROM apps ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apps []*App
	for rows.Next() {
		app := &App{}
		err := rows.Scan(&app.ID, &app.Name, &app.Description, &app.ComposeContent, &app.TunnelToken, &app.TunnelID, &app.TunnelDomain, &app.PublicURL, &app.Status, &app.ErrorMessage, &app.CreatedAt, &app.UpdatedAt)
		if err != nil {
			return nil, err
		}
		apps = append(apps, app)
	}

	return apps, nil
}

// UpdateApp updates an app
func (db *DB) UpdateApp(app *App) error {
	var errorMessage interface{}
	if app.ErrorMessage != nil {
		errorMessage = *app.ErrorMessage
	} else {
		errorMessage = nil
	}

	_, err := db.Exec(
		"UPDATE apps SET name = ?, description = ?, compose_content = ?, tunnel_token = ?, tunnel_id = ?, tunnel_domain = ?, public_url = ?, status = ?, error_message = ?, updated_at = ? WHERE id = ?",
		app.Name, app.Description, app.ComposeContent, app.TunnelToken, app.TunnelID, app.TunnelDomain, app.PublicURL, app.Status, errorMessage, time.Now(), app.ID,
	)
	return err
}

// DeleteApp deletes an app
func (db *DB) DeleteApp(id int64) error {
	_, err := db.Exec("DELETE FROM apps WHERE id = ?", id)
	return err
}

// GetSettings retrieves the settings
func (db *DB) GetSettings() (*Settings, error) {
	settings := &Settings{}
	err := db.QueryRow(
		"SELECT id, cloudflare_api_token, cloudflare_account_id, auth_enabled, auto_start_apps, updated_at FROM settings WHERE id = 1",
	).Scan(&settings.ID, &settings.CloudflareAPIToken, &settings.CloudflareAccountID, &settings.AuthEnabled, &settings.AutoStartApps, &settings.UpdatedAt)
	return settings, err
}

// UpdateSettings updates the settings
func (db *DB) UpdateSettings(settings *Settings) error {
	var apiToken, accountID interface{}
	if settings.CloudflareAPIToken != nil {
		apiToken = *settings.CloudflareAPIToken
	} else {
		apiToken = nil
	}
	if settings.CloudflareAccountID != nil {
		accountID = *settings.CloudflareAccountID
	} else {
		accountID = nil
	}
	_, err := db.Exec(
		"UPDATE settings SET cloudflare_api_token = ?, cloudflare_account_id = ?, auth_enabled = ?, auto_start_apps = ?, updated_at = ? WHERE id = 1",
		apiToken, accountID, settings.AuthEnabled, settings.AutoStartApps, time.Now(),
	)
	return err
}

// CreateUser creates a new user
func (db *DB) CreateUser(user *User) error {
	result, err := db.Exec(
		"INSERT INTO users (username, password, created_at) VALUES (?, ?, ?)",
		user.Username, user.Password, time.Now(),
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	user.ID = id
	return nil
}

// GetUser retrieves a user by username
func (db *DB) GetUser(username string) (*User, error) {
	user := &User{}
	err := db.QueryRow(
		"SELECT id, username, password, created_at FROM users WHERE username = ?",
		username,
	).Scan(&user.ID, &user.Username, &user.Password, &user.CreatedAt)
	return user, err
}

// CreateCloudflareTunnel creates a new Cloudflare tunnel record
func (db *DB) CreateCloudflareTunnel(tunnel *CloudflareTunnel) error {
	var errorDetails interface{}
	if tunnel.ErrorDetails != nil {
		errorDetails = *tunnel.ErrorDetails
	} else {
		errorDetails = nil
	}

	result, err := db.Exec(
		"INSERT INTO cloudflare_tunnels (app_id, tunnel_id, tunnel_name, tunnel_token, account_id, is_active, status, created_at, updated_at, last_synced_at, error_details) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		tunnel.AppID, tunnel.TunnelID, tunnel.TunnelName, tunnel.TunnelToken, tunnel.AccountID, tunnel.IsActive, tunnel.Status, time.Now(), time.Now(), tunnel.LastSyncedAt, errorDetails,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	tunnel.ID = id
	return nil
}

// GetCloudflareTunnelByAppID retrieves a Cloudflare tunnel by app ID
func (db *DB) GetCloudflareTunnelByAppID(appID int64) (*CloudflareTunnel, error) {
	tunnel := &CloudflareTunnel{}
	var errorDetails sql.NullString
	var lastSyncedAt interface{} // Use interface{} to handle NULL values
	err := db.QueryRow(
		"SELECT id, app_id, tunnel_id, tunnel_name, tunnel_token, account_id, is_active, status, created_at, updated_at, last_synced_at, error_details FROM cloudflare_tunnels WHERE app_id = ?",
		appID,
	).Scan(&tunnel.ID, &tunnel.AppID, &tunnel.TunnelID, &tunnel.TunnelName, &tunnel.TunnelToken, &tunnel.AccountID, &tunnel.IsActive, &tunnel.Status, &tunnel.CreatedAt, &tunnel.UpdatedAt, &lastSyncedAt, &errorDetails)
	
	// Handle NULL last_synced_at
	if err == nil {
		if lastSyncedAt != nil {
			// Convert to time.Time if not NULL
			if t, ok := lastSyncedAt.(time.Time); ok {
				tunnel.LastSyncedAt = &t
			} else {
				// Fallback to zero time if type is unexpected
				zeroTime := time.Time{}
				tunnel.LastSyncedAt = &zeroTime
			}
		} else {
			tunnel.LastSyncedAt = nil
		}

		// Handle error_details
		if errorDetails.Valid {
			tunnel.ErrorDetails = &errorDetails.String
		} else {
			tunnel.ErrorDetails = nil
		}
	}
	return tunnel, err
}

// UpdateCloudflareTunnel updates a Cloudflare tunnel record
func (db *DB) UpdateCloudflareTunnel(tunnel *CloudflareTunnel) error {
	var errorDetails interface{}
	if tunnel.ErrorDetails != nil {
		errorDetails = *tunnel.ErrorDetails
	} else {
		errorDetails = nil
	}

	_, err := db.Exec(
		"UPDATE cloudflare_tunnels SET tunnel_name = ?, is_active = ?, status = ?, updated_at = ?, last_synced_at = ?, error_details = ? WHERE id = ?",
		tunnel.TunnelName, tunnel.IsActive, tunnel.Status, time.Now(), tunnel.LastSyncedAt, errorDetails, tunnel.ID,
	)
	return err
}

// DeleteCloudflareTunnel deletes a Cloudflare tunnel record
func (db *DB) DeleteCloudflareTunnel(appID int64) error {
	_, err := db.Exec("DELETE FROM cloudflare_tunnels WHERE app_id = ?", appID)
	return err
}

// ListActiveCloudflareTunnels retrieves all active Cloudflare tunnels
func (db *DB) ListActiveCloudflareTunnels() ([]*CloudflareTunnel, error) {
	rows, err := db.Query("SELECT id, app_id, tunnel_id, tunnel_name, tunnel_token, account_id, is_active, status, created_at, updated_at, last_synced_at, error_details FROM cloudflare_tunnels WHERE is_active = 1 ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tunnels []*CloudflareTunnel
	for rows.Next() {
		tunnel := &CloudflareTunnel{}
		var lastSyncedAt interface{}
		var errorDetails sql.NullString
		err := rows.Scan(&tunnel.ID, &tunnel.AppID, &tunnel.TunnelID, &tunnel.TunnelName, &tunnel.TunnelToken, &tunnel.AccountID, &tunnel.IsActive, &tunnel.Status, &tunnel.CreatedAt, &tunnel.UpdatedAt, &lastSyncedAt, &errorDetails)
		if err != nil {
			return nil, err
		}

		// Handle NULL last_synced_at
		if lastSyncedAt != nil {
			// Convert to time.Time if not NULL
			if t, ok := lastSyncedAt.(time.Time); ok {
				tunnel.LastSyncedAt = &t
			} else {
				// Fallback to zero time if type is unexpected
				zeroTime := time.Time{}
				tunnel.LastSyncedAt = &zeroTime
			}
		} else {
			tunnel.LastSyncedAt = nil
		}

		if errorDetails.Valid {
			tunnel.ErrorDetails = &errorDetails.String
		} else {
			tunnel.ErrorDetails = nil
		}

		tunnels = append(tunnels, tunnel)
	}

	return tunnels, nil
}

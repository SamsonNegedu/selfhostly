package db

import (
	"database/sql"
	"encoding/json"
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

// migrate runs database migrations
func (db *DB) migrate() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS apps (
			id TEXT PRIMARY KEY,
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
			id TEXT PRIMARY KEY,
			username TEXT NOT NULL UNIQUE,
			password TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS settings (
			id TEXT PRIMARY KEY,
			cloudflare_api_token TEXT,
			cloudflare_account_id TEXT,
			auto_start_apps INTEGER NOT NULL DEFAULT 0,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS cloudflare_tunnels (
			id TEXT PRIMARY KEY,
			app_id TEXT NOT NULL,
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
			ingress_rules TEXT,
			UNIQUE(app_id),
			FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE
		)`,
	}

	for _, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			// Ignore error if column already exists
			if !isDuplicateColumnError(err) {
				return err
			}
		}
	}

	// Check if settings exist and have proper UUIDs
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM settings").Scan(&count); err != nil {
		log.Printf("Error checking settings: %v", err)
		return err
	}

	// If settings exist but have no UUIDs, fix them
	if count > 0 {
		var uuidCount int
		if err := db.QueryRow("SELECT COUNT(*) FROM settings WHERE id IS NOT NULL").Scan(&uuidCount); err != nil {
			log.Printf("Error checking UUIDs: %v", err)
		} else if uuidCount == 0 {
			// All settings have NULL IDs, need to fix them
			settings := NewSettings()
			if _, err := db.Exec("UPDATE settings SET id = ?, updated_at = ? WHERE id IS NULL",
				settings.ID, time.Now()); err != nil {
				log.Printf("Error fixing settings UUIDs: %v", err)
			}
		}
	}

	// Create default settings row if none exist
	if count == 0 {
		settings := NewSettings()
		if _, err := db.Exec("INSERT INTO settings (id, cloudflare_api_token, cloudflare_account_id, auto_start_apps, updated_at) VALUES (?, ?, ?, ?, ?)",
			settings.ID, settings.CloudflareAPIToken, settings.CloudflareAccountID, settings.AutoStartApps, settings.UpdatedAt); err != nil {
			log.Printf("Error inserting default settings: %v", err)
		}
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

	_, err := db.Exec(
		"INSERT INTO apps (id, name, description, compose_content, tunnel_token, tunnel_id, tunnel_domain, public_url, status, error_message, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		app.ID, app.Name, app.Description, app.ComposeContent, app.TunnelToken, app.TunnelID, app.TunnelDomain, app.PublicURL, app.Status, errorMessage, app.CreatedAt, time.Now(),
	)
	if err != nil {
		return err
	}

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
func (db *DB) GetApp(id string) (*App, error) {
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
func (db *DB) DeleteApp(id string) error {
	_, err := db.Exec("DELETE FROM apps WHERE id = ?", id)
	return err
}

// GetSettings retrieves the settings
func (db *DB) GetSettings() (*Settings, error) {
	settings := &Settings{}
	err := db.QueryRow(
		"SELECT id, cloudflare_api_token, cloudflare_account_id, auto_start_apps, updated_at FROM settings LIMIT 1",
	).Scan(&settings.ID, &settings.CloudflareAPIToken, &settings.CloudflareAccountID, &settings.AutoStartApps, &settings.UpdatedAt)

	if err != nil {
		// If no settings exist, create default settings
		if strings.Contains(err.Error(), "no rows in result set") {
			settings = NewSettings()
			if err := db.UpdateSettings(settings); err != nil {
				return nil, err
			}
			return settings, nil
		}
		return nil, err
	}

	return settings, nil
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
		"UPDATE settings SET cloudflare_api_token = ?, cloudflare_account_id = ?, auto_start_apps = ?, updated_at = ? WHERE id = ?",
		apiToken, accountID, settings.AutoStartApps, time.Now(), settings.ID,
	)
	return err
}

// CreateUser creates a new user
func (db *DB) CreateUser(user *User) error {
	_, err := db.Exec(
		"INSERT INTO users (id, username, password, created_at) VALUES (?, ?, ?, ?)",
		user.ID, user.Username, user.Password, user.CreatedAt,
	)
	if err != nil {
		return err
	}

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
	var errorDetails, ingressRules interface{}
	if tunnel.ErrorDetails != nil {
		errorDetails = *tunnel.ErrorDetails
	} else {
		errorDetails = nil
	}

	if tunnel.IngressRules != nil {
		// Serialize ingress rules to JSON for storage
		if jsonRules, err := json.Marshal(tunnel.IngressRules); err == nil {
			ingressRules = string(jsonRules)
		} else {
			ingressRules = nil
		}
	} else {
		ingressRules = nil
	}

	_, err := db.Exec(
		"INSERT INTO cloudflare_tunnels (id, app_id, tunnel_id, tunnel_name, tunnel_token, account_id, is_active, status, ingress_rules, created_at, updated_at, last_synced_at, error_details) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		tunnel.ID, tunnel.AppID, tunnel.TunnelID, tunnel.TunnelName, tunnel.TunnelToken, tunnel.AccountID, tunnel.IsActive, tunnel.Status, ingressRules, tunnel.CreatedAt, time.Now(), tunnel.LastSyncedAt, errorDetails,
	)
	if err != nil {
		return err
	}

	return nil
}

// GetCloudflareTunnelByAppID retrieves a Cloudflare tunnel by app ID
func (db *DB) GetCloudflareTunnelByAppID(appID string) (*CloudflareTunnel, error) {
	tunnel := &CloudflareTunnel{}
	var errorDetails sql.NullString
	var lastSyncedAt, ingressRules interface{} // Use interface{} to handle NULL values
	err := db.QueryRow(
		"SELECT id, app_id, tunnel_id, tunnel_name, tunnel_token, account_id, is_active, status, ingress_rules, created_at, updated_at, last_synced_at, error_details FROM cloudflare_tunnels WHERE app_id = ?",
		appID,
	).Scan(&tunnel.ID, &tunnel.AppID, &tunnel.TunnelID, &tunnel.TunnelName, &tunnel.TunnelToken, &tunnel.AccountID, &tunnel.IsActive, &tunnel.Status, &ingressRules, &tunnel.CreatedAt, &tunnel.UpdatedAt, &lastSyncedAt, &errorDetails)

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

		// Handle ingress_rules
		if ingressRules != nil {
			if rulesStr, ok := ingressRules.(string); ok {
				var parsedRules []IngressRule
				if err := json.Unmarshal([]byte(rulesStr), &parsedRules); err == nil {
					tunnel.IngressRules = &parsedRules
				} else {
					tunnel.IngressRules = nil
				}
			} else {
				tunnel.IngressRules = nil
			}
		} else {
			tunnel.IngressRules = nil
		}
	}
	return tunnel, err
}

// UpdateCloudflareTunnel updates a Cloudflare tunnel record
func (db *DB) UpdateCloudflareTunnel(tunnel *CloudflareTunnel) error {
	var errorDetails, ingressRules interface{}
	if tunnel.ErrorDetails != nil {
		errorDetails = *tunnel.ErrorDetails
	} else {
		errorDetails = nil
	}

	if tunnel.IngressRules != nil {
		// Serialize ingress rules to JSON for storage
		if jsonRules, err := json.Marshal(tunnel.IngressRules); err == nil {
			ingressRules = string(jsonRules)
		} else {
			ingressRules = nil
		}
	} else {
		ingressRules = nil
	}

	_, err := db.Exec(
		"UPDATE cloudflare_tunnels SET tunnel_name = ?, is_active = ?, status = ?, ingress_rules = ?, updated_at = ?, last_synced_at = ?, error_details = ? WHERE id = ?",
		tunnel.TunnelName, tunnel.IsActive, tunnel.Status, ingressRules, time.Now(), tunnel.LastSyncedAt, errorDetails, tunnel.ID,
	)
	return err
}

// DeleteCloudflareTunnel deletes a Cloudflare tunnel record
func (db *DB) DeleteCloudflareTunnel(appID string) error {
	_, err := db.Exec("DELETE FROM cloudflare_tunnels WHERE app_id = ?", appID)
	return err
}

// GetCloudflareTunnelByTunnelID retrieves a Cloudflare tunnel by tunnel ID
func (db *DB) GetCloudflareTunnelByTunnelID(tunnelID string) (*CloudflareTunnel, error) {
	tunnel := &CloudflareTunnel{}
	var errorDetails sql.NullString
	var lastSyncedAt, ingressRules interface{} // Use interface{} to handle NULL values
	err := db.QueryRow(
		"SELECT id, app_id, tunnel_id, tunnel_name, tunnel_token, account_id, is_active, status, ingress_rules, created_at, updated_at, last_synced_at, error_details FROM cloudflare_tunnels WHERE tunnel_id = ?",
		tunnelID,
	).Scan(&tunnel.ID, &tunnel.AppID, &tunnel.TunnelID, &tunnel.TunnelName, &tunnel.TunnelToken, &tunnel.AccountID, &tunnel.IsActive, &tunnel.Status, &ingressRules, &tunnel.CreatedAt, &tunnel.UpdatedAt, &lastSyncedAt, &errorDetails)

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

		// Handle ingress_rules
		if ingressRules != nil {
			if rulesStr, ok := ingressRules.(string); ok {
				var parsedRules []IngressRule
				if err := json.Unmarshal([]byte(rulesStr), &parsedRules); err == nil {
					tunnel.IngressRules = &parsedRules
				} else {
					tunnel.IngressRules = nil
				}
			} else {
				tunnel.IngressRules = nil
			}
		} else {
			tunnel.IngressRules = nil
		}
	}
	return tunnel, err
}

// ListActiveCloudflareTunnels retrieves all active Cloudflare tunnels
func (db *DB) ListActiveCloudflareTunnels() ([]*CloudflareTunnel, error) {
	rows, err := db.Query("SELECT id, app_id, tunnel_id, tunnel_name, tunnel_token, account_id, is_active, status, ingress_rules, created_at, updated_at, last_synced_at, error_details FROM cloudflare_tunnels WHERE is_active = 1 ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tunnels []*CloudflareTunnel
	for rows.Next() {
		tunnel := &CloudflareTunnel{}
		var lastSyncedAt, ingressRules interface{}
		var errorDetails sql.NullString
		err := rows.Scan(&tunnel.ID, &tunnel.AppID, &tunnel.TunnelID, &tunnel.TunnelName, &tunnel.TunnelToken, &tunnel.AccountID, &tunnel.IsActive, &tunnel.Status, &ingressRules, &tunnel.CreatedAt, &tunnel.UpdatedAt, &lastSyncedAt, &errorDetails)
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

		// Handle ingress_rules
		if ingressRules != nil {
			if rulesStr, ok := ingressRules.(string); ok {
				var parsedRules []IngressRule
				if err := json.Unmarshal([]byte(rulesStr), &parsedRules); err == nil {
					tunnel.IngressRules = &parsedRules
				} else {
					tunnel.IngressRules = nil
				}
			} else {
				tunnel.IngressRules = nil
			}
		} else {
			tunnel.IngressRules = nil
		}

		tunnels = append(tunnels, tunnel)
	}

	return tunnels, nil
}

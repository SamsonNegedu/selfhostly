package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/selfhostly/internal/config"
	"github.com/selfhostly/internal/constants"
	_ "modernc.org/sqlite"
)

// DB wraps the database connection
type DB struct {
	*sql.DB
	dbPath string
}

// Tx wraps a database transaction
type Tx struct {
	*sql.Tx
}

// Init initializes the database connection and runs migrations
func Init(dbPath string) (*DB, error) {
	// Ensure data directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	// Open database connection with foreign keys enabled
	sqlDB, err := sql.Open("sqlite", dbPath+"?_foreign_keys=on")
	if err != nil {
		return nil, err
	}

	db := &DB{sqlDB, dbPath}

	// Configure SQLite for reliability and performance
	if err := db.configureSQLite(); err != nil {
		sqlDB.Close()
		return nil, err
	}

	// Run migrations
	if err := db.migrate(); err != nil {
		sqlDB.Close()
		return nil, err
	}

	// Run integrity check
	if err := db.IntegrityCheck(); err != nil {
		slog.Warn("Database integrity check found issues", "error", err)
		// Don't fail startup, but log the issue
	}

	return db, nil
}

// configureSQLite sets optimal SQLite pragmas for reliability and performance
func (db *DB) configureSQLite() error {
	pragmas := []string{
		// Enable WAL mode for better concurrency and crash recovery
		// WAL allows readers and writers to operate concurrently
		"PRAGMA journal_mode=WAL",

		// NORMAL synchronous mode is safe with WAL and faster than FULL
		// Data is written to disk at critical moments but not after every transaction
		"PRAGMA synchronous=NORMAL",

		// Wait up to 5 seconds when database is locked instead of failing immediately
		"PRAGMA busy_timeout=5000",

		// Store temporary tables and indices in memory for performance
		"PRAGMA temp_store=MEMORY",

		// Use memory-mapped I/O for better performance (128MB)
		"PRAGMA mmap_size=134217728",

		// Increase cache size to 10MB (negative value = KB)
		"PRAGMA cache_size=-10000",
	}

	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			slog.Error("Failed to set pragma", "pragma", pragma, "error", err)
			return err
		}
	}

	// Verify WAL mode was enabled
	var journalMode string
	if err := db.QueryRow("PRAGMA journal_mode").Scan(&journalMode); err != nil {
		return err
	}

	slog.Info("SQLite configured", "journal_mode", journalMode, "synchronous", "NORMAL")
	return nil
}

// IntegrityCheck runs SQLite's integrity check
func (db *DB) IntegrityCheck() error {
	var result string
	if err := db.QueryRow("PRAGMA integrity_check").Scan(&result); err != nil {
		return err
	}

	if result != "ok" {
		return fmt.Errorf("integrity check failed: %s", result)
	}

	slog.Debug("Database integrity check passed")
	return nil
}

// BeginTx starts a new transaction
func (db *DB) BeginTx(ctx context.Context) (*Tx, error) {
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &Tx{tx}, nil
}

// CreateAppTx creates a new app within a transaction
func (tx *Tx) CreateApp(app *App) error {
	var errorMessage interface{}
	if app.ErrorMessage != nil {
		errorMessage = *app.ErrorMessage
	} else {
		errorMessage = nil
	}

	_, err := tx.Exec(
		"INSERT INTO apps (id, name, description, compose_content, tunnel_token, tunnel_id, tunnel_domain, public_url, status, error_message, node_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		app.ID, app.Name, app.Description, app.ComposeContent, app.TunnelToken, app.TunnelID, app.TunnelDomain, app.PublicURL, app.Status, errorMessage, app.NodeID, app.CreatedAt, time.Now(),
	)
	return err
}

// UpdateAppTx updates an app within a transaction
func (tx *Tx) UpdateApp(app *App) error {
	var errorMessage interface{}
	if app.ErrorMessage != nil {
		errorMessage = *app.ErrorMessage
	} else {
		errorMessage = nil
	}

	_, err := tx.Exec(
		"UPDATE apps SET name = ?, description = ?, compose_content = ?, tunnel_token = ?, tunnel_id = ?, tunnel_domain = ?, public_url = ?, status = ?, error_message = ?, tunnel_mode = ?, updated_at = ? WHERE id = ?",
		app.Name, app.Description, app.ComposeContent, app.TunnelToken, app.TunnelID, app.TunnelDomain, app.PublicURL, app.Status, errorMessage, app.TunnelMode, time.Now(), app.ID,
	)
	return err
}

// DeleteAppTx deletes an app within a transaction
func (tx *Tx) DeleteApp(id string) error {
	_, err := tx.Exec("DELETE FROM apps WHERE id = ?", id)
	return err
}

// migrate runs database migrations
func (db *DB) migrate() error {
	migrations := []string{
		// Nodes table - must be created before apps table for foreign key
		`CREATE TABLE IF NOT EXISTS nodes (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			api_endpoint TEXT NOT NULL,
			api_key TEXT NOT NULL,
			is_primary INTEGER NOT NULL DEFAULT 0,
			status TEXT NOT NULL DEFAULT 'online',
			last_seen DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
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
			node_id TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		// Add node_id column to existing apps table (for migrations)
		`ALTER TABLE apps ADD COLUMN node_id TEXT`,
		// Create index for faster node lookups
		`CREATE INDEX IF NOT EXISTS idx_apps_node_id ON apps(node_id)`,
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
		`CREATE TABLE IF NOT EXISTS compose_versions (
			id TEXT PRIMARY KEY,
			app_id TEXT NOT NULL,
			version INTEGER NOT NULL,
			compose_content TEXT NOT NULL,
			change_reason TEXT,
			changed_by TEXT,
			is_current INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			rolled_back_from INTEGER,
			UNIQUE(app_id, version),
			FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_compose_versions_app_id ON compose_versions(app_id)`,
		`CREATE INDEX IF NOT EXISTS idx_compose_versions_is_current ON compose_versions(app_id, is_current)`,
		// Add health check tracking columns to nodes table
		`ALTER TABLE nodes ADD COLUMN consecutive_failures INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE nodes ADD COLUMN last_health_check DATETIME`,
		// Add multi-provider tunnel support to settings table
		`ALTER TABLE settings ADD COLUMN active_tunnel_provider TEXT DEFAULT 'cloudflare'`,
		`ALTER TABLE settings ADD COLUMN tunnel_provider_config TEXT`,
		// Tunnel is source of truth for public_url (avoids app lookup when listing tunnels)
		`ALTER TABLE cloudflare_tunnels ADD COLUMN public_url TEXT`,
		// Quick Tunnel support: app tunnel type (custom = named tunnel, quick = trycloudflare.com, empty = none)
		`ALTER TABLE apps ADD COLUMN tunnel_mode TEXT DEFAULT ''`,
	}

	for _, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			// Ignore error if column already exists
			if !isDuplicateColumnError(err) {
				return err
			}
		}
	}

	// Migrate existing Cloudflare settings to new multi-provider structure
	if err := migrateCloudflareSettingsToProviderConfig(db.DB); err != nil {
		slog.Warn("Failed to migrate cloudflare settings", "error", err)
		// Don't fail the migration - old format still works with fallback
	}

	// Check if settings exist and have proper UUIDs
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM settings").Scan(&count); err != nil {
		slog.Error("Error checking settings", "error", err)
		return err
	}

	// If settings exist but have no UUIDs, fix them
	if count > 0 {
		var uuidCount int
		if err := db.QueryRow("SELECT COUNT(*) FROM settings WHERE id IS NOT NULL").Scan(&uuidCount); err != nil {
			slog.Error("Error checking UUIDs", "error", err)
		} else if uuidCount == 0 {
			// All settings have NULL IDs, need to fix them
			settings := NewSettings()
			if _, err := db.Exec("UPDATE settings SET id = ?, updated_at = ? WHERE id IS NULL",
				settings.ID, time.Now()); err != nil {
				slog.Error("Error fixing settings UUIDs", "error", err)
			}
		}
	}

	// Create default settings row if none exist
	if count == 0 {
		settings := NewSettings()
		if _, err := db.Exec("INSERT INTO settings (id, cloudflare_api_token, cloudflare_account_id, auto_start_apps, updated_at) VALUES (?, ?, ?, ?, ?)",
			settings.ID, settings.CloudflareAPIToken, settings.CloudflareAccountID, settings.AutoStartApps, settings.UpdatedAt); err != nil {
			slog.Error("Error inserting default settings", "error", err)
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

// GetProviderConfig parses the tunnel_provider_config JSON and returns configuration
// for the specified provider. Falls back to old cloudflare-specific fields if new
// config doesn't exist (backward compatibility).
func (settings *Settings) GetProviderConfig(providerName string) (map[string]interface{}, error) {
	// Try new provider config first
	if settings.TunnelProviderConfig != nil && *settings.TunnelProviderConfig != "" {
		var providerConfigs map[string]interface{}
		if err := json.Unmarshal([]byte(*settings.TunnelProviderConfig), &providerConfigs); err != nil {
			return nil, fmt.Errorf("failed to parse provider config: %w", err)
		}

		if config, ok := providerConfigs[providerName]; ok {
			if configMap, ok := config.(map[string]interface{}); ok {
				return configMap, nil
			}
		}
	}

	// Fallback to old cloudflare-specific fields for backward compatibility
	if providerName == constants.ProviderCloudflare {
		if settings.CloudflareAPIToken != nil && *settings.CloudflareAPIToken != "" &&
			settings.CloudflareAccountID != nil && *settings.CloudflareAccountID != "" {
			return map[string]interface{}{
				"api_token":  *settings.CloudflareAPIToken,
				"account_id": *settings.CloudflareAccountID,
			}, nil
		}
	}

	return nil, fmt.Errorf("provider %s not configured", providerName)
}

// GetActiveProviderName returns the active tunnel provider name.
// Falls back to "cloudflare" if not set (backward compatibility).
func (settings *Settings) GetActiveProviderName() string {
	if settings.ActiveTunnelProvider != nil && *settings.ActiveTunnelProvider != "" {
		return *settings.ActiveTunnelProvider
	}

	// Fallback: if cloudflare credentials exist, assume cloudflare
	if settings.CloudflareAPIToken != nil && *settings.CloudflareAPIToken != "" {
		return constants.ProviderCloudflare
	}

	return constants.DefaultProviderName // Default
}

// SetProviderConfig updates the configuration for a specific provider.
func (settings *Settings) SetProviderConfig(providerName string, config map[string]interface{}) error {
	var providerConfigs map[string]interface{}

	// Parse existing config if it exists
	if settings.TunnelProviderConfig != nil && *settings.TunnelProviderConfig != "" {
		if err := json.Unmarshal([]byte(*settings.TunnelProviderConfig), &providerConfigs); err != nil {
			return fmt.Errorf("failed to parse existing provider config: %w", err)
		}
	} else {
		providerConfigs = make(map[string]interface{})
	}

	// Update the specific provider's config
	providerConfigs[providerName] = config

	// Marshal back to JSON
	configJSON, err := json.Marshal(providerConfigs)
	if err != nil {
		return fmt.Errorf("failed to marshal provider config: %w", err)
	}

	configStr := string(configJSON)
	settings.TunnelProviderConfig = &configStr

	return nil
}

// InitNode initializes the node entry in the database (bootstrap for primary nodes)
func (db *DB) InitNode(cfg *config.Config) error {
	// Auto-bootstrap for existing single-node installations (primary only)
	if err := db.bootstrapSingleNode(cfg); err != nil {
		return err
	}

	return nil
}

// bootstrapSingleNode creates a node entry for existing installations
// and assigns all existing apps to it. Only runs on PRIMARY nodes.
func (db *DB) bootstrapSingleNode(cfg *config.Config) error {
	// Delegate to the standalone migration function
	return bootstrapSingleNode(db.DB, cfg)
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
		"INSERT INTO apps (id, name, description, compose_content, tunnel_token, tunnel_id, tunnel_domain, public_url, status, error_message, node_id, tunnel_mode, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		app.ID, app.Name, app.Description, app.ComposeContent, app.TunnelToken, app.TunnelID, app.TunnelDomain, app.PublicURL, app.Status, errorMessage, app.NodeID, app.TunnelMode, app.CreatedAt, time.Now(),
	)
	if err != nil {
		return err
	}

	return nil
}

// GetAllApps retrieves all apps
// SECURITY: Returns ALL apps without user filtering (single-user design)
// For multi-user support, implement GetUserApps(userID string) instead
func (db *DB) GetAllApps() ([]*App, error) {
	rows, err := db.Query("SELECT id, name, description, compose_content, tunnel_token, tunnel_id, tunnel_domain, public_url, status, error_message, node_id, tunnel_mode, created_at, updated_at FROM apps ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apps []*App
	for rows.Next() {
		app := &App{}
		var errorMessage sql.NullString
		var nodeID sql.NullString
		err := rows.Scan(&app.ID, &app.Name, &app.Description, &app.ComposeContent, &app.TunnelToken, &app.TunnelID, &app.TunnelDomain, &app.PublicURL, &app.Status, &errorMessage, &nodeID, &app.TunnelMode, &app.CreatedAt, &app.UpdatedAt)
		if err != nil {
			return nil, err
		}
		if errorMessage.Valid {
			app.ErrorMessage = &errorMessage.String
		} else {
			app.ErrorMessage = nil
		}
		if nodeID.Valid {
			app.NodeID = nodeID.String
		} else {
			app.NodeID = ""
		}
		apps = append(apps, app)
	}

	return apps, nil
}

// GetApp retrieves an app by ID
func (db *DB) GetApp(id string) (*App, error) {
	app := &App{}
	var errorMessage sql.NullString
	var nodeID sql.NullString
	err := db.QueryRow(
		"SELECT id, name, description, compose_content, tunnel_token, tunnel_id, tunnel_domain, public_url, status, error_message, node_id, tunnel_mode, created_at, updated_at FROM apps WHERE id = ?",
		id,
	).Scan(&app.ID, &app.Name, &app.Description, &app.ComposeContent, &app.TunnelToken, &app.TunnelID, &app.TunnelDomain, &app.PublicURL, &app.Status, &errorMessage, &nodeID, &app.TunnelMode, &app.CreatedAt, &app.UpdatedAt)

	if err == nil {
		if errorMessage.Valid {
			app.ErrorMessage = &errorMessage.String
		} else {
			app.ErrorMessage = nil
		}
		if nodeID.Valid {
			app.NodeID = nodeID.String
		} else {
			app.NodeID = ""
		}
	}
	return app, err
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
		"UPDATE apps SET name = ?, description = ?, compose_content = ?, tunnel_token = ?, tunnel_id = ?, tunnel_domain = ?, public_url = ?, status = ?, error_message = ?, tunnel_mode = ?, updated_at = ? WHERE id = ?",
		app.Name, app.Description, app.ComposeContent, app.TunnelToken, app.TunnelID, app.TunnelDomain, app.PublicURL, app.Status, errorMessage, app.TunnelMode, time.Now(), app.ID,
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
	var apiToken, accountID, activeTunnelProvider, tunnelProviderConfig interface{}
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
	if settings.ActiveTunnelProvider != nil {
		activeTunnelProvider = *settings.ActiveTunnelProvider
	} else {
		activeTunnelProvider = nil
	}
	if settings.TunnelProviderConfig != nil {
		tunnelProviderConfig = *settings.TunnelProviderConfig
	} else {
		tunnelProviderConfig = nil
	}
	_, err := db.Exec(
		"UPDATE settings SET cloudflare_api_token = ?, cloudflare_account_id = ?, auto_start_apps = ?, active_tunnel_provider = ?, tunnel_provider_config = ?, updated_at = ? WHERE id = ?",
		apiToken, accountID, settings.AutoStartApps, activeTunnelProvider, tunnelProviderConfig, time.Now(), settings.ID,
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
	var publicURL sql.NullString
	err := db.QueryRow(
		"SELECT id, app_id, tunnel_id, tunnel_name, tunnel_token, account_id, is_active, status, ingress_rules, public_url, created_at, updated_at, last_synced_at, error_details FROM cloudflare_tunnels WHERE app_id = ?",
		appID,
	).Scan(&tunnel.ID, &tunnel.AppID, &tunnel.TunnelID, &tunnel.TunnelName, &tunnel.TunnelToken, &tunnel.AccountID, &tunnel.IsActive, &tunnel.Status, &ingressRules, &publicURL, &tunnel.CreatedAt, &tunnel.UpdatedAt, &lastSyncedAt, &errorDetails)
	if err == nil && publicURL.Valid {
		tunnel.PublicURL = publicURL.String
	}

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
		"UPDATE cloudflare_tunnels SET tunnel_name = ?, is_active = ?, status = ?, ingress_rules = ?, public_url = ?, updated_at = ?, last_synced_at = ?, error_details = ? WHERE id = ?",
		tunnel.TunnelName, tunnel.IsActive, tunnel.Status, ingressRules, tunnel.PublicURL, time.Now(), tunnel.LastSyncedAt, errorDetails, tunnel.ID,
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
	var errorDetails, publicURL sql.NullString
	var lastSyncedAt, ingressRules interface{}
	err := db.QueryRow(
		"SELECT id, app_id, tunnel_id, tunnel_name, tunnel_token, account_id, is_active, status, ingress_rules, public_url, created_at, updated_at, last_synced_at, error_details FROM cloudflare_tunnels WHERE tunnel_id = ?",
		tunnelID,
	).Scan(&tunnel.ID, &tunnel.AppID, &tunnel.TunnelID, &tunnel.TunnelName, &tunnel.TunnelToken, &tunnel.AccountID, &tunnel.IsActive, &tunnel.Status, &ingressRules, &publicURL, &tunnel.CreatedAt, &tunnel.UpdatedAt, &lastSyncedAt, &errorDetails)
	if err == nil && publicURL.Valid {
		tunnel.PublicURL = publicURL.String
	}

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
	rows, err := db.Query("SELECT id, app_id, tunnel_id, tunnel_name, tunnel_token, account_id, is_active, status, ingress_rules, public_url, created_at, updated_at, last_synced_at, error_details FROM cloudflare_tunnels WHERE is_active = 1 ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tunnels []*CloudflareTunnel
	for rows.Next() {
		tunnel := &CloudflareTunnel{}
		var lastSyncedAt, ingressRules interface{}
		var errorDetails, publicURL sql.NullString
		err := rows.Scan(&tunnel.ID, &tunnel.AppID, &tunnel.TunnelID, &tunnel.TunnelName, &tunnel.TunnelToken, &tunnel.AccountID, &tunnel.IsActive, &tunnel.Status, &ingressRules, &publicURL, &tunnel.CreatedAt, &tunnel.UpdatedAt, &lastSyncedAt, &errorDetails)
		if err != nil {
			return nil, err
		}
		if publicURL.Valid {
			tunnel.PublicURL = publicURL.String
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

// CreateComposeVersion creates a new compose version record
func (db *DB) CreateComposeVersion(version *ComposeVersion) error {
	var changeReason, changedBy, rolledBackFrom interface{}
	if version.ChangeReason != nil {
		changeReason = *version.ChangeReason
	} else {
		changeReason = nil
	}
	if version.ChangedBy != nil {
		changedBy = *version.ChangedBy
	} else {
		changedBy = nil
	}
	if version.RolledBackFrom != nil {
		rolledBackFrom = *version.RolledBackFrom
	} else {
		rolledBackFrom = nil
	}

	_, err := db.Exec(
		"INSERT INTO compose_versions (id, app_id, version, compose_content, change_reason, changed_by, is_current, created_at, rolled_back_from) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		version.ID, version.AppID, version.Version, version.ComposeContent, changeReason, changedBy, version.IsCurrent, version.CreatedAt, rolledBackFrom,
	)
	return err
}

// GetComposeVersionsByAppID retrieves all compose versions for an app, ordered by version DESC
func (db *DB) GetComposeVersionsByAppID(appID string) ([]*ComposeVersion, error) {
	rows, err := db.Query("SELECT id, app_id, version, compose_content, change_reason, changed_by, is_current, created_at, rolled_back_from FROM compose_versions WHERE app_id = ? ORDER BY version DESC", appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []*ComposeVersion
	for rows.Next() {
		version := &ComposeVersion{}
		var changeReason, changedBy sql.NullString
		var rolledBackFrom sql.NullInt64
		err := rows.Scan(&version.ID, &version.AppID, &version.Version, &version.ComposeContent, &changeReason, &changedBy, &version.IsCurrent, &version.CreatedAt, &rolledBackFrom)
		if err != nil {
			return nil, err
		}

		if changeReason.Valid {
			version.ChangeReason = &changeReason.String
		}
		if changedBy.Valid {
			version.ChangedBy = &changedBy.String
		}
		if rolledBackFrom.Valid {
			rbf := int(rolledBackFrom.Int64)
			version.RolledBackFrom = &rbf
		}

		versions = append(versions, version)
	}

	return versions, nil
}

// GetComposeVersion retrieves a specific compose version by app ID and version number
func (db *DB) GetComposeVersion(appID string, version int) (*ComposeVersion, error) {
	v := &ComposeVersion{}
	var changeReason, changedBy sql.NullString
	var rolledBackFrom sql.NullInt64
	err := db.QueryRow(
		"SELECT id, app_id, version, compose_content, change_reason, changed_by, is_current, created_at, rolled_back_from FROM compose_versions WHERE app_id = ? AND version = ?",
		appID, version,
	).Scan(&v.ID, &v.AppID, &v.Version, &v.ComposeContent, &changeReason, &changedBy, &v.IsCurrent, &v.CreatedAt, &rolledBackFrom)

	if err == nil {
		if changeReason.Valid {
			v.ChangeReason = &changeReason.String
		}
		if changedBy.Valid {
			v.ChangedBy = &changedBy.String
		}
		if rolledBackFrom.Valid {
			rbf := int(rolledBackFrom.Int64)
			v.RolledBackFrom = &rbf
		}
	}
	return v, err
}

// GetCurrentComposeVersion retrieves the current active compose version for an app
func (db *DB) GetCurrentComposeVersion(appID string) (*ComposeVersion, error) {
	v := &ComposeVersion{}
	var changeReason, changedBy sql.NullString
	var rolledBackFrom sql.NullInt64
	err := db.QueryRow(
		"SELECT id, app_id, version, compose_content, change_reason, changed_by, is_current, created_at, rolled_back_from FROM compose_versions WHERE app_id = ? AND is_current = 1",
		appID,
	).Scan(&v.ID, &v.AppID, &v.Version, &v.ComposeContent, &changeReason, &changedBy, &v.IsCurrent, &v.CreatedAt, &rolledBackFrom)

	if err == nil {
		if changeReason.Valid {
			v.ChangeReason = &changeReason.String
		}
		if changedBy.Valid {
			v.ChangedBy = &changedBy.String
		}
		if rolledBackFrom.Valid {
			rbf := int(rolledBackFrom.Int64)
			v.RolledBackFrom = &rbf
		}
	}
	return v, err
}

// GetLatestVersionNumber retrieves the latest version number for an app
func (db *DB) GetLatestVersionNumber(appID string) (int, error) {
	var version sql.NullInt64
	err := db.QueryRow("SELECT MAX(version) FROM compose_versions WHERE app_id = ?", appID).Scan(&version)
	if err != nil {
		return 0, err
	}
	if !version.Valid {
		return 0, nil
	}
	return int(version.Int64), nil
}

// MarkAllVersionsAsNotCurrent marks all versions for an app as not current
func (db *DB) MarkAllVersionsAsNotCurrent(appID string) error {
	_, err := db.Exec("UPDATE compose_versions SET is_current = 0 WHERE app_id = ?", appID)
	return err
}

// MarkVersionAsCurrent marks a specific version as current and all others as not current
func (db *DB) MarkVersionAsCurrent(appID string, version int) error {
	// Start a transaction
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Mark all versions as not current
	if _, err := tx.Exec("UPDATE compose_versions SET is_current = 0 WHERE app_id = ?", appID); err != nil {
		return err
	}

	// Mark the specified version as current
	if _, err := tx.Exec("UPDATE compose_versions SET is_current = 1 WHERE app_id = ? AND version = ?", appID, version); err != nil {
		return err
	}

	return tx.Commit()
}

// DeleteComposeVersionsByAppID deletes all compose versions for an app
func (db *DB) DeleteComposeVersionsByAppID(appID string) error {
	_, err := db.Exec("DELETE FROM compose_versions WHERE app_id = ?", appID)
	return err
}

// ===========================
// Node CRUD Operations
// ===========================

// CreateNode creates a new node
func (db *DB) CreateNode(node *Node) error {
	_, err := db.Exec(
		`INSERT INTO nodes (id, name, api_endpoint, api_key, is_primary, status, last_seen, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		node.ID, node.Name, node.APIEndpoint, node.APIKey,
		node.IsPrimary, node.Status, node.LastSeen,
		node.CreatedAt, node.UpdatedAt,
	)
	return err
}

// GetNode retrieves a node by ID
func (db *DB) GetNode(id string) (*Node, error) {
	node := &Node{}
	var lastSeen sql.NullTime
	var lastHealthCheck sql.NullTime
	err := db.QueryRow(
		`SELECT id, name, api_endpoint, api_key, is_primary, status, last_seen, consecutive_failures, last_health_check, created_at, updated_at 
		 FROM nodes WHERE id = ?`,
		id,
	).Scan(&node.ID, &node.Name, &node.APIEndpoint, &node.APIKey,
		&node.IsPrimary, &node.Status, &lastSeen, &node.ConsecutiveFailures, &lastHealthCheck,
		&node.CreatedAt, &node.UpdatedAt)

	if err == nil {
		if lastSeen.Valid {
			node.LastSeen = &lastSeen.Time
		}
		if lastHealthCheck.Valid {
			node.LastHealthCheck = &lastHealthCheck.Time
		}
	}

	return node, err
}

// GetAllNodes retrieves all nodes
func (db *DB) GetAllNodes() ([]*Node, error) {
	rows, err := db.Query(
		`SELECT id, name, api_endpoint, api_key, is_primary, status, last_seen, consecutive_failures, last_health_check, created_at, updated_at 
		 FROM nodes ORDER BY created_at ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []*Node
	for rows.Next() {
		node := &Node{}
		var lastSeen sql.NullTime
		var lastHealthCheck sql.NullTime
		err := rows.Scan(&node.ID, &node.Name, &node.APIEndpoint, &node.APIKey,
			&node.IsPrimary, &node.Status, &lastSeen, &node.ConsecutiveFailures, &lastHealthCheck,
			&node.CreatedAt, &node.UpdatedAt)
		if err != nil {
			return nil, err
		}

		if lastSeen.Valid {
			node.LastSeen = &lastSeen.Time
		}
		if lastHealthCheck.Valid {
			node.LastHealthCheck = &lastHealthCheck.Time
		}

		nodes = append(nodes, node)
	}

	return nodes, nil
}

// GetPrimaryNode retrieves the primary node
func (db *DB) GetPrimaryNode() (*Node, error) {
	node := &Node{}
	var lastSeen sql.NullTime
	var lastHealthCheck sql.NullTime
	err := db.QueryRow(
		`SELECT id, name, api_endpoint, api_key, is_primary, status, last_seen, consecutive_failures, last_health_check, created_at, updated_at 
		 FROM nodes WHERE is_primary = 1 LIMIT 1`,
	).Scan(&node.ID, &node.Name, &node.APIEndpoint, &node.APIKey,
		&node.IsPrimary, &node.Status, &lastSeen, &node.ConsecutiveFailures, &lastHealthCheck,
		&node.CreatedAt, &node.UpdatedAt)

	if err == nil {
		if lastSeen.Valid {
			node.LastSeen = &lastSeen.Time
		}
		if lastHealthCheck.Valid {
			node.LastHealthCheck = &lastHealthCheck.Time
		}
	}

	return node, err
}

// UpdateNode updates a node
func (db *DB) UpdateNode(node *Node) error {
	_, err := db.Exec(
		`UPDATE nodes SET name = ?, api_endpoint = ?, api_key = ?, is_primary = ?, status = ?, last_seen = ?, consecutive_failures = ?, last_health_check = ?, updated_at = ? 
		 WHERE id = ?`,
		node.Name, node.APIEndpoint, node.APIKey, node.IsPrimary,
		node.Status, node.LastSeen, node.ConsecutiveFailures, node.LastHealthCheck, time.Now(), node.ID,
	)
	return err
}

// DeleteNode deletes a node
func (db *DB) DeleteNode(id string) error {
	_, err := db.Exec("DELETE FROM nodes WHERE id = ?", id)
	return err
}

// GetNodeByName retrieves a node by name
func (db *DB) GetNodeByName(name string) (*Node, error) {
	node := &Node{}
	var lastSeen sql.NullTime
	var lastHealthCheck sql.NullTime
	err := db.QueryRow(
		`SELECT id, name, api_endpoint, api_key, is_primary, status, last_seen, consecutive_failures, last_health_check, created_at, updated_at 
		 FROM nodes WHERE name = ?`,
		name,
	).Scan(&node.ID, &node.Name, &node.APIEndpoint, &node.APIKey,
		&node.IsPrimary, &node.Status, &lastSeen, &node.ConsecutiveFailures, &lastHealthCheck,
		&node.CreatedAt, &node.UpdatedAt)

	if err == nil {
		if lastSeen.Valid {
			node.LastSeen = &lastSeen.Time
		}
		if lastHealthCheck.Valid {
			node.LastHealthCheck = &lastHealthCheck.Time
		}
	}

	return node, err
}

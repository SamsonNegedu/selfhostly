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
		// Jobs table for background async operations
		// Drop existing table if it exists (for dev environments - allows schema updates)
		`DROP TABLE IF EXISTS jobs`,
		`CREATE TABLE IF NOT EXISTS jobs (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			app_id TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			payload TEXT,
			progress INTEGER NOT NULL DEFAULT 0,
			progress_message TEXT,
			result TEXT,
			error_message TEXT,
			started_at DATETIME,
			completed_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			
			-- Worker tracking for multi-worker support
			claimed_by TEXT,
			claimed_at DATETIME,
			
			-- Retry support
			retry_count INTEGER NOT NULL DEFAULT 0,
			max_retries INTEGER NOT NULL DEFAULT 0,
			retry_after DATETIME,
			
			-- Cancellation support
			cancelled_at DATETIME,
			
			-- Timeout in seconds
			timeout_seconds INTEGER,
			
			-- Deduplication hash
			job_hash TEXT,
			
			FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_jobs_app_id ON jobs(app_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status) WHERE status IN ('pending', 'running')`,
		`CREATE INDEX IF NOT EXISTS idx_jobs_app_status ON jobs(app_id, status) WHERE status IN ('pending', 'running')`,
		`CREATE INDEX IF NOT EXISTS idx_jobs_hash ON jobs(job_hash, status) WHERE job_hash IS NOT NULL`,
		`CREATE INDEX IF NOT EXISTS idx_jobs_claimed ON jobs(claimed_by, status) WHERE claimed_by IS NOT NULL`,
	}

	// Run migrations
	for _, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			// Ignore duplicate column errors for ALTER TABLE ADD COLUMN statements
			// This allows migrations to be idempotent
			if isDuplicateColumnError(err) {
				slog.Debug("Skipping migration - column already exists", "error", err)
				continue
			}
			return err
		}
	}

	// One-time migration: Check if jobs table needs to be recreated with new columns
	return db.migrateJobsTableIfNeeded()
}

// isDuplicateColumnError checks if error is about duplicate column
func isDuplicateColumnError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "duplicate column name") ||
		strings.Contains(errStr, "duplicate column") ||
		strings.Contains(errStr, "already exists")
}

// migrateJobsTableIfNeeded checks if the jobs table has the new columns and recreates it if needed
// This is a one-time migration that will only run if the table is missing the new columns
func (db *DB) migrateJobsTableIfNeeded() error {
	// Check if job_hash column exists (using it as a marker for the new schema)
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('jobs') WHERE name = 'job_hash'
	`).Scan(&count)
	
	if err != nil {
		// Table might not exist yet, which is fine - CREATE TABLE IF NOT EXISTS handles it
		return nil
	}
	
	// If column exists, migration already done
	if count > 0 {
		return nil
	}
	
	// Table exists but missing new columns - need to recreate
	slog.Info("Jobs table missing new columns, recreating table...")
	
	// Drop old table
	if _, err := db.Exec(`DROP TABLE IF EXISTS jobs`); err != nil {
		return fmt.Errorf("failed to drop old jobs table: %w", err)
	}
	
	// Recreate with new schema
	recreateSQL := `CREATE TABLE jobs (
		id TEXT PRIMARY KEY,
		type TEXT NOT NULL,
		app_id TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending',
		payload TEXT,
		progress INTEGER NOT NULL DEFAULT 0,
		progress_message TEXT,
		result TEXT,
		error_message TEXT,
		started_at DATETIME,
		completed_at DATETIME,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		
		-- Worker tracking for multi-worker support
		claimed_by TEXT,
		claimed_at DATETIME,
		
		-- Retry support
		retry_count INTEGER NOT NULL DEFAULT 0,
		max_retries INTEGER NOT NULL DEFAULT 0,
		retry_after DATETIME,
		
		-- Cancellation support
		cancelled_at DATETIME,
		
		-- Timeout in seconds
		timeout_seconds INTEGER,
		
		-- Deduplication hash
		job_hash TEXT,
		
		FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE
	)`
	
	if _, err := db.Exec(recreateSQL); err != nil {
		return fmt.Errorf("failed to recreate jobs table: %w", err)
	}
	
	// Recreate indexes
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_jobs_app_id ON jobs(app_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status) WHERE status IN ('pending', 'running')`,
		`CREATE INDEX IF NOT EXISTS idx_jobs_app_status ON jobs(app_id, status) WHERE status IN ('pending', 'running')`,
		`CREATE INDEX IF NOT EXISTS idx_jobs_hash ON jobs(job_hash, status) WHERE job_hash IS NOT NULL`,
		`CREATE INDEX IF NOT EXISTS idx_jobs_claimed ON jobs(claimed_by, status) WHERE claimed_by IS NOT NULL`,
	}
	
	for _, indexSQL := range indexes {
		if _, err := db.Exec(indexSQL); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}
	
	slog.Info("Jobs table recreated successfully with new schema")
	return nil
}

// GetProviderConfig parses the tunnel_provider_config JSON and returns configuration
// for the specified provider.
func (settings *Settings) GetProviderConfig(providerName string) (map[string]interface{}, error) {
	if settings.TunnelProviderConfig == nil || *settings.TunnelProviderConfig == "" {
		return nil, fmt.Errorf("provider %s not configured", providerName)
	}

	var providerConfigs map[string]interface{}
	if err := json.Unmarshal([]byte(*settings.TunnelProviderConfig), &providerConfigs); err != nil {
		return nil, fmt.Errorf("failed to parse provider config: %w", err)
	}

	if config, ok := providerConfigs[providerName]; ok {
		if configMap, ok := config.(map[string]interface{}); ok {
			return configMap, nil
		}
	}

	return nil, fmt.Errorf("provider %s not configured", providerName)
}

// GetActiveProviderName returns the active tunnel provider name.
func (settings *Settings) GetActiveProviderName() string {
	if settings.ActiveTunnelProvider != nil && *settings.ActiveTunnelProvider != "" {
		return *settings.ActiveTunnelProvider
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
	var apiToken, accountID, activeTunnelProvider, tunnelProviderConfig sql.NullString
	err := db.QueryRow(
		"SELECT id, cloudflare_api_token, cloudflare_account_id, auto_start_apps, active_tunnel_provider, tunnel_provider_config, updated_at FROM settings LIMIT 1",
	).Scan(&settings.ID, &apiToken, &accountID, &settings.AutoStartApps, &activeTunnelProvider, &tunnelProviderConfig, &settings.UpdatedAt)

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

	// Convert sql.NullString to *string
	if apiToken.Valid {
		settings.CloudflareAPIToken = &apiToken.String
	}
	if accountID.Valid {
		settings.CloudflareAccountID = &accountID.String
	}
	if activeTunnelProvider.Valid {
		settings.ActiveTunnelProvider = &activeTunnelProvider.String
	}
	if tunnelProviderConfig.Valid {
		settings.TunnelProviderConfig = &tunnelProviderConfig.String
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

// ============================================================================
// Job Operations
// ============================================================================

// scanJob scans a job row from the database into a Job struct
func scanJob(rows *sql.Rows) (*Job, error) {
	job := &Job{}
	var payload, progressMessage, result, errorMessage, claimedBy, jobHash sql.NullString
	var startedAt, completedAt, claimedAt, retryAfter, cancelledAt sql.NullTime
	var timeoutSeconds sql.NullInt64

	var err error
	if rows != nil {
		err = rows.Scan(
			&job.ID, &job.Type, &job.AppID, &job.Status, &payload, &job.Progress, &progressMessage,
			&result, &errorMessage, &startedAt, &completedAt, &job.CreatedAt, &job.UpdatedAt,
			&claimedBy, &claimedAt, &job.RetryCount, &job.MaxRetries, &retryAfter,
			&cancelledAt, &timeoutSeconds, &jobHash,
		)
	} else {
		return nil, fmt.Errorf("rows is nil")
	}

	if err != nil {
		return nil, err
	}

	// Handle nullable fields
	if payload.Valid {
		job.Payload = &payload.String
	}
	if progressMessage.Valid {
		job.ProgressMessage = &progressMessage.String
	}
	if result.Valid {
		job.Result = &result.String
	}
	if errorMessage.Valid {
		job.ErrorMessage = &errorMessage.String
	}
	if startedAt.Valid {
		job.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		job.CompletedAt = &completedAt.Time
	}
	if claimedBy.Valid {
		job.ClaimedBy = &claimedBy.String
	}
	if claimedAt.Valid {
		job.ClaimedAt = &claimedAt.Time
	}
	if retryAfter.Valid {
		job.RetryAfter = &retryAfter.Time
	}
	if cancelledAt.Valid {
		job.CancelledAt = &cancelledAt.Time
	}
	if timeoutSeconds.Valid {
		val := int(timeoutSeconds.Int64)
		job.TimeoutSeconds = &val
	}
	if jobHash.Valid {
		job.JobHash = &jobHash.String
	}

	return job, nil
}

// scanJobFromRow scans a job from a QueryRow result
func scanJobFromRow(row *sql.Row) (*Job, error) {
	job := &Job{}
	var payload, progressMessage, result, errorMessage, claimedBy, jobHash sql.NullString
	var startedAt, completedAt, claimedAt, retryAfter, cancelledAt sql.NullTime
	var timeoutSeconds sql.NullInt64

	err := row.Scan(
		&job.ID, &job.Type, &job.AppID, &job.Status, &payload, &job.Progress, &progressMessage,
		&result, &errorMessage, &startedAt, &completedAt, &job.CreatedAt, &job.UpdatedAt,
		&claimedBy, &claimedAt, &job.RetryCount, &job.MaxRetries, &retryAfter,
		&cancelledAt, &timeoutSeconds, &jobHash,
	)

	if err != nil {
		return nil, err
	}

	// Handle nullable fields
	if payload.Valid {
		job.Payload = &payload.String
	}
	if progressMessage.Valid {
		job.ProgressMessage = &progressMessage.String
	}
	if result.Valid {
		job.Result = &result.String
	}
	if errorMessage.Valid {
		job.ErrorMessage = &errorMessage.String
	}
	if startedAt.Valid {
		job.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		job.CompletedAt = &completedAt.Time
	}
	if claimedBy.Valid {
		job.ClaimedBy = &claimedBy.String
	}
	if claimedAt.Valid {
		job.ClaimedAt = &claimedAt.Time
	}
	if retryAfter.Valid {
		job.RetryAfter = &retryAfter.Time
	}
	if cancelledAt.Valid {
		job.CancelledAt = &cancelledAt.Time
	}
	if timeoutSeconds.Valid {
		val := int(timeoutSeconds.Int64)
		job.TimeoutSeconds = &val
	}
	if jobHash.Valid {
		job.JobHash = &jobHash.String
	}

	return job, nil
}

// CreateJob creates a new job
func (db *DB) CreateJob(job *Job) error {
	_, err := db.Exec(
		`INSERT INTO jobs (id, type, app_id, status, payload, progress, progress_message, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		job.ID, job.Type, job.AppID, job.Status, job.Payload, job.Progress, job.ProgressMessage,
		job.CreatedAt, job.UpdatedAt,
	)
	return err
}

// GetJob retrieves a job by ID
func (db *DB) GetJob(id string) (*Job, error) {
	row := db.QueryRow(
		`SELECT id, type, app_id, status, payload, progress, progress_message, result, error_message,
		        started_at, completed_at, created_at, updated_at,
		        claimed_by, claimed_at, retry_count, max_retries, retry_after,
		        cancelled_at, timeout_seconds, job_hash
		 FROM jobs WHERE id = ?`,
		id,
	)
	return scanJobFromRow(row)
}

// GetJobsByAppID retrieves jobs for a specific app, ordered by creation date (newest first)
func (db *DB) GetJobsByAppID(appID string, limit int) ([]*Job, error) {
	rows, err := db.Query(
		`SELECT id, type, app_id, status, payload, progress, progress_message, result, error_message,
		        started_at, completed_at, created_at, updated_at,
		        claimed_by, claimed_at, retry_count, max_retries, retry_after,
		        cancelled_at, timeout_seconds, job_hash
		 FROM jobs
		 WHERE app_id = ?
		 ORDER BY created_at DESC
		 LIMIT ?`,
		appID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*Job
	for rows.Next() {
		job, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}

	return jobs, nil
}

// GetActiveJobForApp retrieves any pending or running job for an app (for concurrency check)
func (db *DB) GetActiveJobForApp(appID string) (*Job, error) {
	row := db.QueryRow(
		`SELECT id, type, app_id, status, payload, progress, progress_message, result, error_message,
		        started_at, completed_at, created_at, updated_at,
		        claimed_by, claimed_at, retry_count, max_retries, retry_after,
		        cancelled_at, timeout_seconds, job_hash
		 FROM jobs
		 WHERE app_id = ? AND status IN (?, ?)
		 ORDER BY created_at DESC
		 LIMIT 1`,
		appID, constants.JobStatusPending, constants.JobStatusRunning,
	)
	
	job, err := scanJobFromRow(row)
	if err == sql.ErrNoRows {
		return nil, nil // No active job found
	}
	return job, err
}

// UpdateJobStatus updates a job's status and progress
func (db *DB) UpdateJobStatus(id, status string, progress int, message *string) error {
	now := time.Now()

	// If transitioning to running, set started_at
	if status == constants.JobStatusRunning {
		_, err := db.Exec(
			`UPDATE jobs
			 SET status = ?, progress = ?, progress_message = ?, started_at = COALESCE(started_at, ?), updated_at = ?
			 WHERE id = ?`,
			status, progress, message, now, now, id,
		)
		return err
	}

	_, err := db.Exec(
		`UPDATE jobs
		 SET status = ?, progress = ?, progress_message = ?, updated_at = ?
		 WHERE id = ?`,
		status, progress, message, now, id,
	)
	return err
}

// UpdateJobCompleted marks a job as completed or failed
func (db *DB) UpdateJobCompleted(id, status string, result *string, errorMsg *string) error {
	now := time.Now()
	progress := 100
	if status == constants.JobStatusFailed {
		// Keep current progress on failure
		var currentProgress int
		err := db.QueryRow(`SELECT progress FROM jobs WHERE id = ?`, id).Scan(&currentProgress)
		if err == nil {
			progress = currentProgress
		}
	}

	_, err := db.Exec(
		`UPDATE jobs
		 SET status = ?, progress = ?, result = ?, error_message = ?, completed_at = ?, updated_at = ?
		 WHERE id = ?`,
		status, progress, result, errorMsg, now, now, id,
	)
	return err
}

// GetPendingJobs retrieves pending jobs, ordered by creation date (oldest first)
func (db *DB) GetPendingJobs(limit int) ([]*Job, error) {
	rows, err := db.Query(
		`SELECT id, type, app_id, status, payload, progress, progress_message, result, error_message,
		        started_at, completed_at, created_at, updated_at,
		        claimed_by, claimed_at, retry_count, max_retries, retry_after,
		        cancelled_at, timeout_seconds, job_hash
		 FROM jobs
		 WHERE status = ?
		 ORDER BY created_at ASC
		 LIMIT ?`,
		constants.JobStatusPending, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*Job
	for rows.Next() {
		job, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}

	return jobs, nil
}

// ClaimPendingJob atomically claims a pending job for a worker
// This prevents race conditions where multiple workers claim the same job
func (db *DB) ClaimPendingJob(workerID string) (*Job, error) {
	now := time.Now()
	
	// SQLite doesn't support UPDATE ... RETURNING, so we use a transaction-based approach:
	// 1. Find a pending job that isn't claimed
	// 2. Try to claim it atomically
	// 3. Return the claimed job
	
	// Use a transaction to ensure atomicity
	tx, err := db.BeginTx(context.Background())
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	
	// Find a pending job that isn't claimed
	var jobID string
	err = tx.QueryRow(
		`SELECT id FROM jobs
		 WHERE status = ? AND (claimed_by IS NULL OR claimed_by = '')
		 ORDER BY created_at ASC
		 LIMIT 1`,
		constants.JobStatusPending,
	).Scan(&jobID)
	
	if err == sql.ErrNoRows {
		return nil, nil // No job available
	}
	if err != nil {
		return nil, err
	}
	
	// Try to claim it atomically
	result, err := tx.Exec(
		`UPDATE jobs
		 SET status = ?, claimed_by = ?, claimed_at = ?, started_at = COALESCE(started_at, ?), updated_at = ?
		 WHERE id = ? AND status = ? AND (claimed_by IS NULL OR claimed_by = '')`,
		constants.JobStatusRunning, workerID, now, now, now,
		jobID, constants.JobStatusPending,
	)
	if err != nil {
		return nil, err
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	
	if rowsAffected == 0 {
		// Another worker claimed it first, return nil
		return nil, nil
	}
	
	// Retrieve the claimed job
	job := &Job{}
	var payload, progressMessage, resultStr, errorMessage, claimedBy, jobHash sql.NullString
	var startedAt, completedAt, claimedAt, retryAfter, cancelledAt sql.NullTime
	var timeoutSeconds sql.NullInt64
	
	err = tx.QueryRow(
		`SELECT id, type, app_id, status, payload, progress, progress_message, result, error_message,
		        started_at, completed_at, created_at, updated_at,
		        claimed_by, claimed_at, retry_count, max_retries, retry_after,
		        cancelled_at, timeout_seconds, job_hash
		 FROM jobs
		 WHERE id = ?`,
		jobID,
	).Scan(
		&job.ID, &job.Type, &job.AppID, &job.Status, &payload, &job.Progress, &progressMessage,
		&resultStr, &errorMessage, &startedAt, &completedAt, &job.CreatedAt, &job.UpdatedAt,
		&claimedBy, &claimedAt, &job.RetryCount, &job.MaxRetries, &retryAfter,
		&cancelledAt, &timeoutSeconds, &jobHash,
	)
	if err != nil {
		return nil, err
	}
	
	// Handle nullable fields
	if payload.Valid {
		job.Payload = &payload.String
	}
	if progressMessage.Valid {
		job.ProgressMessage = &progressMessage.String
	}
	if resultStr.Valid {
		job.Result = &resultStr.String
	}
	if errorMessage.Valid {
		job.ErrorMessage = &errorMessage.String
	}
	if startedAt.Valid {
		job.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		job.CompletedAt = &completedAt.Time
	}
	if claimedBy.Valid {
		job.ClaimedBy = &claimedBy.String
	}
	if claimedAt.Valid {
		job.ClaimedAt = &claimedAt.Time
	}
	if retryAfter.Valid {
		job.RetryAfter = &retryAfter.Time
	}
	if cancelledAt.Valid {
		job.CancelledAt = &cancelledAt.Time
	}
	if timeoutSeconds.Valid {
		val := int(timeoutSeconds.Int64)
		job.TimeoutSeconds = &val
	}
	if jobHash.Valid {
		job.JobHash = &jobHash.String
	}
	
	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	
	return job, nil
}

// ReleaseJobClaim releases a job claim (e.g., if worker crashes)
func (db *DB) ReleaseJobClaim(jobID string) error {
	_, err := db.Exec(
		`UPDATE jobs
		 SET claimed_by = NULL, claimed_at = NULL, updated_at = ?
		 WHERE id = ?`,
		time.Now(), jobID,
	)
	return err
}

// CancelJob marks a job as cancelled
func (db *DB) CancelJob(jobID string) error {
	now := time.Now()
	_, err := db.Exec(
		`UPDATE jobs
		 SET cancelled_at = ?, updated_at = ?
		 WHERE id = ? AND status IN (?, ?)`,
		now, now, jobID, constants.JobStatusPending, constants.JobStatusRunning,
	)
	return err
}

// IsJobCancelled checks if a job has been cancelled
func (db *DB) IsJobCancelled(jobID string) (bool, error) {
	var cancelledAt sql.NullTime
	err := db.QueryRow(
		`SELECT cancelled_at FROM jobs WHERE id = ?`,
		jobID,
	).Scan(&cancelledAt)
	if err != nil {
		return false, err
	}
	return cancelledAt.Valid, nil
}

// MarkStaleJobsAsFailed marks jobs that have been in "running" state for too long as failed
// This handles recovery from crashes/restarts
func (db *DB) MarkStaleJobsAsFailed(staleThreshold time.Duration) error {
	cutoffTime := time.Now().Add(-staleThreshold)
	errorMsg := fmt.Sprintf("Job marked as failed due to stale state (no updates for %v)", staleThreshold)

	result, err := db.Exec(
		`UPDATE jobs
		 SET status = ?, error_message = ?, completed_at = ?, updated_at = ?
		 WHERE status = ? AND updated_at < ?`,
		constants.JobStatusFailed, errorMsg, time.Now(), time.Now(),
		constants.JobStatusRunning, cutoffTime,
	)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		slog.Info("Marked stale jobs as failed", "count", rowsAffected, "threshold", staleThreshold)
	}

	return nil
}

// CleanupOldCompletedJobs deletes old completed/failed jobs for an app, keeping only the most recent N
func (db *DB) CleanupOldCompletedJobs(appID string, keepCount int) error {
	// Delete all but the most recent N completed/failed jobs for this app
	_, err := db.Exec(
		`DELETE FROM jobs
		 WHERE app_id = ?
		 AND status IN (?, ?)
		 AND id NOT IN (
		     SELECT id FROM jobs
		     WHERE app_id = ? AND status IN (?, ?)
		     ORDER BY created_at DESC
		     LIMIT ?
		 )`,
		appID, constants.JobStatusCompleted, constants.JobStatusFailed,
		appID, constants.JobStatusCompleted, constants.JobStatusFailed, keepCount,
	)
	return err
}

// CleanupAllOldCompletedJobs deletes old completed/failed jobs for all apps in a single query
// This is more efficient than calling CleanupOldCompletedJobs for each app
func (db *DB) CleanupAllOldCompletedJobs(keepCount int) error {
	// For each app, keep only the most recent N completed/failed jobs
	_, err := db.Exec(
		`DELETE FROM jobs
		 WHERE status IN (?, ?)
		 AND id NOT IN (
		     SELECT id FROM (
		         SELECT id, app_id,
		                ROW_NUMBER() OVER (PARTITION BY app_id ORDER BY created_at DESC) as rn
		         FROM jobs
		         WHERE status IN (?, ?)
		     ) ranked
		     WHERE rn <= ?
		 )`,
		constants.JobStatusCompleted, constants.JobStatusFailed,
		constants.JobStatusCompleted, constants.JobStatusFailed, keepCount,
	)
	return err
}

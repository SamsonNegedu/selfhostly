package cloudflare

import (
	"log/slog"

	"github.com/selfhostly/internal/db"
	"github.com/selfhostly/internal/domain"
)

// SettingsManager manages Cloudflare settings and configuration
type SettingsManager struct {
	database *db.DB
	logger   *slog.Logger
}

// NewSettingsManager creates a new settings manager
func NewSettingsManager(database *db.DB, logger *slog.Logger) *SettingsManager {
	return &SettingsManager{
		database: database,
		logger:   logger,
	}
}

// GetConfiguredTunnelManager returns a tunnel manager if Cloudflare is configured
// Returns nil if Cloudflare is not configured (without error)
func (sm *SettingsManager) GetConfiguredTunnelManager() (*TunnelManager, *db.Settings, error) {
	settings, err := sm.database.GetSettings()
	if err != nil {
		sm.logger.Error("failed to get settings", "error", err)
		return nil, nil, domain.WrapDatabaseOperation("get settings", err)
	}

	if !sm.isConfigured(settings) {
		return nil, settings, nil
	}

	tunnelManager := NewTunnelManager(*settings.CloudflareAPIToken, *settings.CloudflareAccountID, sm.database)
	return tunnelManager, settings, nil
}

// IsCloudflareConfigured checks if Cloudflare credentials are set
func (sm *SettingsManager) IsCloudflareConfigured() (bool, error) {
	settings, err := sm.database.GetSettings()
	if err != nil {
		return false, domain.WrapDatabaseOperation("get settings", err)
	}

	return sm.isConfigured(settings), nil
}

// RequireCloudflareSettings returns settings or error if not configured
func (sm *SettingsManager) RequireCloudflareSettings() (*db.Settings, error) {
	settings, err := sm.database.GetSettings()
	if err != nil {
		sm.logger.Error("failed to get settings", "error", err)
		return nil, domain.WrapDatabaseOperation("get settings", err)
	}

	if !sm.isConfigured(settings) {
		return nil, domain.ErrTunnelNotConfigured
	}

	return settings, nil
}

// GetSettings returns the current settings (may or may not be configured)
func (sm *SettingsManager) GetSettings() (*db.Settings, error) {
	settings, err := sm.database.GetSettings()
	if err != nil {
		sm.logger.Error("failed to get settings", "error", err)
		return nil, domain.WrapDatabaseOperation("get settings", err)
	}

	return settings, nil
}

// isConfigured checks if Cloudflare credentials are properly set
func (sm *SettingsManager) isConfigured(settings *db.Settings) bool {
	return settings.CloudflareAPIToken != nil &&
		settings.CloudflareAccountID != nil &&
		*settings.CloudflareAPIToken != "" &&
		*settings.CloudflareAccountID != ""
}

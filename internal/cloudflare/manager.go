package cloudflare

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/selfhostly/internal/db"
)

// TunnelManager handles Cloudflare tunnel management operations
type TunnelManager struct {
	ApiManager *Manager
	database   *db.DB
}

// NewTunnelManager creates a new tunnel manager
func NewTunnelManager(apiToken, accountID string, database *db.DB) *TunnelManager {
	return &TunnelManager{
		ApiManager: NewManager(apiToken, accountID),
		database:   database,
	}
}

// NewTunnelManagerWithManager creates a new tunnel manager with a custom API manager (for testing)
func NewTunnelManagerWithManager(apiManager *Manager, database *db.DB) *TunnelManager {
	return &TunnelManager{
		ApiManager: apiManager,
		database:   database,
	}
}

// CreateTunnelWithMetadata creates a tunnel and stores its metadata
func (tm *TunnelManager) CreateTunnelWithMetadata(appName string, appID string) (*db.CloudflareTunnel, error) {
	// Create tunnel via API
	tunnelID, token, err := tm.ApiManager.CreateTunnel(appName)
	if err != nil {
		return nil, fmt.Errorf("failed to create tunnel: %w", err)
	}

	// Create metadata record
	var errorDetails *string
	tunnel := &db.CloudflareTunnel{
		AppID:        appID,
		TunnelID:     tunnelID,
		TunnelName:   appName,
		TunnelToken:  token,
		AccountID:    tm.ApiManager.config.AccountID,
		IsActive:     true,
		Status:       "active",
		PublicURL:    "", // Set when ingress is configured
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		LastSyncedAt: &[]time.Time{time.Now()}[0],
		ErrorDetails: errorDetails,
	}

	// Store in database
	if err := tm.database.CreateCloudflareTunnel(tunnel); err != nil {
		// Try to clean up the tunnel from API if database insertion fails
		_ = tm.ApiManager.DeleteTunnel(tunnelID)
		return nil, fmt.Errorf("failed to store tunnel metadata: %w", err)
	}

	return tunnel, nil
}

// UpdateTunnelStatus updates the status of a tunnel
func (tm *TunnelManager) UpdateTunnelStatus(tunnelID string, status string, errorDetails string) error {
	// Get the tunnel record first
	tunnel, err := tm.database.GetCloudflareTunnelByTunnelID(tunnelID)
	if err != nil {
		return fmt.Errorf("failed to get tunnel: %w", err)
	}

	// Update the tunnel status
	tunnel.Status = status
	var errorDetailsPtr *string
	if errorDetails != "" {
		errorDetailsPtr = &errorDetails
	}
	tunnel.ErrorDetails = errorDetailsPtr
	tunnel.UpdatedAt = time.Now()

	if err := tm.database.UpdateCloudflareTunnel(tunnel); err != nil {
		return fmt.Errorf("failed to update tunnel status: %w", err)
	}

	return nil
}

// DeleteTunnelByAppID deletes a tunnel by app ID
func (tm *TunnelManager) DeleteTunnelByAppID(appID string) error {
	// Get the tunnel record
	tunnel, err := tm.database.GetCloudflareTunnelByAppID(appID)
	if err != nil {
		return fmt.Errorf("failed to get tunnel: %w", err)
	}

	// Delete from API
	if err := tm.ApiManager.DeleteTunnel(tunnel.TunnelID); err != nil {
		slog.Warn("failed to delete tunnel from API", "tunnel_id", tunnel.TunnelID, "error", err)
	}

	// Mark as inactive in database
	tunnel.IsActive = false
	tunnel.Status = "deleted"
	var emptyError *string
	tunnel.ErrorDetails = emptyError
	tunnel.UpdatedAt = time.Now()

	if err := tm.database.UpdateCloudflareTunnel(tunnel); err != nil {
		return fmt.Errorf("failed to update tunnel status: %w", err)
	}

	return nil
}

// GetAllActiveTunnels retrieves all active tunnels
func (tm *TunnelManager) GetAllActiveTunnels() ([]*db.CloudflareTunnel, error) {
	return tm.database.ListActiveCloudflareTunnels()
}

// mapCloudflareStatusToInternal maps Cloudflare API tunnel status to our internal status.
// Cloudflare returns: healthy, degraded, inactive, down (see Cloudflare API docs).
// We store: active, inactive, error so the UI can show Healthy / Inactive / Error.
func mapCloudflareStatusToInternal(apiStatus string) string {
	switch apiStatus {
	case "healthy", "degraded":
		return "active"
	case "inactive", "down":
		return "inactive"
	default:
		return apiStatus
	}
}

// SyncTunnelStatus synchronizes tunnel status with Cloudflare API and persists it in the database.
func (tm *TunnelManager) SyncTunnelStatus(tunnelID string) error {
	apiStatus, err := tm.ApiManager.GetTunnelStatus(tunnelID)
	if err != nil {
		// Tunnel may have been deleted on Cloudflare; update DB to reflect failure/unknown state
		tunnel, dbErr := tm.database.GetCloudflareTunnelByTunnelID(tunnelID)
		if dbErr == nil {
			tunnel.Status = "error"
			errMsg := err.Error()
			tunnel.ErrorDetails = &errMsg
			now := time.Now()
			tunnel.LastSyncedAt = &now
			tunnel.UpdatedAt = now
			_ = tm.database.UpdateCloudflareTunnel(tunnel)
		}
		return fmt.Errorf("failed to get tunnel status from Cloudflare: %w", err)
	}

	tunnel, err := tm.database.GetCloudflareTunnelByTunnelID(tunnelID)
	if err != nil {
		return fmt.Errorf("failed to get tunnel from database: %w", err)
	}

	now := time.Now()
	tunnel.Status = mapCloudflareStatusToInternal(apiStatus)
	tunnel.LastSyncedAt = &now
	tunnel.UpdatedAt = now
	tunnel.ErrorDetails = nil // Clear error on successful sync

	if err := tm.database.UpdateCloudflareTunnel(tunnel); err != nil {
		return fmt.Errorf("failed to update tunnel status: %w", err)
	}

	slog.Info("tunnel status synced", "tunnel_id", tunnelID, "api_status", apiStatus, "status", tunnel.Status, "synced_at", now.Format(time.RFC3339))
	return nil
}

// UpdateTunnelIngress updates the ingress configuration for a tunnel
func (tm *TunnelManager) UpdateTunnelIngress(tunnelID string, ingressRules []IngressRule, hostname string, targetDomain string) error {
	// First, validate that the tunnel exists and get the app ID
	tunnel, err := tm.database.GetCloudflareTunnelByTunnelID(tunnelID)
	if err != nil {
		return fmt.Errorf("failed to validate tunnel: %w", err)
	}

	// Update the ingress configuration via API
	err = tm.ApiManager.CreateIngressConfiguration(tunnelID, ingressRules)
	if err != nil {
		return err // API error already has full context
	}

	// Determine the primary hostname from either the provided hostname or the ingress rules
	primaryHostname := hostname
	if primaryHostname == "" {
		// Extract first hostname from ingress rules if available
		for _, rule := range ingressRules {
			if rule.Hostname != "" {
				primaryHostname = rule.Hostname
				break
			}
		}
	}

	// Update the app record with the tunnel domain and public URL
	app, err := tm.database.GetApp(tunnel.AppID)
	if err != nil {
		slog.Warn("Failed to get app for tunnel domain update", "appID", tunnel.AppID, "error", err)
	} else {
		// Update tunnel domain and public URL
		if primaryHostname != "" {
			app.TunnelDomain = primaryHostname
			app.PublicURL = fmt.Sprintf("https://%s", primaryHostname)
		} else {
			// Use default tunnel URL if no custom hostname
			app.TunnelDomain = fmt.Sprintf("%s.cfargotunnel.com", tunnelID)
			app.PublicURL = fmt.Sprintf("https://%s.cfargotunnel.com", tunnelID)
		}
		app.UpdatedAt = time.Now()

		if err := tm.database.UpdateApp(app); err != nil {
			slog.Warn("Failed to update app with tunnel domain", "appID", tunnel.AppID, "error", err)
		} else {
			slog.Info("Updated app with tunnel domain",
				"appID", tunnel.AppID,
				"tunnelDomain", app.TunnelDomain,
				"publicURL", app.PublicURL)
		}
	}

	// If hostname is provided, create a DNS record
	if primaryHostname != "" {
		// Extract domain from hostname (remove subdomain)
		domain := primaryHostname
		if strings.Contains(primaryHostname, ".") {
			parts := strings.Split(primaryHostname, ".")
			if len(parts) > 1 {
				domain = strings.Join(parts[len(parts)-2:], ".")
			}
		}

		// Get zone ID for the domain
		zoneID, err := tm.ApiManager.GetZoneID(domain)
		if err != nil {
			return fmt.Errorf("failed to get zone ID for domain %s: %w", domain, err)
		}

		// Create DNS record
		recordID, err := tm.ApiManager.CreateDNSRecord(zoneID, primaryHostname, tunnelID)
		if err != nil {
			return fmt.Errorf("failed to create DNS record: %w", err)
		}

		slog.Info("DNS record created successfully",
			"zoneID", zoneID,
			"hostname", primaryHostname,
			"targetDomain", targetDomain,
			"recordID", recordID)
	}

	return nil
}

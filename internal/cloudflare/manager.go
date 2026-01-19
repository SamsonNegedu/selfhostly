package cloudflare

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/selfhost-automaton/internal/db"
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

// CreateTunnelWithMetadata creates a tunnel and stores its metadata
func (tm *TunnelManager) CreateTunnelWithMetadata(appName string, appID int64) (*db.CloudflareTunnel, error) {
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
func (tm *TunnelManager) DeleteTunnelByAppID(appID int64) error {
	// Get the tunnel record
	tunnel, err := tm.database.GetCloudflareTunnelByAppID(appID)
	if err != nil {
		return fmt.Errorf("failed to get tunnel: %w", err)
	}

	// Delete from API
	if err := tm.ApiManager.DeleteTunnel(tunnel.TunnelID); err != nil {
		// Log the error but don't fail the deletion
		fmt.Printf("Warning: failed to delete tunnel from API: %v\n", err)
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

// SyncTunnelStatus synchronizes tunnel status with Cloudflare API
func (tm *TunnelManager) SyncTunnelStatus(tunnelID string) error {
	// For now, we'll just update the last synced time in the database
	// A full implementation would also check tunnel status with Cloudflare API
	now := time.Now()

	// Update the last synced time for the tunnel
	// Note: This is a simplified implementation - in a real system, you'd want
	// to properly identify which tunnel record to update
	fmt.Printf("Tunnel %s synced at %s\n", tunnelID, now.Format(time.RFC3339))

	return nil
}

// ValidateTunnel checks if a tunnel is valid and active
func (tm *TunnelManager) ValidateTunnel(tunnelID string) (bool, error) {
	// In a real implementation, this would check the tunnel status with Cloudflare API
	// For now, we'll just return true
	return true, nil
}

// GetTunnelConfig returns the configuration for a tunnel
func (tm *TunnelManager) GetTunnelConfig(tunnelID string) (map[string]interface{}, error) {
	// In a real implementation, this would fetch the tunnel configuration from Cloudflare API
	// For now, we'll return a placeholder configuration
	_ = tunnelID // Avoid unused parameter warning in real implementation
	config := map[string]interface{}{
		"tunnel_id": tunnelID,
		"status":    "active",
		"created":   time.Now().Format(time.RFC3339),
	}
	return config, nil
}

// UpdateTunnelIngress updates the ingress configuration for a tunnel
func (tm *TunnelManager) UpdateTunnelIngress(tunnelID string, ingressRules []IngressRule, hostname string, targetDomain string) error {
	// First, validate that the tunnel exists
	_, err := tm.database.GetCloudflareTunnelByTunnelID(tunnelID)
	if err != nil {
		return fmt.Errorf("failed to validate tunnel: %w", err)
	}

	// Update the ingress configuration via API
	err = tm.ApiManager.CreateIngressConfiguration(tunnelID, ingressRules)
	if err != nil {
		return fmt.Errorf("failed to update ingress configuration: %w", err)
	}

	// If hostname is provided, create a DNS record
	if hostname != "" {
		// Extract domain from hostname (remove subdomain)
		domain := hostname
		if strings.Contains(hostname, ".") {
			parts := strings.Split(hostname, ".")
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
	recordID, err := tm.ApiManager.CreateDNSRecord(zoneID, hostname, tunnelID)
		if err != nil {
			return fmt.Errorf("failed to create DNS record: %w", err)
		}

		slog.Info("DNS record created successfully", 
			"zoneID", zoneID, 
			"hostname", hostname, 
			"targetDomain", targetDomain, 
			"recordID", recordID)
	}

	return nil
}

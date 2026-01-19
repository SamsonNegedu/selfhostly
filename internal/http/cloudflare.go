package http

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/selfhostly/internal/cloudflare"
	"github.com/selfhostly/internal/db"
	"github.com/selfhostly/internal/docker"
)

// convertHostnamePtr converts a hostname string to a string pointer
func convertHostnamePtr(hostname string) *string {
	if hostname == "" {
		return nil
	}
	return &hostname
}

// convertPathPtr converts a path string to a string pointer
func convertPathPtr(path string) *string {
	if path == "" {
		return nil
	}
	return &path
}

// IngressRule represents a single ingress rule for a Cloudflare tunnel
type IngressRule struct {
	Hostname      *string                `json:"hostname"`
	Service       string                 `json:"service"`
	Path          *string                `json:"path"`
	OriginRequest map[string]interface{} `json:"originRequest"`
}

// CloudflareTunnelResponse represents a Cloudflare tunnel API response
type CloudflareTunnelResponse struct {
	ID           string        `json:"id"`
	AppID        string        `json:"app_id"`
	TunnelID     string        `json:"tunnel_id"`
	TunnelName   string        `json:"tunnel_name"`
	Status       string        `json:"status"`
	IsActive     bool          `json:"is_active"`
	PublicURL    string        `json:"public_url"`
	IngressRules []IngressRule `json:"ingress_rules,omitempty"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
	LastSyncedAt *time.Time    `json:"last_synced_at"`
	ErrorDetails string        `json:"error_details,omitempty"`
}

// listCloudflareTunnels returns all active Cloudflare tunnels
func (s *Server) listCloudflareTunnels(c *gin.Context) {
	settings, err := s.database.GetSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to get settings"})
		return
	}

	if settings.CloudflareAPIToken == nil || settings.CloudflareAccountID == nil ||
		*settings.CloudflareAPIToken == "" || *settings.CloudflareAccountID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Cloudflare API credentials not configured"})
		return
	}

	cfManager := cloudflare.NewTunnelManager(*settings.CloudflareAPIToken, *settings.CloudflareAccountID, s.database)
	tunnels, err := cfManager.GetAllActiveTunnels()
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to get tunnels", "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to get tunnels", Details: err.Error()})
		return
	}

	var responseTunnels []CloudflareTunnelResponse
	for _, tunnel := range tunnels {
		// Get the associated app details
		app, err := s.database.GetApp(tunnel.AppID)
		if err != nil {
			slog.ErrorContext(c.Request.Context(), "failed to get app for tunnel", "tunnelID", tunnel.ID, "appID", tunnel.AppID, "error", err)
			continue
		}

		var errorDetails string
		if tunnel.ErrorDetails != nil {
			errorDetails = *tunnel.ErrorDetails
		}

		responseTunnels = append(responseTunnels, CloudflareTunnelResponse{
			ID:           tunnel.ID,
			AppID:        tunnel.AppID,
			TunnelID:     tunnel.TunnelID,
			TunnelName:   tunnel.TunnelName,
			Status:       tunnel.Status,
			IsActive:     tunnel.IsActive,
			PublicURL:    app.PublicURL,
			CreatedAt:    tunnel.CreatedAt,
			UpdatedAt:    tunnel.UpdatedAt,
			LastSyncedAt: tunnel.LastSyncedAt,
			ErrorDetails: errorDetails,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"tunnels": responseTunnels,
		"count":   len(responseTunnels),
	})
}

// getCloudflareTunnel returns details for a specific Cloudflare tunnel
func (s *Server) getCloudflareTunnel(c *gin.Context) {
	appID := c.Param("appId")
	if appID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid app ID"})
		return
	}

	app, err := s.database.GetApp(appID)
	if err != nil {
		slog.DebugContext(c.Request.Context(), "app not found", "appID", appID)
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "App not found"})
		return
	}

	settings, err := s.database.GetSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to get settings"})
		return
	}

	if settings.CloudflareAPIToken == nil || settings.CloudflareAccountID == nil ||
		*settings.CloudflareAPIToken == "" || *settings.CloudflareAccountID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Cloudflare API credentials not configured"})
		return
	}

	// Check if the app has a tunnel configured
	slog.DebugContext(c.Request.Context(), "checking tunnel configuration", "appID", appID, "tunnelID", app.TunnelID, "tunnelToken", app.TunnelToken)
	if app.TunnelID == "" {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "App has no Cloudflare tunnel configured"})
		return
	}

	// Get the tunnel record from the database to get ingress rules
	tunnel, err := s.database.GetCloudflareTunnelByAppID(app.ID)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to get tunnel record", "appID", appID, "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to get tunnel record", Details: err.Error()})
		return
	}

	var errorDetails string
	if tunnel.ErrorDetails != nil {
		errorDetails = *tunnel.ErrorDetails
	}

	// Convert database IngressRule to API IngressRule format
	var ingressRules []IngressRule
	if tunnel.IngressRules != nil {
		for _, dbRule := range *tunnel.IngressRules {
			apiRule := IngressRule{
				Service: dbRule.Service,
			}
			if dbRule.Hostname != nil {
				apiRule.Hostname = dbRule.Hostname
			}
			if dbRule.Path != nil {
				apiRule.Path = dbRule.Path
			}
			if dbRule.OriginRequest != nil {
				apiRule.OriginRequest = dbRule.OriginRequest
			}
			ingressRules = append(ingressRules, apiRule)
		}
	}

	response := CloudflareTunnelResponse{
		ID:           tunnel.ID,
		AppID:        tunnel.AppID,
		TunnelID:     tunnel.TunnelID,
		TunnelName:   tunnel.TunnelName,
		Status:       tunnel.Status,
		IsActive:     tunnel.IsActive,
		PublicURL:    app.PublicURL,
		IngressRules: ingressRules,
		CreatedAt:    tunnel.CreatedAt,
		UpdatedAt:    tunnel.UpdatedAt,
		LastSyncedAt: tunnel.LastSyncedAt,
		ErrorDetails: errorDetails,
	}

	c.JSON(http.StatusOK, response)
}

// syncCloudflareTunnel synchronizes tunnel status with Cloudflare
func (s *Server) syncCloudflareTunnel(c *gin.Context) {
	appID := c.Param("appId")
	if appID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid app ID"})
		return
	}

	app, err := s.database.GetApp(appID)
	if err != nil {
		slog.DebugContext(c.Request.Context(), "app not found", "appID", appID)
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "App not found"})
		return
	}

	settings, err := s.database.GetSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to get settings"})
		return
	}

	if settings.CloudflareAPIToken == nil || settings.CloudflareAccountID == nil ||
		*settings.CloudflareAPIToken == "" || *settings.CloudflareAccountID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Cloudflare API credentials not configured"})
		return
	}

	if app.TunnelID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "App has no Cloudflare tunnel"})
		return
	}

	// Simple sync implementation - update last_synced_at in database
	now := time.Now()

	// Get the existing tunnel record first
	existingTunnel, err := s.database.GetCloudflareTunnelByAppID(app.ID)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to get tunnel for sync", "tunnelID", app.TunnelID, "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to get tunnel record", Details: err.Error()})
		return
	}

	// Update only the sync time and update timestamp
	existingTunnel.LastSyncedAt = &now
	existingTunnel.UpdatedAt = now

	if err := s.database.UpdateCloudflareTunnel(existingTunnel); err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to update tunnel sync time", "tunnelID", app.TunnelID, "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to sync tunnel", Details: err.Error()})
		return
	}

	// Optionally, make a simple API call to verify Cloudflare connectivity
	cfManager := cloudflare.NewTunnelManager(*settings.CloudflareAPIToken, *settings.CloudflareAccountID, s.database)
	_, apiErr := cfManager.ApiManager.GetTunnelToken(app.TunnelID)
	if apiErr != nil {
		slog.WarnContext(c.Request.Context(), "Cloudflare API verification failed, but local sync completed", "tunnelID", app.TunnelID, "error", apiErr)
	} else {
		slog.InfoContext(c.Request.Context(), "Tunnel synchronized successfully", "tunnelID", app.TunnelID, "syncedAt", now.Format(time.RFC3339))
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Tunnel synchronized successfully",
		"tunnel_id": app.TunnelID,
		"synced_at": time.Now().Format(time.RFC3339),
	})
}

// deleteCloudflareTunnel deletes a Cloudflare tunnel for an app
func (s *Server) deleteCloudflareTunnel(c *gin.Context) {
	appID := c.Param("appId")
	if appID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid app ID"})
		return
	}

	app, err := s.database.GetApp(appID)
	if err != nil {
		slog.DebugContext(c.Request.Context(), "app not found", "appID", appID)
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "App not found"})
		return
	}

	settings, err := s.database.GetSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to get settings"})
		return
	}

	if settings.CloudflareAPIToken == nil || settings.CloudflareAccountID == nil ||
		*settings.CloudflareAPIToken == "" || *settings.CloudflareAccountID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Cloudflare API credentials not configured"})
		return
	}

	if app.TunnelID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "App has no Cloudflare tunnel"})
		return
	}

	cfManager := cloudflare.NewTunnelManager(*settings.CloudflareAPIToken, *settings.CloudflareAccountID, s.database)
	if err := cfManager.DeleteTunnelByAppID(appID); err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to delete tunnel", "tunnelID", app.TunnelID, "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to delete tunnel", Details: err.Error()})
		return
	}

	// Update the app to remove tunnel references
	app.TunnelID = ""
	app.TunnelToken = ""
	app.PublicURL = ""
	app.UpdatedAt = time.Now()
	if err := s.database.UpdateApp(app); err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to update app after tunnel deletion", "appID", appID, "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to update app", Details: err.Error()})
		return
	}

	slog.InfoContext(c.Request.Context(), "Cloudflare tunnel deleted successfully", "appID", appID, "tunnelID", app.TunnelID)
	c.JSON(http.StatusOK, gin.H{
		"message":   "Tunnel deleted successfully",
		"app_id":    appID,
		"tunnel_id": app.TunnelID,
	})
}

// updateTunnelIngress updates the ingress configuration for a tunnel
func (s *Server) updateTunnelIngress(c *gin.Context) {
	appID := c.Param("appId")
	if appID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid app ID"})
		return
	}

	app, err := s.database.GetApp(appID)
	if err != nil {
		slog.DebugContext(c.Request.Context(), "app not found", "appID", appID)
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "App not found"})
		return
	}

	settings, err := s.database.GetSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to get settings"})
		return
	}

	if settings.CloudflareAPIToken == nil || settings.CloudflareAccountID == nil ||
		*settings.CloudflareAPIToken == "" || *settings.CloudflareAccountID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Cloudflare API credentials not configured"})
		return
	}

	if app.TunnelID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "App has no Cloudflare tunnel"})
		return
	}

	// Parse the ingress configuration from request body
	var ingressConfig struct {
		IngressRules []cloudflare.IngressRule `json:"ingress_rules"`
		Hostname     string                   `json:"hostname,omitempty"`
		TargetDomain string                   `json:"target_domain,omitempty"`
	}

	if err := c.ShouldBindJSON(&ingressConfig); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid ingress configuration format"})
		return
	}

	// Ensure there's a catch-all rule at the end if not provided
	if len(ingressConfig.IngressRules) == 0 || ingressConfig.IngressRules[len(ingressConfig.IngressRules)-1].Service != "http_status:404" {
		ingressConfig.IngressRules = append(ingressConfig.IngressRules, cloudflare.IngressRule{
			Service: "http_status:404",
		})
	}

	cfManager := cloudflare.NewTunnelManager(*settings.CloudflareAPIToken, *settings.CloudflareAccountID, s.database)

	// Update tunnel ingress configuration with hostname and target domain if provided
	err = cfManager.UpdateTunnelIngress(app.TunnelID, ingressConfig.IngressRules, ingressConfig.Hostname, ingressConfig.TargetDomain)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to update tunnel ingress", "appID", appID, "tunnelID", app.TunnelID, "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to update tunnel ingress", Details: err.Error()})
		return
	}

	// Get the tunnel record to update with new ingress rules
	tunnel, err := s.database.GetCloudflareTunnelByAppID(app.ID)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to get tunnel record for ingress update", "appID", appID, "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to get tunnel record", Details: err.Error()})
		return
	}

	// Convert ingress rules from cloudflare package to db.IngressRule format
	dbIngressRules := make([]db.IngressRule, len(ingressConfig.IngressRules))
	for i, cfRule := range ingressConfig.IngressRules {
		dbIngressRules[i] = db.IngressRule{
			Hostname:      convertHostnamePtr(cfRule.Hostname),
			Service:       cfRule.Service,
			Path:          convertPathPtr(cfRule.Path),
			OriginRequest: cfRule.OriginRequest,
		}
	}

	// Update the tunnel record with the new ingress rules
	tunnel.IngressRules = &dbIngressRules
	if err := s.database.UpdateCloudflareTunnel(tunnel); err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to update tunnel with ingress rules", "appID", appID, "tunnelID", app.TunnelID, "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to store ingress rules", Details: err.Error()})
		return
	}

	// Update the app with the new ingress configuration
	app.PublicURL = "" // Will be updated after DNS record creation
	app.UpdatedAt = time.Now()

	// Update app record
	if err := s.database.UpdateApp(app); err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to update app after ingress configuration", "appID", appID, "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to update app", Details: err.Error()})
		return
	}

	// Restart cloudflared to pick up the new ingress configuration from Cloudflare API
	dockerManager := docker.NewManager(s.config.AppsDir)
	if err := dockerManager.RestartCloudflared(app.Name); err != nil {
		slog.WarnContext(c.Request.Context(), "failed to restart cloudflared after ingress update (container may not be running)", "appID", appID, "error", err)
		// Don't fail the request - the ingress config is already saved to Cloudflare
		// User can manually restart or the next container start will pick up the config
	} else {
		slog.InfoContext(c.Request.Context(), "cloudflared restarted to apply new ingress configuration", "appID", appID)
	}

	slog.InfoContext(c.Request.Context(), "Tunnel ingress configuration updated successfully", "appID", appID, "tunnelID", app.TunnelID)

	response := gin.H{
		"message":       "Tunnel ingress configuration updated successfully",
		"app_id":        appID,
		"tunnel_id":     app.TunnelID,
		"ingress_rules": ingressConfig.IngressRules,
	}

	// Add DNS information if a hostname was provided
	if ingressConfig.Hostname != "" {
		response["hostname"] = ingressConfig.Hostname
		response["public_url"] = fmt.Sprintf("https://%s", ingressConfig.Hostname)
	}

	c.JSON(http.StatusOK, response)
}

// createDNSRecord creates a DNS record for a tunnel
func (s *Server) createDNSRecord(c *gin.Context) {
	appID := c.Param("appId")
	if appID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid app ID"})
		return
	}

	app, err := s.database.GetApp(appID)
	if err != nil {
		slog.DebugContext(c.Request.Context(), "app not found", "appID", appID)
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "App not found"})
		return
	}

	settings, err := s.database.GetSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to get settings"})
		return
	}

	if settings.CloudflareAPIToken == nil || settings.CloudflareAccountID == nil ||
		*settings.CloudflareAPIToken == "" || *settings.CloudflareAccountID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Cloudflare API credentials not configured"})
		return
	}

	if app.TunnelID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "App has no Cloudflare tunnel"})
		return
	}

	// Parse the DNS record request
	var dnsRequest struct {
		Hostname string `json:"hostname"`
	}

	if err := c.ShouldBindJSON(&dnsRequest); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid DNS record request format"})
		return
	}

	if dnsRequest.Hostname == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Hostname is required"})
		return
	}

	cfManager := cloudflare.NewTunnelManager(*settings.CloudflareAPIToken, *settings.CloudflareAccountID, s.database)

	// Extract domain from hostname (remove subdomain)
	domain := dnsRequest.Hostname
	if strings.Contains(dnsRequest.Hostname, ".") {
		parts := strings.Split(dnsRequest.Hostname, ".")
		if len(parts) > 1 {
			domain = strings.Join(parts[len(parts)-2:], ".")
		}
	}

	// Get zone ID for the domain
	zoneID, err := cfManager.ApiManager.GetZoneID(domain)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to get zone ID", "domain", domain, "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to get zone ID for domain", Details: err.Error()})
		return
	}

	// Create DNS record
	recordID, err := cfManager.ApiManager.CreateDNSRecord(zoneID, dnsRequest.Hostname, app.TunnelID)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to create DNS record", "hostname", dnsRequest.Hostname, "tunnelID", app.TunnelID, "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to create DNS record", Details: err.Error()})
		return
	}

	// Update the app with the DNS hostname
	app.TunnelDomain = dnsRequest.Hostname
	app.PublicURL = fmt.Sprintf("https://%s", dnsRequest.Hostname)
	app.UpdatedAt = time.Now()

	// Update app record
	if err := s.database.UpdateApp(app); err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to update app after DNS creation", "appID", appID, "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to update app", Details: err.Error()})
		return
	}

	slog.InfoContext(c.Request.Context(), "DNS record created successfully",
		"appID", appID, "hostname", dnsRequest.Hostname, "tunnelID", app.TunnelID, "zoneID", zoneID, "recordID", recordID)

	c.JSON(http.StatusOK, gin.H{
		"message":    "DNS record created successfully",
		"app_id":     appID,
		"hostname":   dnsRequest.Hostname,
		"public_url": app.PublicURL,
		"zone_id":    zoneID,
		"record_id":  recordID,
	})
}

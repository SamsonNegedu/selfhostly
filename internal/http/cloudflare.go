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
	"github.com/selfhostly/internal/domain"
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

// applyIngressRulesInternal is an internal helper that applies ingress rules to a tunnel
// It handles: updating tunnel ingress config, storing to DB, creating DNS records, and restarting cloudflared
func (s *Server) applyIngressRulesInternal(app *db.App, tunnelID string, ingressRules []cloudflare.IngressRule, settings *db.Settings, createDNS bool) error {
	tunnelManager := cloudflare.NewTunnelManager(*settings.CloudflareAPIToken, *settings.CloudflareAccountID, s.database)

	// Extract hostname for backward compatibility
	var hostname, targetDomain string
	if len(ingressRules) > 0 && ingressRules[0].Hostname != "" {
		hostname = ingressRules[0].Hostname
	}

	// Update tunnel ingress configuration
	if err := tunnelManager.UpdateTunnelIngress(tunnelID, ingressRules, hostname, targetDomain); err != nil {
		return fmt.Errorf("failed to update tunnel ingress: %w", err)
	}

	// Get and update tunnel record
	tunnel, err := s.database.GetCloudflareTunnelByAppID(app.ID)
	if err != nil {
		return fmt.Errorf("failed to get tunnel record: %w", err)
	}

	// Convert ingress rules to db format
	dbIngressRules := make([]db.IngressRule, len(ingressRules))
	for i, cfRule := range ingressRules {
		dbIngressRules[i] = db.IngressRule{
			Hostname:      convertHostnamePtr(cfRule.Hostname),
			Service:       cfRule.Service,
			Path:          convertPathPtr(cfRule.Path),
			OriginRequest: cfRule.OriginRequest,
		}
	}

	tunnel.IngressRules = &dbIngressRules
	if err := s.database.UpdateCloudflareTunnel(tunnel); err != nil {
		return fmt.Errorf("failed to update tunnel record: %w", err)
	}

	// Create DNS records if requested
	if createDNS {
		initialPublicURL := app.PublicURL
		for _, rule := range ingressRules {
			if rule.Hostname != "" {
				// Extract domain from hostname
				domain := rule.Hostname
				if strings.Contains(rule.Hostname, ".") {
					parts := strings.Split(rule.Hostname, ".")
					if len(parts) > 1 {
						domain = strings.Join(parts[len(parts)-2:], ".")
					}
				}

				// Get zone ID for the domain
				zoneID, err := tunnelManager.ApiManager.GetZoneID(domain)
				if err != nil {
					slog.Error("failed to get zone ID for DNS record", "hostname", rule.Hostname, "domain", domain, "error", err)
					continue
				}

				// Create DNS record
				_, err = tunnelManager.ApiManager.CreateDNSRecord(zoneID, rule.Hostname, tunnelID)
				if err != nil {
					slog.Error("failed to create DNS record", "hostname", rule.Hostname, "tunnelID", tunnelID, "error", err)
					continue
				}

				slog.Info("DNS record created successfully", "appID", app.ID, "hostname", rule.Hostname, "tunnelID", tunnelID)

				// Update app with first hostname as public URL
				if app.PublicURL == "" || app.PublicURL == initialPublicURL {
					app.TunnelDomain = rule.Hostname
					app.PublicURL = fmt.Sprintf("https://%s", rule.Hostname)
					app.UpdatedAt = time.Now()
					if err := s.database.UpdateApp(app); err != nil {
						slog.Error("failed to update app with custom hostname", "appID", app.ID, "error", err)
					}
				}
			}
		}
	}

	// Restart cloudflared to pick up the new ingress configuration
	if err := s.dockerManager.RestartCloudflared(app.Name); err != nil {
		slog.Warn("failed to restart cloudflared after ingress update", "appID", app.ID, "error", err)
		// Don't fail - config is already saved
	} else {
		slog.Info("cloudflared restarted to apply new ingress configuration", "appID", app.ID)
	}

	return nil
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
	tunnels, err := s.tunnelService.ListActiveTunnels(c.Request.Context())
	if err != nil {
		s.handleServiceError(c, "list tunnels", err)
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

	tunnel, err := s.tunnelService.GetTunnelByAppID(c.Request.Context(), appID)
	if err != nil {
		s.handleServiceError(c, "get tunnel", err)
		return
	}

	// Get app for public URL
	app, _ := s.appService.GetApp(c.Request.Context(), appID)

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

	if err := s.tunnelService.SyncTunnelStatus(c.Request.Context(), appID); err != nil {
		s.handleServiceError(c, "sync tunnel", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Tunnel synchronized successfully",
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

	if err := s.tunnelService.DeleteTunnel(c.Request.Context(), appID); err != nil {
		s.handleServiceError(c, "delete tunnel", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Tunnel deleted successfully",
		"app_id":  appID,
	})
}

// updateTunnelIngress updates the ingress configuration for a tunnel
func (s *Server) updateTunnelIngress(c *gin.Context) {
	appID := c.Param("appId")
	if appID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid app ID"})
		return
	}

	var req domain.UpdateIngressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid ingress configuration format"})
		return
	}

	if err := s.tunnelService.UpdateTunnelIngress(c.Request.Context(), appID, req); err != nil {
		s.handleServiceError(c, "update tunnel ingress", err)
		return
	}

	app, _ := s.appService.GetApp(c.Request.Context(), appID)

	response := gin.H{
		"message":       "Tunnel ingress configuration updated successfully",
		"app_id":        appID,
		"ingress_rules": req.IngressRules,
	}

	if req.Hostname != "" {
		response["hostname"] = req.Hostname
		response["public_url"] = fmt.Sprintf("https://%s", req.Hostname)
	}
	if app != nil {
		response["tunnel_id"] = app.TunnelID
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

	var dnsRequest struct {
		Hostname string `json:"hostname" binding:"required"`
	}

	if err := c.ShouldBindJSON(&dnsRequest); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid DNS record request format"})
		return
	}

	// Extract domain from hostname
	domainName := dnsRequest.Hostname
	if strings.Contains(dnsRequest.Hostname, ".") {
		parts := strings.Split(dnsRequest.Hostname, ".")
		if len(parts) > 1 {
			domainName = strings.Join(parts[len(parts)-2:], ".")
		}
	}

	req := domain.CreateDNSRequest{
		Hostname: dnsRequest.Hostname,
		Domain:   domainName,
	}

	if err := s.tunnelService.CreateDNSRecord(c.Request.Context(), appID, req); err != nil {
		s.handleServiceError(c, "create DNS record", err)
		return
	}

	// Get updated app
	app, _ := s.appService.GetApp(c.Request.Context(), appID)

	c.JSON(http.StatusOK, gin.H{
		"message":    "DNS record created successfully",
		"app_id":     appID,
		"hostname":   dnsRequest.Hostname,
		"public_url": fmt.Sprintf("https://%s", dnsRequest.Hostname),
	})
	
	if app != nil {
		c.JSON(http.StatusOK, gin.H{
			"message":    "DNS record created successfully",
			"app_id":     appID,
			"hostname":   dnsRequest.Hostname,
			"public_url": app.PublicURL,
		})
	}
}

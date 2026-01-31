package http

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/selfhostly/internal/db"
	"github.com/selfhostly/internal/domain"
	"github.com/selfhostly/internal/tunnel"
)

// ListTunnelProviders lists all available tunnel providers
// GET /api/tunnels/providers
func (s *Server) ListTunnelProviders(c *gin.Context) {
	ctx := c.Request.Context()
	slog.InfoContext(ctx, "listing tunnel providers")

	providers, err := s.tunnelService.ListProviders(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to list providers", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list providers"})
		return
	}

	// Get active provider name
	settings, err := s.database.GetSettings()
	activeProvider := "cloudflare" // default
	if err == nil {
		activeProvider = settings.GetActiveProviderName()
	}

	c.JSON(http.StatusOK, gin.H{
		"providers": providers,
		"active":    activeProvider,
	})
}

// GetProviderFeatures returns the features supported by a specific provider
// GET /api/tunnels/providers/:provider/features
func (s *Server) GetProviderFeatures(c *gin.Context) {
	ctx := c.Request.Context()
	providerName := c.Param("provider")

	slog.InfoContext(ctx, "getting provider features", "provider", providerName)

	features, err := s.tunnelService.GetProviderFeatures(ctx, providerName)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get provider features", "provider", providerName, "error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, features)
}

// GetTunnelByAppIDGeneric retrieves tunnel information using provider abstraction
// This is the new provider-agnostic endpoint
// GET /api/tunnels/apps/:appId
func (s *Server) GetTunnelByAppIDGeneric(c *gin.Context) {
	ctx := c.Request.Context()
	appID := c.Param("appId")

	// Get node_id from middleware (already validated)
	nodeID := getNodeIDFromContext(c)
	if nodeID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "node_id is required"})
		return
	}

	slog.InfoContext(ctx, "getting tunnel", "appID", appID, "nodeID", nodeID)

	tun, err := s.tunnelService.GetTunnelByAppID(ctx, appID, nodeID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get tunnel", "appID", appID, "error", err)
		s.handleServiceError(c, "get tunnel", err)
		return
	}
	
	// Get app for public URL
	var app *db.App
	if nodeID != "" {
		app, _ = s.appService.GetApp(ctx, appID, nodeID)
	}

	publicURL := ""
	if app != nil {
		publicURL = app.PublicURL
	}

	// Add public_url to response
	c.JSON(http.StatusOK, gin.H{
		"id":             tun.ID,
		"app_id":         tun.AppID,
		"tunnel_id":      tun.TunnelID,
		"tunnel_name":    tun.TunnelName,
		"status":         tun.Status,
		"is_active":      tun.IsActive,
		"public_url":     publicURL,
		"ingress_rules":  tun.IngressRules,
		"created_at":     tun.CreatedAt,
		"updated_at":     tun.UpdatedAt,
		"last_synced_at": tun.LastSyncedAt,
		"error_details":  func() string { if tun.ErrorDetails != nil { return *tun.ErrorDetails }; return "" }(),
	})
}

// ListTunnelsGeneric lists all tunnels using provider abstraction
// GET /api/tunnels
func (s *Server) ListTunnelsGeneric(c *gin.Context) {
	ctx := c.Request.Context()
	nodeIDsParam := c.QueryArray("node_ids[]")

	slog.InfoContext(ctx, "listing tunnels", "nodeIDs", nodeIDsParam)

	tunnels, err := s.tunnelService.ListActiveTunnels(ctx, nodeIDsParam)
	if err != nil {
		slog.ErrorContext(ctx, "failed to list tunnels", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tunnels": tunnels,
		"count":   len(tunnels),
	})
}

// SyncTunnelStatusGeneric syncs tunnel status (if provider supports it)
// POST /api/tunnels/apps/:appId/sync
func (s *Server) SyncTunnelStatusGeneric(c *gin.Context) {
	ctx := c.Request.Context()
	appID := c.Param("appId")

	// Get node_id from middleware (already validated)
	nodeID := getNodeIDFromContext(c)
	if nodeID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "node_id is required"})
		return
	}

	slog.InfoContext(ctx, "syncing tunnel status", "appID", appID, "nodeID", nodeID)

	if err := s.tunnelService.SyncTunnelStatus(ctx, appID, nodeID); err != nil {
		// Check if it's a feature not supported error
		if _, ok := err.(*tunnel.FeatureNotSupportedError); ok {
			c.JSON(http.StatusNotImplemented, gin.H{"error": err.Error()})
			return
		}

		slog.ErrorContext(ctx, "failed to sync tunnel status", "appID", appID, "error", err)
		s.handleServiceError(c, "sync tunnel status", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "tunnel status synced successfully"})
}

// UpdateTunnelIngressGeneric updates tunnel ingress rules (if provider supports it)
// PUT /api/tunnels/apps/:appId/ingress
func (s *Server) UpdateTunnelIngressGeneric(c *gin.Context) {
	ctx := c.Request.Context()
	appID := c.Param("appId")

	// Get node_id from middleware (already validated)
	nodeID := getNodeIDFromContext(c)
	if nodeID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "node_id is required"})
		return
	}

	var req domain.UpdateIngressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	slog.InfoContext(ctx, "updating tunnel ingress", "appID", appID, "nodeID", nodeID)

	if err := s.tunnelService.UpdateTunnelIngress(ctx, appID, nodeID, req); err != nil {
		// Check if it's a feature not supported error
		if _, ok := err.(*tunnel.FeatureNotSupportedError); ok {
			c.JSON(http.StatusNotImplemented, gin.H{"error": err.Error()})
			return
		}

		slog.ErrorContext(ctx, "failed to update tunnel ingress", "appID", appID, "error", err)
		s.handleServiceError(c, "update tunnel ingress", err)
		return
	}
	
	// Restart tunnel container if needed (best effort)
	if err := s.appService.RestartCloudflared(ctx, appID, nodeID); err != nil {
		slog.WarnContext(ctx, "failed to restart tunnel container", "appID", appID, "error", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "ingress rules updated successfully",
		"appID":         appID,
		"ingress_rules": req.IngressRules,
	})
}

// CreateDNSRecordGeneric creates a DNS record (if provider supports it)
// POST /api/tunnels/apps/:appId/dns
func (s *Server) CreateDNSRecordGeneric(c *gin.Context) {
	ctx := c.Request.Context()
	appID := c.Param("appId")

	// Get node_id from middleware (already validated)
	nodeID := getNodeIDFromContext(c)
	if nodeID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "node_id is required"})
		return
	}

	var dnsRequest struct {
		Hostname string `json:"hostname" binding:"required"`
	}

	if err := c.ShouldBindJSON(&dnsRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	slog.InfoContext(ctx, "creating DNS record", "appID", appID, "hostname", dnsRequest.Hostname, "nodeID", nodeID)

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

	if err := s.tunnelService.CreateDNSRecord(ctx, appID, nodeID, req); err != nil {
		// Check if it's a feature not supported error
		if _, ok := err.(*tunnel.FeatureNotSupportedError); ok {
			c.JSON(http.StatusNotImplemented, gin.H{"error": err.Error()})
			return
		}

		slog.ErrorContext(ctx, "failed to create DNS record", "appID", appID, "error", err)
		s.handleServiceError(c, "create DNS record", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "DNS record created successfully",
		"app_id":   appID,
		"hostname": dnsRequest.Hostname,
	})
}

// DeleteTunnelGeneric deletes a tunnel
// DELETE /api/tunnels/apps/:appId
func (s *Server) DeleteTunnelGeneric(c *gin.Context) {
	ctx := c.Request.Context()
	appID := c.Param("appId")

	// Get node_id from middleware (already validated)
	nodeID := getNodeIDFromContext(c)
	if nodeID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "node_id is required"})
		return
	}

	slog.InfoContext(ctx, "deleting tunnel", "appID", appID, "nodeID", nodeID)

	if err := s.tunnelService.DeleteTunnel(ctx, appID, nodeID); err != nil {
		slog.ErrorContext(ctx, "failed to delete tunnel", "appID", appID, "error", err)
		s.handleServiceError(c, "delete tunnel", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "tunnel deleted successfully", "app_id": appID})
}

// ===== Local Tunnel Handlers (for inter-node communication) =====

// getLocalTunnelByAppID returns tunnel for a local app
func (s *Server) getLocalTunnelByAppID(c *gin.Context) {
	appID := c.Param("appId")

	// Directly access database, bypassing routing logic
	tunnel, err := s.database.GetCloudflareTunnelByAppID(appID)
	if err != nil {
		// Check for not found error
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "Resource not found", Details: "tunnel not found"})
			return
		}
		s.handleServiceError(c, "get local tunnel", err)
		return
	}

	c.JSON(http.StatusOK, tunnel)
}

// syncLocalTunnelStatus syncs tunnel status for a local app
func (s *Server) syncLocalTunnelStatus(c *gin.Context) {
	ctx := c.Request.Context()
	appID := c.Param("appId")

	nodeID := s.config.Node.ID

	if err := s.tunnelService.SyncTunnelStatus(ctx, appID, nodeID); err != nil {
		s.handleServiceError(c, "sync local tunnel status", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "tunnel status synced successfully"})
}

// updateLocalTunnelIngress updates tunnel ingress rules for a local app
func (s *Server) updateLocalTunnelIngress(c *gin.Context) {
	ctx := c.Request.Context()
	appID := c.Param("appId")

	var req domain.UpdateIngressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	nodeID := s.config.Node.ID

	if err := s.tunnelService.UpdateTunnelIngress(ctx, appID, nodeID, req); err != nil {
		s.handleServiceError(c, "update local tunnel ingress", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "ingress rules updated successfully"})
}

// createLocalTunnelDNSRecord creates a DNS record for a local tunnel
func (s *Server) createLocalTunnelDNSRecord(c *gin.Context) {
	ctx := c.Request.Context()
	appID := c.Param("appId")

	var req domain.CreateDNSRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	nodeID := s.config.Node.ID

	if err := s.tunnelService.CreateDNSRecord(ctx, appID, nodeID, req); err != nil {
		s.handleServiceError(c, "create local tunnel DNS record", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "DNS record created successfully"})
}

// deleteLocalTunnel deletes a local tunnel
func (s *Server) deleteLocalTunnel(c *gin.Context) {
	ctx := c.Request.Context()
	appID := c.Param("appId")

	nodeID := s.config.Node.ID

	if err := s.tunnelService.DeleteTunnel(ctx, appID, nodeID); err != nil {
		s.handleServiceError(c, "delete local tunnel", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "tunnel deleted successfully"})
}

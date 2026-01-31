package http

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/selfhostly/internal/db"
	"github.com/selfhostly/internal/domain"
	"github.com/selfhostly/internal/httputil"
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
		c.JSON(http.StatusNotFound, gin.H{"error": domain.PublicMessage(err)})
		return
	}

	c.JSON(http.StatusOK, features)
}

// tunnelByAppEnvelope is the single response shape for GET /api/tunnels/apps/:appId (primary and secondary).
// Always returned so primary vs secondary responses are consistent.
func tunnelByAppEnvelope(appID, nodeID, tunnelMode, publicURL string, tun *db.CloudflareTunnel) gin.H {
	env := gin.H{
		"tunnel":      nil,
		"app_id":      appID,
		"tunnel_mode": tunnelMode,
		"node_id":     nodeID,
		"public_url":  publicURL,
	}
	if tun != nil {
		url := publicURL
		if tun.PublicURL != "" {
			url = tun.PublicURL
		}
		env["tunnel"] = gin.H{
			"id":             tun.ID,
			"app_id":         tun.AppID,
			"tunnel_id":      tun.TunnelID,
			"tunnel_name":    tun.TunnelName,
			"status":         tun.Status,
			"is_active":      tun.IsActive,
			"public_url":     url,
			"ingress_rules":  tun.IngressRules,
			"created_at":     tun.CreatedAt,
			"updated_at":     tun.UpdatedAt,
			"last_synced_at": tun.LastSyncedAt,
			"error_details":  func() string { if tun.ErrorDetails != nil { return *tun.ErrorDetails }; return "" }(),
		}
	}
	return env
}

// GetTunnelByAppIDGeneric retrieves tunnel information using provider abstraction.
// Always returns the same envelope: { tunnel, app_id, tunnel_mode, node_id, public_url } so primary and secondary responses are consistent.
// GET /api/tunnels/apps/:appId
func (s *Server) GetTunnelByAppIDGeneric(c *gin.Context) {
	ctx := c.Request.Context()
	appID := c.Param("appId")

	nodeID := getNodeIDFromContext(c)
	if nodeID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "node_id is required"})
		return
	}

	slog.InfoContext(ctx, "getting tunnel", "appID", appID, "nodeID", nodeID)

	app, errApp := s.appService.GetApp(ctx, appID, nodeID)
	if errApp != nil {
		s.handleServiceError(c, "get app", errApp)
		return
	}

	tun, err := s.tunnelService.GetTunnelByAppID(ctx, appID, nodeID)
	if err != nil {
		if domain.IsNotFoundError(err) {
			c.JSON(http.StatusOK, tunnelByAppEnvelope(appID, nodeID, app.TunnelMode, app.PublicURL, nil))
			return
		}
		slog.ErrorContext(ctx, "failed to get tunnel", "appID", appID, "error", err)
		s.handleServiceError(c, "get tunnel", err)
		return
	}

	publicURL := app.PublicURL
	if tun.PublicURL != "" {
		publicURL = tun.PublicURL
	}
	c.JSON(http.StatusOK, tunnelByAppEnvelope(appID, nodeID, "custom", publicURL, tun))
}

// ListTunnelsGeneric lists all tunnels using provider abstraction
// GET /api/tunnels
func (s *Server) ListTunnelsGeneric(c *gin.Context) {
	ctx := c.Request.Context()
	var nodeIDs []string
	if scope, ok := c.Get("request_scope"); ok && scope == "local" {
		nodeIDs = []string{s.config.Node.ID}
	} else {
		nodeIDs = httputil.ParseNodeIDs(c)
	}

	slog.InfoContext(ctx, "listing tunnels", "nodeIDs", nodeIDs)

	tunnels, err := s.tunnelService.ListActiveTunnels(ctx, nodeIDs)
	if err != nil {
		slog.ErrorContext(ctx, "failed to list tunnels", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.PublicMessage(err)})
		return
	}

	// Tunnel is source of truth for public_url. Only set node_id from app (and fallback public_url for legacy rows).
	for _, t := range tunnels {
		app, err := s.database.GetApp(t.AppID)
		if err == nil {
			t.NodeID = app.NodeID
			if t.PublicURL == "" {
				t.PublicURL = app.PublicURL
			}
		}
	}

	// When request_scope is local (node-to-node), node client expects raw array
	if scope, ok := c.Get("request_scope"); ok && scope == "local" {
		c.JSON(http.StatusOK, tunnels)
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
			c.JSON(http.StatusNotImplemented, gin.H{"error": domain.PublicMessage(err)})
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
			c.JSON(http.StatusNotImplemented, gin.H{"error": domain.PublicMessage(err)})
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
			c.JSON(http.StatusNotImplemented, gin.H{"error": domain.PublicMessage(err)})
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

// CreateTunnelForAppGeneric creates a named (custom domain) tunnel for an app that has none.
// POST /api/tunnels/apps/:appId (with node_id). Optional body: { "ingress_rules": [ { "hostname": "...", "service": "...", "path": "..." } ] }
func (s *Server) CreateTunnelForAppGeneric(c *gin.Context) {
	ctx := c.Request.Context()
	appID := c.Param("appId")
	nodeID := getNodeIDFromContext(c)
	if nodeID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "node_id is required"})
		return
	}
	var body struct {
		IngressRules []db.IngressRule `json:"ingress_rules"`
	}
	_ = c.ShouldBindJSON(&body)

	app, handledLocally, err := s.appService.CreateTunnelForApp(ctx, appID, nodeID, &body)
	if err != nil {
		s.handleServiceError(c, "create tunnel for app", err)
		return
	}
	if handledLocally && len(body.IngressRules) > 0 {
		ingressReq := domain.UpdateIngressRequest{IngressRules: body.IngressRules}
		if err := s.tunnelService.UpdateTunnelIngress(ctx, appID, nodeID, ingressReq); err != nil {
			if _, ok := err.(*tunnel.FeatureNotSupportedError); !ok {
				slog.WarnContext(ctx, "failed to apply ingress rules after create tunnel", "appID", appID, "error", err)
			}
		} else {
			if app, err = s.appService.GetApp(ctx, appID, nodeID); err == nil {
				// Return app with updated public_url from first hostname
			}
		}
	}
	c.JSON(http.StatusOK, app)
}

// SwitchAppToCustomTunnelGeneric switches an app from Quick Tunnel to a named (custom domain) tunnel.
// POST /api/tunnels/apps/:appId/switch-to-custom (with node_id). Optional body: { "ingress_rules": [ { "hostname": "...", "service": "...", "path": "..." } ] }
// When ingress_rules are provided, they are applied immediately so the public URL is the custom domain, not cfargotunnel.com.
func (s *Server) SwitchAppToCustomTunnelGeneric(c *gin.Context) {
	ctx := c.Request.Context()
	appID := c.Param("appId")
	nodeID := getNodeIDFromContext(c)
	if nodeID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "node_id is required"})
		return
	}
	var body struct {
		IngressRules []db.IngressRule `json:"ingress_rules"`
	}
	_ = c.ShouldBindJSON(&body)

	app, err := s.appService.SwitchAppToCustomTunnel(ctx, appID, nodeID, &body)
	if err != nil {
		s.handleServiceError(c, "switch to custom tunnel", err)
		return
	}
	if len(body.IngressRules) > 0 {
		ingressReq := domain.UpdateIngressRequest{IngressRules: body.IngressRules}
		if err := s.tunnelService.UpdateTunnelIngress(ctx, appID, nodeID, ingressReq); err != nil {
			if _, ok := err.(*tunnel.FeatureNotSupportedError); !ok {
				slog.WarnContext(ctx, "failed to apply ingress rules after switch to custom", "appID", appID, "error", err)
			}
		} else {
			if app, err = s.appService.GetApp(ctx, appID, nodeID); err == nil {
				c.JSON(http.StatusOK, app)
				return
			}
		}
	}
	c.JSON(http.StatusOK, app)
}

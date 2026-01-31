package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// Frontend auth: login, logout, OAuth callbacks. When behind the gateway, the gateway
	// proxies /auth/* and /avatar/* here; the primary runs OAuth and sets cookies for the
	// gateway's domain. The primary (not the gateway) implements auth so secrets stay in one place.
	// go-pkgz/auth expects paths relative to mount point, so we strip /auth prefix.
	if s.authService != nil {
		authHandler, avatarHandler := s.AuthHandlers()
		if authHandler != nil {
			// Rewrite OAuth redirects to X-Forwarded-Host when behind gateway (no BASE_URL on primary)
			s.engine.Any("/auth/*path", wrapAuthHandler(s.wrapAuthRedirects(authHandler), "/auth"))
		}
		if avatarHandler != nil {
			s.engine.Any("/avatar/*path", wrapAuthHandler(avatarHandler, "/avatar"))
		}
	}

	// Health check endpoint (no auth required)
	s.engine.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "selfhostly",
		})
	})

	// Node auto-registration: no pre-auth (node doesn't exist yet). Handler validates REGISTRATION_TOKEN in body.
	s.engine.POST("/api/nodes/register", s.autoRegisterNode)

	// Single API: user auth OR node auth (composite auth)
	api := s.engine.Group("/api")
	api.Use(s.userOrNodeAuthMiddleware())
	{
		// App routes (resolveNodeMiddleware sets node_id_param for resource-by-id when user auth)
		s.setupAppRoutes(api)

		// Settings: GET dispatches by auth (user=getSettings, node=getSettingsForNode)
		s.setupSettingsRoutes(api)

		// Tunnel routes (provider-agnostic)
		s.setupTunnelRoutes(api)

		// System/monitoring routes
		s.setupSystemRoutes(api)

		// Node management routes
		s.setupNodeRoutes(api)

		// Node-only routes (require node auth)
		api.POST("/nodes/:id/heartbeat", s.requireNodeAuthMiddleware(), s.sendNodeHeartbeat)

		// User info endpoint (user auth only in practice)
		api.GET("/me", s.getCurrentUser)
	}

	// Serve frontend static files
	s.engine.Static("/assets", "./web/dist/assets")

	// Serve favicon explicitly before NoRoute catches it
	s.engine.StaticFile("/favicon.svg", "./web/dist/favicon.svg")

	s.engine.NoRoute(func(c *gin.Context) {
		c.File("./web/dist/index.html")
	})
}

func (s *Server) setupAppRoutes(api *gin.RouterGroup) {
	apps := api.Group("/apps")
	{
		// List and create don't require node_id
		apps.GET("", s.listApps)
		apps.POST("", s.createApp)

		// App-specific operations require node_id (from query when user auth, from context when node auth)
		appSpecific := apps.Group("/:id", s.resolveNodeMiddleware())
		{
			appSpecific.GET("", s.getApp)
			appSpecific.PUT("", s.updateApp)
			appSpecific.DELETE("", s.deleteApp)
			appSpecific.POST("/start", s.startApp)
			appSpecific.POST("/stop", s.stopApp)
			appSpecific.POST("/update", s.updateAppContainers)
			appSpecific.GET("/logs", s.getAppLogs)
			appSpecific.GET("/stats", s.getAppStats)
			appSpecific.GET("/quick-tunnel-url", s.getQuickTunnelURL)
			appSpecific.POST("/quick-tunnel", s.createQuickTunnelForApp)

			// Compose version routes
			appSpecific.GET("/compose/versions", s.getComposeVersions)
			appSpecific.GET("/compose/versions/:version", s.getComposeVersion)
			appSpecific.POST("/compose/rollback/:version", s.rollbackToVersion)
		}
	}
}

func (s *Server) setupTunnelRoutes(api *gin.RouterGroup) {
	tunnels := api.Group("/tunnels")
	{
		// Provider discovery
		tunnels.GET("/providers", s.ListTunnelProviders)
		tunnels.GET("/providers/:provider/features", s.GetProviderFeatures)

		// List all tunnels
		tunnels.GET("", s.ListTunnelsGeneric)

		// App-specific tunnel operations require node_id
		tunnelOps := tunnels.Group("/apps/:appId", s.resolveNodeMiddleware())
		{
			tunnelOps.GET("", s.GetTunnelByAppIDGeneric)
			tunnelOps.POST("", s.CreateTunnelForAppGeneric)
			tunnelOps.POST("/switch-to-custom", s.SwitchAppToCustomTunnelGeneric)
			tunnelOps.POST("/sync", s.SyncTunnelStatusGeneric)
			tunnelOps.PUT("/ingress", s.UpdateTunnelIngressGeneric)
			tunnelOps.POST("/dns", s.CreateDNSRecordGeneric)
			tunnelOps.DELETE("", s.DeleteTunnelGeneric)
		}
	}
}

func (s *Server) setupSettingsRoutes(api *gin.RouterGroup) {
	settings := api.Group("/settings")
	{
		settings.GET("", s.getSettingsDispatch)
		settings.PUT("", s.updateSettings)
	}
}

func (s *Server) setupSystemRoutes(api *gin.RouterGroup) {
	systemGroup := api.Group("/system")
	{
		systemGroup.GET("/stats", s.getSystemStats)

		// Only expose debug endpoints in non-production environments
		if s.config.Environment != "production" {
			systemGroup.GET("/debug/docker-stats/:id", s.getDebugDockerStats)
		}

		systemGroup.POST("/containers/:id/restart", s.restartContainer)
		systemGroup.POST("/containers/:id/stop", s.stopContainer)
		systemGroup.DELETE("/containers/:id", s.deleteContainer)
	}
}

func (s *Server) setupNodeRoutes(api *gin.RouterGroup) {
	nodes := api.Group("/nodes")
	{
		nodes.GET("", s.listNodes)
		nodes.POST("", s.registerNode)
		nodes.GET("/:id", s.getNode)
		nodes.PUT("/:id", s.updateNode)
		nodes.DELETE("/:id", s.deleteNode)
		nodes.GET("/:id/health", s.checkNodeHealth)
		nodes.POST("/:id/check", s.manualCheckNode) // Manual health check trigger (for UI)
	}

	// Current node info
	api.GET("/node/info", s.getCurrentNodeInfo)
}

// getCurrentUser returns the authenticated user info
func (s *Server) getCurrentUser(c *gin.Context) {
	user, exists := getUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Not authenticated",
			Details: "Please login with GitHub to continue",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":      user.ID,
		"name":    user.Name,
		"picture": user.Picture,
	})
}

// wrapAuthHandler wraps an http.Handler for use with Gin, stripping the prefix
// go-pkgz/auth expects paths relative to where it's mounted
func wrapAuthHandler(handler http.Handler, prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Strip the prefix from the URL path for the handler
		originalPath := c.Request.URL.Path
		c.Request.URL.Path = strings.TrimPrefix(originalPath, prefix)

		// Serve using the wrapped handler
		handler.ServeHTTP(c.Writer, c.Request)

		// Restore original path (in case anything else needs it)
		c.Request.URL.Path = originalPath
	}
}

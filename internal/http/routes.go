package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// Mount auth routes (login, logout, callbacks)
	// go-pkgz/auth expects paths relative to mount point, so we strip /auth prefix
	if s.authService != nil {
		authHandler, avatarHandler := s.AuthHandlers()
		if authHandler != nil {
			s.engine.Any("/auth/*path", wrapAuthHandler(authHandler, "/auth"))
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

	// API routes - all protected by authentication
	api := s.engine.Group("/api")
	api.Use(s.getAuthMiddleware())
	{
		// App routes
		s.setupAppRoutes(api)

		// Settings routes
		s.setupSettingsRoutes(api)

		// Cloudflare routes
		s.setupCloudflareRoutes(api)

		// System/monitoring routes
		s.setupSystemRoutes(api)

		// Node management routes
		s.setupNodeRoutes(api)

		// User info endpoint
		api.GET("/me", s.getCurrentUser)
	}

	// Internal API routes for inter-node communication (requires node auth)
	internal := s.engine.Group("/api/internal")
	internal.Use(s.nodeAuthMiddleware())
	{
		internal.GET("/settings", s.getSettingsForNode)

		// App management for inter-node communication (local only, no aggregation)
		internal.GET("/apps", s.listLocalApps)
		internal.GET("/apps/:id", s.getLocalApp)
		internal.POST("/apps", s.createLocalApp)
		internal.PUT("/apps/:id", s.updateLocalApp)
		internal.POST("/apps/:id/start", s.startLocalApp)
		internal.POST("/apps/:id/stop", s.stopLocalApp)
		internal.DELETE("/apps/:id", s.deleteLocalApp)

		// System stats for inter-node communication (local only, no aggregation)
		internal.GET("/system/stats", s.getLocalSystemStats)

		// Cloudflare tunnels for inter-node communication (local only, no aggregation)
		internal.GET("/cloudflare/tunnels", s.listLocalTunnels)
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

		// App-specific operations require node_id
		appSpecific := apps.Group("/:id", s.requireNodeIDMiddleware())
		{
			appSpecific.GET("", s.getApp)
			appSpecific.PUT("", s.updateApp)
			appSpecific.DELETE("", s.deleteApp)
			appSpecific.POST("/start", s.startApp)
			appSpecific.POST("/stop", s.stopApp)
			appSpecific.POST("/update", s.updateAppContainers)
			appSpecific.GET("/logs", s.getAppLogs)
			appSpecific.GET("/stats", s.getAppStats)
			appSpecific.POST("/repair", s.repairApp)

			// Compose version routes
			appSpecific.GET("/compose/versions", s.getComposeVersions)
			appSpecific.GET("/compose/versions/:version", s.getComposeVersion)
			appSpecific.POST("/compose/rollback/:version", s.rollbackToVersion)
		}
	}
}

func (s *Server) setupCloudflareRoutes(api *gin.RouterGroup) {
	cloudflare := api.Group("/cloudflare")
	{
		cloudflare.GET("/tunnels", s.listCloudflareTunnels)

		// App-specific tunnel operations require node_id
		tunnelOps := cloudflare.Group("/apps/:appId", s.requireNodeIDMiddleware())
		{
			tunnelOps.GET("/tunnel", s.getCloudflareTunnel)
			tunnelOps.POST("/tunnel/sync", s.syncCloudflareTunnel)
			tunnelOps.PUT("/tunnel/ingress", s.updateTunnelIngress)
			tunnelOps.POST("/tunnel/dns", s.createDNSRecord)
			tunnelOps.DELETE("/tunnel", s.deleteCloudflareTunnel)
		}
	}
}

func (s *Server) setupSettingsRoutes(api *gin.RouterGroup) {
	settings := api.Group("/settings")
	{
		settings.GET("", s.getSettings)
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

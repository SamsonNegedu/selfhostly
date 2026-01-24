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

		// User info endpoint
		api.GET("/me", s.getCurrentUser)
	}

	// Serve frontend static files
	s.engine.Static("/assets", "./web/dist/assets")
	s.engine.NoRoute(func(c *gin.Context) {
		c.File("./web/dist/index.html")
	})
}

func (s *Server) setupAppRoutes(api *gin.RouterGroup) {
	apps := api.Group("/apps")
	{
		apps.GET("", s.listApps)
		apps.POST("", s.createApp)
		apps.GET("/:id", s.getApp)
		apps.PUT("/:id", s.updateApp)
		apps.DELETE("/:id", s.deleteApp)
		apps.POST("/:id/start", s.startApp)
		apps.POST("/:id/stop", s.stopApp)
		apps.POST("/:id/update", s.updateAppContainers)
		apps.GET("/:id/logs", s.getAppLogs)
		apps.GET("/:id/stats", s.getAppStats)
		apps.POST("/:id/repair", s.repairApp)

		// Compose version routes
		apps.GET("/:id/compose/versions", s.getComposeVersions)
		apps.GET("/:id/compose/versions/:version", s.getComposeVersion)
		apps.POST("/:id/compose/rollback/:version", s.rollbackToVersion)
	}
}

func (s *Server) setupCloudflareRoutes(api *gin.RouterGroup) {
	cloudflare := api.Group("/cloudflare")
	{
		cloudflare.GET("/tunnels", s.listCloudflareTunnels)
		cloudflare.GET("/apps/:appId/tunnel", s.getCloudflareTunnel)
		cloudflare.POST("/apps/:appId/tunnel/sync", s.syncCloudflareTunnel)
		cloudflare.PUT("/apps/:appId/tunnel/ingress", s.updateTunnelIngress)
		cloudflare.POST("/apps/:appId/tunnel/dns", s.createDNSRecord)
		cloudflare.DELETE("/apps/:appId/tunnel", s.deleteCloudflareTunnel)
	}
}

func (s *Server) setupSettingsRoutes(api *gin.RouterGroup) {
	settings := api.Group("/settings")
	{
		settings.GET("", s.getSettings)
		settings.PUT("", s.updateSettings)
	}
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

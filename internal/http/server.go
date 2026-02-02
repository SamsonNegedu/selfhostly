package http

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-pkgz/auth"
	"github.com/go-pkgz/auth/avatar"
	"github.com/go-pkgz/auth/token"
	"github.com/selfhostly/internal/config"
	"github.com/selfhostly/internal/constants"
	"github.com/selfhostly/internal/db"
	"github.com/selfhostly/internal/docker"
	"github.com/selfhostly/internal/domain"
	"github.com/selfhostly/internal/jobs"
	"github.com/selfhostly/internal/node"
	"github.com/selfhostly/internal/routing"
	"github.com/selfhostly/internal/service"
)

// Server wraps the HTTP server
type Server struct {
	config         *config.Config
	database       *db.DB          // Kept temporarily for settings access
	dockerManager  *docker.Manager // Kept temporarily for backward compatibility
	appService     domain.AppService
	tunnelService  domain.TunnelService
	systemService  domain.SystemService
	composeService domain.ComposeService
	nodeService    domain.NodeService
	jobWorker      *jobs.Worker // Background job processor
	engine         *gin.Engine
	authService    *auth.Service
	httpServer     *http.Server
	shutdownCtx    context.Context
	shutdownCancel context.CancelFunc
}

// NewServer creates a new HTTP server
func NewServer(cfg *config.Config, database *db.DB) *Server {
	// Set Gin mode based on environment
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	engine := gin.Default()

	// Middleware - order matters
	engine.Use(securityHeadersMiddleware())
	engine.Use(corsMiddleware(cfg))
	engine.Use(cacheControlMiddleware())
	engine.Use(loggerMiddleware())
	engine.Use(jsonBodyLimitMiddleware(maxBodySize))

	// Initialize auth service
	var authService *auth.Service
	if cfg.Auth.Enabled {
		authService = initAuthService(cfg)
	}

	// Request body size limit
	engine.MaxMultipartMemory = maxBodySize

	// Initialize docker manager
	dockerManager := docker.NewManager(cfg.AppsDir)

	// Initialize logger
	logger := slog.Default()

	// Initialize services (Phase 2 integration)
	tunnelService := service.NewTunnelService(database, dockerManager, cfg, logger)
	appService := service.NewAppService(database, dockerManager, cfg, logger, tunnelService)
	systemService := service.NewSystemService(database, dockerManager, cfg, logger)

	// Initialize routing dependencies for compose service
	composeNodeClient := node.NewClient()
	composeRouter := routing.NewNodeRouter(database, composeNodeClient, cfg.Node.ID, logger)
	composeService := service.NewComposeService(database, dockerManager, composeRouter, composeNodeClient, logger)

	nodeService := service.NewNodeService(database, cfg, logger)

	// Initialize job processing system
	jobProcessor := jobs.NewProcessor(database, dockerManager, appService, tunnelService, logger)
	jobWorker := jobs.NewWorker(jobProcessor, database, constants.JobWorkerPollInterval, logger)

	// Create shutdown context
	shutdownCtx, shutdownCancel := context.WithCancel(context.Background())

	// Initialize server
	server := &Server{
		config:         cfg,
		database:       database,
		dockerManager:  dockerManager,
		appService:     appService,
		tunnelService:  tunnelService,
		systemService:  systemService,
		composeService: composeService,
		nodeService:    nodeService,
		jobWorker:      jobWorker,
		engine:         engine,
		authService:    authService,
		shutdownCtx:    shutdownCtx,
		shutdownCancel: shutdownCancel,
	}

	// Setup routes
	server.setupRoutes()

	return server
}

// initAuthService initializes go-pkgz/auth with GitHub OAuth
func initAuthService(cfg *config.Config) *auth.Service {
	// Determine base URL - must include /auth since we mount at /auth/*
	baseURL := cfg.Auth.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	// Create auth service options
	// URL must include /auth prefix since that's where we mount the handlers
	opts := auth.Opts{
		SecretReader: token.SecretFunc(func(id string) (string, error) {
			return cfg.Auth.JWTSecret, nil
		}),
		TokenDuration:  time.Hour * 24,     // Token valid for 24 hours
		CookieDuration: time.Hour * 24 * 7, // Cookie valid for 7 days
		Issuer:         "selfhostly",
		URL:            baseURL + "/auth", // Include /auth prefix for callback URLs
		AvatarStore:    avatar.NewNoOp(),  // No avatar storage
		SecureCookies:  cfg.Auth.SecureCookie,
		DisableXSRF:    true, // Disable for API usage
		Validator: token.ValidatorFunc(func(_ string, claims token.Claims) bool {
			// Verify user exists
			if claims.User == nil {
				slog.Warn("JWT validation failed: no user in claims")
				return false
			}

			// If no whitelist is configured, reject all access (fail-secure)
			if len(cfg.Auth.GitHub.AllowedUsers) == 0 {
				slog.Warn("GitHub auth enabled but no allowed users configured - rejecting access", "username", claims.User.Name)
				return false
			}

			// Check if GitHub username is in the whitelist
			// GitHub usernames are case-insensitive, so normalize for comparison
			username := strings.ToLower(claims.User.Name)
			for _, allowedUser := range cfg.Auth.GitHub.AllowedUsers {
				if username == strings.ToLower(allowedUser) {
					slog.Info("User authorized", "username", claims.User.Name)
					return true
				}
			}

			// User not in whitelist
			slog.Warn("Unauthorized GitHub user attempted access", "username", username, "allowedUsers", len(cfg.Auth.GitHub.AllowedUsers))
			return false
		}),
	}

	// Create auth service
	authService := auth.NewService(opts)

	// Add GitHub OAuth provider
	authService.AddProvider("github", cfg.Auth.GitHub.ClientID, cfg.Auth.GitHub.ClientSecret)

	return authService
}

// wrapAuthRedirects wraps the auth handler so that when behind a gateway (X-Forwarded-Host set),
// 3xx redirect Location URLs pointing at the primary's host are rewritten to the public host.
// This keeps OAuth callbacks on the gateway URL without configuring the gateway URL on the primary.
func (s *Server) wrapAuthRedirects(handler http.Handler) http.Handler {
	primaryBase := s.config.Auth.BaseURL
	primaryURL, _ := url.Parse(primaryBase)
	primaryHost := ""
	if primaryURL != nil {
		primaryHost = primaryURL.Host
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		forwardedHost := r.Header.Get("X-Forwarded-Host")
		forwardedProto := r.Header.Get("X-Forwarded-Proto")
		if forwardedHost == "" || primaryHost == "" {
			handler.ServeHTTP(w, r)
			return
		}
		if forwardedProto == "" {
			forwardedProto = "http"
		}
		rw := &redirectRewriter{
			ResponseWriter: w,
			primaryHost:    primaryHost,
			publicHost:     forwardedHost,
			publicScheme:   forwardedProto,
		}
		handler.ServeHTTP(rw, r)
	})
}

// redirectRewriter rewrites 3xx Location headers from primary host to public host (X-Forwarded-*).
type redirectRewriter struct {
	http.ResponseWriter
	primaryHost  string
	publicHost   string
	publicScheme string
	status       int
	wroteHeader  bool
}

func (r *redirectRewriter) WriteHeader(code int) {
	if r.wroteHeader {
		return
	}
	r.status = code
	r.wroteHeader = true
	if code >= 300 && code < 400 {
		if loc := r.Header().Get("Location"); loc != "" {
			u, err := url.Parse(loc)
			if err == nil {
				// Case 1: Direct redirect to primary host (e.g. callback landing page)
				if u.Host == r.primaryHost {
					u.Scheme = r.publicScheme
					u.Host = r.publicHost
					r.Header().Set("Location", u.String())
				} else {
					// Case 2: OAuth redirect to external provider with redirect_uri query param
					// (e.g. GitHub OAuth URL with redirect_uri=http://primary:8082/auth/callback)
					// Rewrite redirect_uri if it points to primary host
					query := u.Query()
					if redirectURI := query.Get("redirect_uri"); redirectURI != "" {
						if redirectURL, err := url.Parse(redirectURI); err == nil && redirectURL.Host == r.primaryHost {
							redirectURL.Scheme = r.publicScheme
							redirectURL.Host = r.publicHost
							query.Set("redirect_uri", redirectURL.String())
							u.RawQuery = query.Encode()
							r.Header().Set("Location", u.String())
						}
					}
				}
			}
		}
	}
	r.ResponseWriter.WriteHeader(code)
}

const (
	maxBodySize = 10 << 20 // 10MB max request body
)

var (
	readTimeout  = constants.ServerReadTimeout  // 30s for reading request
	writeTimeout = constants.ServerWriteTimeout // 2 minutes for long operations (app creation, tunnel setup)
	idleTimeout  = constants.ServerIdleTimeout  // 2 minutes idle
)

// Run starts the HTTP server and background tasks
func (s *Server) Run() error {
	addr := s.config.ServerAddress
	if addr == "" {
		addr = ":8080"
	}

	// Start background tasks
	s.startBackgroundTasks()

	// Configure server with timeouts
	s.httpServer = &http.Server{
		Addr:           addr,
		Handler:        s.engine,
		ReadTimeout:    readTimeout,
		WriteTimeout:   writeTimeout,
		IdleTimeout:    idleTimeout,
		MaxHeaderBytes: 1 << 20, // 1MB max header size
	}

	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	slog.Info("Starting graceful shutdown...")

	// Cancel background tasks
	if s.shutdownCancel != nil {
		slog.Info("Stopping background tasks...")
		s.shutdownCancel()
	}

	// Shutdown HTTP server (stops accepting new connections, waits for existing requests)
	if s.httpServer != nil {
		slog.Info("Shutting down HTTP server...")
		if err := s.httpServer.Shutdown(ctx); err != nil {
			slog.Error("Error during HTTP server shutdown", "error", err)
			return err
		}
	}

	// Close database connections
	if s.database != nil {
		slog.Info("Closing database connections...")
		if err := s.database.Close(); err != nil {
			slog.Error("Error closing database", "error", err)
			return err
		}
	}

	slog.Info("Graceful shutdown completed successfully")
	return nil
}

// startBackgroundTasks starts periodic background tasks like health checks
func (s *Server) startBackgroundTasks() {
	// Start periodic health checks for all nodes
	go s.runPeriodicHealthChecks()

	// If this is a secondary node with a configured primary, attempt auto-registration
	if !s.config.Node.IsPrimary && s.config.Node.PrimaryNodeURL != "" {
		go s.attemptAutoRegistration()
		// After registration, start continuous heartbeats
		go s.sendPeriodicHeartbeats()
	}

	// Start job worker for background async operations
	go func() {
		slog.Info("starting job worker")
		if err := s.jobWorker.Start(s.shutdownCtx); err != nil {
			slog.Error("job worker stopped with error", "error", err)
		}
	}()

	slog.Info("background tasks started", "health_check_interval", "30s", "job_worker_enabled", true)
}

// runPeriodicHealthChecks performs health checks on all nodes every 30 seconds
func (s *Server) runPeriodicHealthChecks() {
	// Run immediately on startup
	ctx := context.Background()
	if err := s.nodeService.HealthCheckAllNodes(ctx); err != nil {
		slog.Warn("initial health check failed", "error", err)
	}

	ticker := time.NewTicker(constants.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.shutdownCtx.Done():
			slog.Info("Health check routine shutting down...")
			return
		case <-ticker.C:
			ctx := context.Background()
			if err := s.nodeService.HealthCheckAllNodes(ctx); err != nil {
				slog.Warn("periodic health check failed", "error", err)
			} else {
				slog.Debug("periodic health check completed successfully")
			}
		}
	}
}

// securityHeadersMiddleware adds security-related HTTP headers
func securityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent MIME type sniffing
		c.Writer.Header().Set("X-Content-Type-Options", "nosniff")
		// Prevent clickjacking
		c.Writer.Header().Set("X-Frame-Options", "DENY")
		// Enable XSS protection
		c.Writer.Header().Set("X-XSS-Protection", "1; mode=block")
		// Referrer policy
		c.Writer.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		// HSTS (only if using HTTPS)
		if c.Request.TLS != nil {
			c.Writer.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		c.Next()
	}
}

// corsMiddleware adds CORS headers with configurable origin
func corsMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Check if origin is in allowed list
		allowed := false
		for _, allowedOrigin := range cfg.CORS.AllowedOrigins {
			if origin == allowedOrigin {
				allowed = true
				break
			}
		}

		if allowed {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization, X-XSRF-TOKEN")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400") // 24 hours

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// cacheControlMiddleware sets appropriate cache headers based on content type
func cacheControlMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// API endpoints - no caching for dynamic data
		if strings.HasPrefix(path, "/api/") {
			c.Writer.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			c.Writer.Header().Set("Pragma", "no-cache")
			c.Writer.Header().Set("Expires", "0")
		} else if strings.HasPrefix(path, "/auth/") {
			// Auth endpoints - no caching
			c.Writer.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			c.Writer.Header().Set("Pragma", "no-cache")
			c.Writer.Header().Set("Expires", "0")
		} else if strings.HasPrefix(path, "/assets/") {
			// Static assets - long-term caching with immutable flag
			// These are versioned/hashed files that never change
			c.Writer.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		// For other paths (like index.html), don't set cache headers
		// Let default Gin static file handling decide

		c.Next()
	}
}

// jsonBodyLimitMiddleware limits the size of JSON request bodies to prevent DoS
func jsonBodyLimitMiddleware(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only apply to JSON requests
		if c.Request.Method != "GET" && c.Request.Method != "DELETE" && c.Request.Method != "OPTIONS" {
			contentType := c.GetHeader("Content-Type")
			if strings.Contains(contentType, "application/json") {
				if c.Request.ContentLength > maxBytes {
					c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{
						"error": "Request body too large",
					})
					return
				}
				// Wrap the request body with MaxBytesReader
				c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
			}
		}
		c.Next()
	}
}

// loggerMiddleware logs HTTP requests
func loggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		slog.InfoContext(c.Request.Context(), "HTTP request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"remote_addr", c.Request.RemoteAddr,
		)
		c.Next()
	}
}

// getAuthMiddleware returns a Gin middleware that requires authentication
func (s *Server) getAuthMiddleware() gin.HandlerFunc {
	if s.authService == nil {
		// Auth disabled - allow all requests
		return func(c *gin.Context) {
			c.Next()
		}
	}

	// Get the auth middleware from go-pkgz/auth
	authMiddleware := s.authService.Middleware()

	return func(c *gin.Context) {
		// Debug: log incoming auth attempt
		hasCookie := c.Request.Header.Get("Cookie") != ""
		hasAuth := c.Request.Header.Get("Authorization") != ""
		slog.InfoContext(c.Request.Context(), "user auth attempt",
			"path", c.Request.URL.Path,
			"has_cookie", hasCookie,
			"has_auth_header", hasAuth,
			"host", c.Request.Host,
			"cookie_length", len(c.Request.Header.Get("Cookie")),
		)

		// Wrap the Gin handler for go-pkgz/auth middleware
		var userInfo token.User
		var authenticated bool

		handler := authMiddleware.Auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract user from context
			if u, err := token.GetUserInfo(r); err == nil {
				userInfo = u
				authenticated = true
				slog.InfoContext(r.Context(), "user authenticated",
					"user_id", u.ID,
					"user_name", u.Name,
				)
			} else {
				slog.InfoContext(r.Context(), "user auth failed", "error", err)
			}
			// Update request in gin context
			c.Request = r
		}))

		// Create a response writer wrapper to check if auth failed
		handler.ServeHTTP(c.Writer, c.Request)

		if !authenticated {
			// Override the text/plain response from go-pkgz/auth with JSON
			c.Writer.Header().Set("Content-Type", "application/json")
			c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Authentication required. Please login with GitHub."})
			c.Abort()
			return
		}

		// Store user info in gin context for handlers
		c.Set("user", userInfo)
		c.Next()
	}
}

// getUserFromContext extracts the authenticated user from context
func getUserFromContext(c *gin.Context) (token.User, bool) {
	if user, exists := c.Get("user"); exists {
		if u, ok := user.(token.User); ok {
			return u, true
		}
	}
	return token.User{}, false
}

// userOrNodeAuthMiddleware accepts gateway auth (X-Gateway-API-Key), node auth (X-Node-ID + X-Node-API-Key), or user auth (JWT/session).
// When gateway or node auth is valid, sets node_id_param = local node ID and request_scope = "local" so handlers treat the request as local-only.
// When user auth is valid, does not set target/scope; resolveNodeMiddleware or handlers will use node_id from query/body.
func (s *Server) userOrNodeAuthMiddleware() gin.HandlerFunc {
	tryGatewayAuth := func(c *gin.Context) bool {
		key := c.GetHeader("X-Gateway-API-Key")
		if key == "" || s.config.Node.GatewayAPIKey == "" {
			return false
		}
		if key != s.config.Node.GatewayAPIKey {
			c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "invalid gateway API key", Details: "X-Gateway-API-Key invalid"})
			c.Abort()
			return true
		}
		c.Set("node_id_param", s.config.Node.ID)
		c.Set("request_scope", "local")
		c.Next()
		return true
	}
	tryNodeAuth := func(c *gin.Context) bool {
		nodeID := c.GetHeader("X-Node-ID")
		apiKey := c.GetHeader("X-Node-API-Key")
		if nodeID == "" || apiKey == "" {
			return false
		}
		if !s.config.Node.IsPrimary {
			if apiKey != s.config.Node.APIKey {
				c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "invalid API key", Details: "provided API key does not match this node"})
				c.Abort()
				return true // handled
			}
			c.Set("node_id", nodeID)
		} else {
			node, err := s.database.GetNode(nodeID)
			if err != nil || node.APIKey != apiKey {
				c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unknown or invalid node", Details: "node ID or API key invalid"})
				c.Abort()
				return true
			}
			c.Set("node_id", node.ID)
		}
		// Node auth valid: set target = local, scope = local for list
		c.Set("node_id_param", s.config.Node.ID)
		c.Set("request_scope", "local")
		c.Next()
		return true
	}

	return func(c *gin.Context) {
		if tryGatewayAuth(c) {
			return
		}
		if tryNodeAuth(c) {
			return
		}
		// No gateway or node auth; require user auth
		s.getAuthMiddleware()(c)
	}
}

// resolveNodeMiddleware sets node_id_param from query for user-authenticated requests.
// When request_scope is already set (node auth), does nothing. Otherwise requires node_id query and sets node_id_param.
// Used on resource-by-id routes so handlers get target node from context.
func (s *Server) resolveNodeMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, hasScope := c.Get("request_scope"); hasScope {
			// Node auth already set node_id_param and request_scope
			c.Next()
			return
		}
		nodeID := c.Query("node_id")
		if nodeID == "" {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "node_id is required",
				Details: "node_id query parameter must be provided for this operation",
			})
			c.Abort()
			return
		}
		c.Set("node_id_param", nodeID)
		c.Next()
	}
}

// requireNodeAuthMiddleware ensures the request was authenticated as a node (request_scope set by userOrNodeAuthMiddleware).
// Used for node-only routes (heartbeat, register) that must not be called with user auth.
func (s *Server) requireNodeAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if scope, ok := c.Get("request_scope"); ok && scope == "local" {
			c.Next()
			return
		}
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "node authentication required",
			Details: "this endpoint accepts only node credentials (X-Node-ID, X-Node-API-Key)",
		})
		c.Abort()
	}
}

// AuthHandlers returns the auth HTTP handlers for mounting
func (s *Server) AuthHandlers() (http.Handler, http.Handler) {
	if s.authService == nil {
		return nil, nil
	}
	return s.authService.Handlers()
}

package http

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-pkgz/auth"
	"github.com/go-pkgz/auth/avatar"
	"github.com/go-pkgz/auth/token"
	"github.com/selfhostly/internal/config"
	"github.com/selfhostly/internal/db"
	"github.com/selfhostly/internal/docker"
)

// Server wraps the HTTP server
type Server struct {
	config        *config.Config
	database      *db.DB
	dockerManager *docker.Manager
	engine        *gin.Engine
	authService   *auth.Service
}

// NewServer creates a new HTTP server
func NewServer(cfg *config.Config, database *db.DB) *Server {
	if cfg.Auth.Enabled {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	engine := gin.Default()

	// Middleware - order matters
	engine.Use(securityHeadersMiddleware())
	engine.Use(corsMiddleware(cfg))
	engine.Use(loggerMiddleware())

	// Initialize auth service
	var authService *auth.Service
	if cfg.Auth.Enabled {
		authService = initAuthService(cfg)
	}

	// Request body size limit
	engine.MaxMultipartMemory = maxBodySize

	// Initialize docker manager
	dockerManager := docker.NewManager(cfg.AppsDir)

	// Initialize server
	server := &Server{
		config:        cfg,
		database:      database,
		dockerManager: dockerManager,
		engine:        engine,
		authService:   authService,
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
			// Accept all valid tokens
			return claims.User != nil
		}),
	}

	// Create auth service
	authService := auth.NewService(opts)

	// Add GitHub OAuth provider
	authService.AddProvider("github", cfg.Auth.GitHub.ClientID, cfg.Auth.GitHub.ClientSecret)

	return authService
}

const (
	maxBodySize  = 10 << 20 // 10MB max request body
	readTimeout  = 15 * time.Second
	writeTimeout = 15 * time.Second
	idleTimeout  = 60 * time.Second
)

// Run starts the HTTP server
func (s *Server) Run() error {
	addr := s.config.ServerAddress
	if addr == "" {
		addr = ":8080"
	}

	// Configure server with timeouts
	server := &http.Server{
		Addr:           addr,
		Handler:        s.engine,
		ReadTimeout:    readTimeout,
		WriteTimeout:   writeTimeout,
		IdleTimeout:    idleTimeout,
		MaxHeaderBytes: 1 << 20, // 1MB max header size
	}

	return server.ListenAndServe()
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
		// Wrap the Gin handler for go-pkgz/auth middleware
		var userInfo token.User
		var authenticated bool

		handler := authMiddleware.Auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract user from context
			if u, err := token.GetUserInfo(r); err == nil {
				userInfo = u
				authenticated = true
			}
			// Update request in gin context
			c.Request = r
		}))

		// Create a response writer wrapper to check if auth failed
		handler.ServeHTTP(c.Writer, c.Request)

		if !authenticated {
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

// AuthHandlers returns the auth HTTP handlers for mounting
func (s *Server) AuthHandlers() (http.Handler, http.Handler) {
	if s.authService == nil {
		return nil, nil
	}
	return s.authService.Handlers()
}

package config

import (
	"fmt"
	"os"
	"strings"
)

// Config holds the application configuration
type Config struct {
	ServerAddress string
	DatabasePath  string
	AppsDir       string
	Environment   string // development, staging, production
	Cloudflare    CloudflareConfig
	Auth          AuthConfig
	AutoStart     bool
	CORS          CORSConfig
}

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowedOrigins []string
}

// CloudflareConfig holds Cloudflare API configuration
type CloudflareConfig struct {
	APIToken  string
	AccountID string
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	Enabled      bool
	JWTSecret    string
	GitHub       GitHubOAuthConfig
	SecureCookie bool
	BaseURL      string // Base URL for OAuth callbacks (e.g., http://localhost:8080)
}

// GitHubOAuthConfig holds GitHub OAuth configuration
type GitHubOAuthConfig struct {
	ClientID     string
	ClientSecret string
	AllowedUsers []string // Whitelist of GitHub usernames allowed to access the system
}

// Load loads configuration from environment variables with defaults
func Load() (*Config, error) {
	// Parse CORS allowed origins from comma-separated string
	corsOrigins := getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173,http://localhost:3000,http://localhost:8080")
	allowedOrigins := parseCommaSeparatedList(corsOrigins)

	authEnabled := getEnv("AUTH_ENABLED", "false") == "true"
	jwtSecret := os.Getenv("JWT_SECRET")

	// Validate JWT secret is provided when auth is enabled
	if authEnabled && jwtSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET environment variable is required when AUTH_ENABLED is true")
	}

	cfg := &Config{
		ServerAddress: getEnv("SERVER_ADDRESS", ":8080"),
		DatabasePath:  getEnv("DATABASE_PATH", "./data/selfhostly.db"),
		AppsDir:       getEnv("APPS_DIR", "./apps"),
		Environment:   getEnv("APP_ENV", "production"),
		Cloudflare: CloudflareConfig{
			APIToken:  os.Getenv("CLOUDFLARE_API_TOKEN"),
			AccountID: os.Getenv("CLOUDFLARE_ACCOUNT_ID"),
		},
		Auth: AuthConfig{
			Enabled:      authEnabled,
			JWTSecret:    jwtSecret,
			SecureCookie: getEnv("AUTH_SECURE_COOKIE", "false") == "true",
			BaseURL:      getEnv("AUTH_BASE_URL", "http://localhost:8080"),
			GitHub: GitHubOAuthConfig{
				ClientID:     os.Getenv("GITHUB_CLIENT_ID"),
				ClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
				AllowedUsers: parseCommaSeparatedList(os.Getenv("GITHUB_ALLOWED_USERS")),
			},
		},
		AutoStart: getEnv("AUTO_START_APPS", "false") == "true",
		CORS: CORSConfig{
			AllowedOrigins: allowedOrigins,
		},
	}

	return cfg, nil
}

// parseCommaSeparatedList splits a comma-separated string into a slice
func parseCommaSeparatedList(s string) []string {
	if s == "" {
		return []string{}
	}

	items := strings.Split(s, ",")
	result := make([]string, 0, len(items))

	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			result = append(result, item)
		}
	}

	return result
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

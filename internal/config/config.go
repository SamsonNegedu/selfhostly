package config

import (
	"os"
	"strings"
)

// Config holds the application configuration
type Config struct {
	ServerAddress string
	DatabasePath  string
	AppsDir       string
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
	CookieDomain string
	SecureCookie bool
	BaseURL      string // Base URL for OAuth callbacks (e.g., http://localhost:8080)
}

// GitHubOAuthConfig holds GitHub OAuth configuration
type GitHubOAuthConfig struct {
	ClientID     string
	ClientSecret string
}

// Load loads configuration from environment variables with defaults
func Load() (*Config, error) {
	// Parse CORS allowed origins from comma-separated string
	corsOrigins := getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173,http://localhost:3000,http://localhost:8080")
	allowedOrigins := parseCommaSeparatedList(corsOrigins)

	return &Config{
		ServerAddress: getEnv("SERVER_ADDRESS", ":8080"),
		DatabasePath:  getEnv("DATABASE_PATH", "./data/automaton.db"),
		AppsDir:       getEnv("APPS_DIR", "./apps"),
		Cloudflare: CloudflareConfig{
			APIToken:  os.Getenv("CLOUDFLARE_API_TOKEN"),
			AccountID: os.Getenv("CLOUDFLARE_ACCOUNT_ID"),
		},
		Auth: AuthConfig{
			Enabled:      getEnv("AUTH_ENABLED", "false") == "true",
			JWTSecret:    getEnv("JWT_SECRET", "change-me-in-production-secret-key"),
			CookieDomain: getEnv("AUTH_COOKIE_DOMAIN", "localhost"),
			SecureCookie: getEnv("AUTH_SECURE_COOKIE", "false") == "true",
			BaseURL:      getEnv("AUTH_BASE_URL", "http://localhost:8080"),
			GitHub: GitHubOAuthConfig{
				ClientID:     os.Getenv("GITHUB_CLIENT_ID"),
				ClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
			},
		},
		AutoStart: getEnv("AUTO_START_APPS", "false") == "true",
		CORS: CORSConfig{
			AllowedOrigins: allowedOrigins,
		},
	}, nil
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

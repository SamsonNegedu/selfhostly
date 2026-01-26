package config

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
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
	Node          NodeConfig
}

// NodeConfig holds node-specific configuration for multi-node support
type NodeConfig struct {
	ID              string // This node's UUID (generated on first run if not set)
	Name            string // This node's name
	IsPrimary       bool   // Whether this is the primary node
	APIKey          string // API key for other nodes to authenticate with this node
	PrimaryNodeURL  string // URL of primary node (only for secondary nodes)
	PrimaryNodeKey  string // API key to authenticate with primary (only for secondary nodes)
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

	// Node configuration
	nodeID := getEnv("NODE_ID", "")
	if nodeID == "" || nodeID == "auto" {
		// Generate a new UUID for this node
		nodeID = uuid.New().String()
	}

	nodeName := getEnv("NODE_NAME", "")
	if nodeName == "" {
		// Default to hostname
		hostname, err := os.Hostname()
		if err != nil || hostname == "" {
			nodeName = "node-" + nodeID[:8]
		} else {
			nodeName = hostname
		}
	}

	apiKey := getEnv("NODE_API_KEY", "")
	if apiKey == "" {
		// Generate secure API key
		apiKey = generateSecureAPIKey()
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
		Node: NodeConfig{
			ID:             nodeID,
			Name:           nodeName,
			IsPrimary:      getEnv("NODE_IS_PRIMARY", "true") == "true", // Default to primary for backward compatibility
			APIKey:         apiKey,
			PrimaryNodeURL: getEnv("PRIMARY_NODE_URL", ""),
			PrimaryNodeKey: getEnv("PRIMARY_NODE_API_KEY", ""),
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

// generateSecureAPIKey generates a cryptographically secure random API key
func generateSecureAPIKey() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to UUID if crypto/rand fails
		return uuid.New().String()
	}
	return base64.URLEncoding.EncodeToString(bytes)
}

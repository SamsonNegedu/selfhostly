package gateway

import (
	"errors"
	"os"
	"strconv"
	"time"
)

// Config holds gateway configuration
type Config struct {
	PrimaryBackendURL string        // Primary backend URL (e.g. http://primary:8082)
	GatewayAPIKey     string        // API key gateway sends to backends; must match backends' GATEWAY_API_KEY
	ListenAddress     string        // Address to listen on (e.g. :8080)
	JWTSecret         string        // JWT secret to validate user tokens (same as primary)
	AuthEnabled       bool          // Whether to validate JWT for user requests
	RegistryTTL       time.Duration // How often to refresh node list from primary
}

var ErrGatewayAPIKeyRequired = errors.New("GATEWAY_API_KEY is required")

// LoadConfig loads gateway configuration from environment
func LoadConfig() (*Config, error) {
	primaryBackendURL := os.Getenv("PRIMARY_BACKEND_URL")
	if primaryBackendURL == "" {
		primaryBackendURL = "http://localhost:8082"
	}
	gatewayAPIKey := os.Getenv("GATEWAY_API_KEY")
	if gatewayAPIKey == "" {
		return nil, ErrGatewayAPIKeyRequired
	}
	listenAddr := os.Getenv("GATEWAY_LISTEN_ADDRESS")
	if listenAddr == "" {
		listenAddr = ":8080"
	}
	jwtSecret := os.Getenv("JWT_SECRET")
	authEnabled := os.Getenv("AUTH_ENABLED") == "true"
	ttlSec := 60
	if t := os.Getenv("GATEWAY_REGISTRY_TTL_SEC"); t != "" {
		if n, err := parseInt(t); err == nil && n > 0 {
			ttlSec = n
		}
	}
	return &Config{
		PrimaryBackendURL: primaryBackendURL,
		GatewayAPIKey:     gatewayAPIKey,
		ListenAddress:     listenAddr,
		JWTSecret:         jwtSecret,
		AuthEnabled:       authEnabled,
		RegistryTTL:       time.Duration(ttlSec) * time.Second,
	}, nil
}

func parseInt(s string) (int, error) {
	return strconv.Atoi(s)
}

package gateway

import (
	"errors"
	"os"
	"strconv"
	"strings"
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
	PublicHosts       []string      // Hostnames users reach the gateway on; empty disables host pinning
}

// pinPublicHost returns host when it is a configured public host. When host pinning is on and the
// value is not one of them (a spoofed X-Forwarded-Host, Referer or cookie) the first public host
// is used instead, so an attacker cannot steer OAuth redirects or cookie scoping elsewhere.
func (c *Config) pinPublicHost(host string) (string, bool) {
	if len(c.PublicHosts) == 0 {
		return host, true
	}
	for _, h := range c.PublicHosts {
		if strings.EqualFold(h, host) {
			return host, true
		}
	}
	return c.PublicHosts[0], false
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
	var publicHosts []string
	for _, h := range strings.Split(os.Getenv("PUBLIC_HOSTS"), ",") {
		if h = strings.TrimSpace(h); h != "" {
			publicHosts = append(publicHosts, h)
		}
	}
	return &Config{
		PublicHosts:       publicHosts,
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

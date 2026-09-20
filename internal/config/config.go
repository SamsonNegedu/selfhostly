package config

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/selfhostly/internal/constants"
	"github.com/selfhostly/internal/update"
)

// Config holds the application configuration
type Config struct {
	ServerAddress string
	DatabasePath  string
	AppsDir       string
	Environment   string // development, staging, production
	LogJSON       bool   // Whether to use JSON logging format (defaults based on environment if not set)
	Cloudflare    CloudflareConfig
	Auth          AuthConfig
	AutoStart     bool
	CORS          CORSConfig
	Node          NodeConfig
	Security      SecurityConfig
	Updates       UpdateConfig

	// Sources records where each implicit or generated value came from, so startup logs and
	// `selfhostly doctor` can show what a restart will actually use.
	Sources map[string]string
}

// NodeConfig holds node-specific configuration for multi-node support
type NodeConfig struct {
	ID                string // This node's UUID (generated on first run if not set)
	Name              string // This node's name
	IsPrimary         bool   // Whether this is the primary node
	APIEndpoint       string // This node's API endpoint URL for inter-node communication
	APIKey            string // API key for other nodes to authenticate with this node
	PrimaryNodeURL    string // URL of primary node (only for secondary nodes)
	PrimaryNodeKey    string // API key to authenticate with primary (only for secondary nodes)
	RegistrationToken string // Token for auto-registration (shared secret between primary and secondaries)
	GatewayAPIKey     string // API key the gateway sends; backends accept this alongside node auth
	Transport         string // secondary only: "direct" (primary and gateway call it) or "tunnel" (it dials out)
}

// UsesLink reports whether this secondary reaches its primary over an outbound link
func (n NodeConfig) UsesLink() bool {
	return !n.IsPrimary && n.Transport == constants.NodeTransportTunnel
}

// UpdateConfig holds the settings for updating Selfhostly from the UI. Off unless UI_UPDATES_ENABLED=true.
type UpdateConfig struct {
	Enabled       bool
	ManifestURL   string        // where release.json is published; its signature is at the same URL plus .sig
	PublicKey     string        // base64 ed25519 key the manifest must be signed with (else the built-in one)
	ImagePrefix   string        // repository prefix release images must live under
	CheckInterval time.Duration // how often the primary looks for a new release
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
	BaseURL      string // Base URL for OAuth callbacks (when behind gateway, primary rewrites redirects using X-Forwarded-Host)
}

// GitHubOAuthConfig holds GitHub OAuth configuration
type GitHubOAuthConfig struct {
	ClientID     string
	ClientSecret string
	AllowedUsers []string // Whitelist of GitHub usernames allowed to access the system
}

// SecurityConfig holds security-related configuration
type SecurityConfig struct {
	// AllowedVolumePaths is a list of host paths that are allowed to be mounted as volumes
	// This whitelist overrides the default security restrictions for specific trusted paths
	// Example: /home/user/Documents/apps,/mnt/backup
	AllowedVolumePaths []string

	// Mode is constants.SecurityModeEnforce or constants.SecurityModeWarn. In warn mode policy
	// violations that could break already-running apps are logged instead of blocked.
	Mode string

	// HostAppsDir is the host filesystem path that backs AppsDir. Empty means "detect at startup".
	HostAppsDir string

	// NodeEndpointAllowLoopback permits node endpoints on loopback addresses (single-host setups).
	// Existing installs (warn mode) and development always allow it.
	NodeEndpointAllowLoopback bool

	// NodeEndpointAllowedCIDRs optionally restricts where node endpoints may resolve to.
	NodeEndpointAllowedCIDRs []string

	// EncryptSecretsAtRest controls whether node keys and provider tokens are written encrypted.
	// Reading always accepts both encrypted and plaintext values.
	EncryptSecretsAtRest bool

	// SettingsEncryptionKey optionally supplies the key; otherwise one is created next to the DB.
	SettingsEncryptionKey string

	// AllowUnauthenticated acknowledges running with AUTH_ENABLED=false on a non-loopback address.
	AllowUnauthenticated bool

	// CloudflareAccess enables verification of the Cf-Access-Jwt-Assertion header.
	CloudflareAccess CloudflareAccessConfig

	// SessionHours is the browser session lifetime
	SessionHours int
}

// CloudflareAccessConfig identifies the Cloudflare Access application whose JWTs are trusted
type CloudflareAccessConfig struct {
	TeamDomain string
	Audience   string
}

// Enabled reports whether Cloudflare Access verification is fully configured
func (c CloudflareAccessConfig) Enabled() bool {
	return c.TeamDomain != "" && c.Audience != ""
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

	databasePath := getEnv("DATABASE_PATH", "./data/selfhostly.db")
	dataDir := filepath.Dir(databasePath)

	// Node configuration. When NODE_ID is unset or "auto" the identity is generated once and
	// persisted next to the database so a redeploy does not mint a new node.
	sources := map[string]string{}
	nodeID := getEnv("NODE_ID", "")
	if nodeID == "" || nodeID == "auto" {
		var src string
		nodeID, src = loadOrCreatePersisted(dataDir, constants.NodeIDFileName, uuid.New().String)
		sources["node_id"] = src
	} else {
		sources["node_id"] = "environment (NODE_ID)"
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
		var src string
		apiKey, src = loadOrCreatePersisted(dataDir, constants.NodeAPIKeyFileName, generateSecureAPIKey)
		sources["node_api_key"] = src
	} else {
		sources["node_api_key"] = "environment (NODE_API_KEY)"
	}

	nodeAPIEndpoint := getEnv("NODE_API_ENDPOINT", "http://localhost:8080")

	// Generate or load registration token for cluster joining
	registrationToken := os.Getenv("REGISTRATION_TOKEN")
	if registrationToken == "" && getEnv("NODE_IS_PRIMARY", "true") == "true" {
		// Primary nodes generate a token if not provided
		var src string
		registrationToken, src = loadOrCreatePersisted(dataDir, constants.RegistrationFile, generateSecureToken)
		sources["registration_token"] = src
		slog.Warn("No REGISTRATION_TOKEN set - using a generated token",
			"token_file", filepath.Join(dataDir, constants.RegistrationFile))
		slog.Info("Copy the token from that file into secondary nodes, or issue single-use join tokens instead")
	}

	authBaseURL := getEnv("AUTH_BASE_URL", "")
	if authBaseURL == "" {
		authBaseURL = nodeAPIEndpoint
	}

	environment := getEnv("APP_ENV", constants.EnvProduction)

	securityMode, modeSource := resolveSecurityMode(databasePath, environment)
	sources["security_mode"] = modeSource

	// Secure cookies default to on when the public URL is https
	secureCookieDefault := "false"
	if strings.HasPrefix(strings.ToLower(authBaseURL), "https://") {
		secureCookieDefault = "true"
	}

	// Determine JSON logging preference
	// If LOG_JSON is explicitly set, use it; otherwise default based on environment
	logJSONEnv := getEnv("LOG_JSON", "")
	var logJSON bool
	if logJSONEnv != "" {
		logJSON = logJSONEnv == "true"
	} else {
		// Default: JSON in production, text in development
		logJSON = environment != "development"
	}

	cfg := &Config{
		Sources:       sources,
		ServerAddress: getEnv("SERVER_ADDRESS", ":8080"),
		DatabasePath:  databasePath,
		AppsDir:       getEnv("APPS_DIR", "./apps"),
		Environment:   environment,
		LogJSON:       logJSON,
		Cloudflare: CloudflareConfig{
			APIToken:  os.Getenv("CLOUDFLARE_API_TOKEN"),
			AccountID: os.Getenv("CLOUDFLARE_ACCOUNT_ID"),
		},
		Auth: AuthConfig{
			Enabled:      authEnabled,
			JWTSecret:    jwtSecret,
			SecureCookie: getEnv("AUTH_SECURE_COOKIE", secureCookieDefault) == "true",
			BaseURL:      authBaseURL,
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
			ID:                nodeID,
			Name:              nodeName,
			IsPrimary:         getEnv("NODE_IS_PRIMARY", "true") == "true", // Default to primary for backward compatibility
			APIEndpoint:       nodeAPIEndpoint,
			APIKey:            apiKey,
			PrimaryNodeURL:    getEnv("PRIMARY_NODE_URL", ""),
			PrimaryNodeKey:    getEnv("PRIMARY_NODE_API_KEY", ""),
			RegistrationToken: registrationToken,
			GatewayAPIKey:     os.Getenv("GATEWAY_API_KEY"),
			Transport:         resolveNodeTransport(os.Getenv("NODE_TRANSPORT")),
		},
		Security: SecurityConfig{
			AllowedVolumePaths:        parseCommaSeparatedList(os.Getenv("ALLOWED_VOLUME_PATHS")),
			Mode:                      securityMode,
			HostAppsDir:               os.Getenv("HOST_APPS_DIR"),
			NodeEndpointAllowedCIDRs:  parseCommaSeparatedList(os.Getenv("NODE_ENDPOINT_ALLOWED_CIDRS")),
			NodeEndpointAllowLoopback: os.Getenv("NODE_ENDPOINT_ALLOW_LOOPBACK") == "true",
			EncryptSecretsAtRest:      os.Getenv("ENCRYPT_SECRETS_AT_REST") == "true",
			SettingsEncryptionKey:     os.Getenv("SETTINGS_ENCRYPTION_KEY"),
			AllowUnauthenticated:      os.Getenv("ALLOW_UNAUTHENTICATED") == "true",
			CloudflareAccess: CloudflareAccessConfig{
				TeamDomain: strings.TrimSuffix(strings.TrimPrefix(os.Getenv("CF_ACCESS_TEAM_DOMAIN"), "https://"), "/"),
				Audience:   os.Getenv("CF_ACCESS_AUD"),
			},
			SessionHours: getEnvInt("AUTH_SESSION_HOURS", constants.DefaultSessionHours),
		},
		Updates: UpdateConfig{
			Enabled:       os.Getenv("UI_UPDATES_ENABLED") == "true",
			ManifestURL:   getEnv("UPDATE_MANIFEST_URL", update.DefaultManifestURL),
			PublicKey:     os.Getenv("UPDATE_PUBLIC_KEY"),
			ImagePrefix:   getEnv("UPDATE_IMAGE_REPO_PREFIX", update.DefaultImagePrefix),
			CheckInterval: time.Duration(getEnvInt("UPDATE_CHECK_INTERVAL_HOURS", update.DefaultCheckIntervalHours)) * time.Hour,
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

// generateSecureToken generates a cryptographically secure random token for registration
func generateSecureToken() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to UUID if crypto/rand fails
		return uuid.New().String()
	}
	return base64.URLEncoding.EncodeToString(bytes)
}

// resolveSecurityMode picks the policy mode. An explicit SECURITY_MODE wins. Otherwise a fresh
// install (no database file yet) enforces, an existing install warns until the operator has
// reviewed the audit output and opts in, and development (APP_ENV=development) always warns.
func resolveSecurityMode(databasePath, environment string) (mode, source string) {
	switch strings.ToLower(os.Getenv("SECURITY_MODE")) {
	case constants.SecurityModeEnforce:
		return constants.SecurityModeEnforce, "environment (SECURITY_MODE)"
	case constants.SecurityModeWarn:
		return constants.SecurityModeWarn, "environment (SECURITY_MODE)"
	}
	if environment == constants.EnvDevelopment {
		return constants.SecurityModeWarn, "default for APP_ENV=development (set SECURITY_MODE to pin it)"
	}

	// The first start decides and remembers. Without this a fresh install would enforce on its first
	// start and, because that start creates the database, silently drop to warn on the next one.
	path := filepath.Join(filepath.Dir(databasePath), constants.SecurityModeFile)
	if b, err := os.ReadFile(path); err == nil {
		switch v := strings.TrimSpace(string(b)); v {
		case constants.SecurityModeEnforce, constants.SecurityModeWarn:
			return v, "saved file " + path + " (decided on the first start; set SECURITY_MODE to change it)"
		}
	}
	mode, source = constants.SecurityModeEnforce, "fresh install default"
	if _, err := os.Stat(databasePath); err == nil {
		mode, source = constants.SecurityModeWarn, "existing database default"
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err == nil {
		if err := os.WriteFile(path, []byte(mode+"\n"), 0o644); err != nil {
			slog.Warn("could not save the security mode; it may change on the next start: set SECURITY_MODE", "error", err)
			return mode, source + " (not saved: set SECURITY_MODE)"
		}
	}
	return mode, source + ", saved to " + path
}

// Enforcing reports whether policy violations should be blocked
func (c *Config) Enforcing() bool {
	return c.Security.Mode != constants.SecurityModeWarn
}

// StartupIssue describes a configuration problem found at startup
type StartupIssue struct {
	Fatal   bool
	Message string
	// Advisory marks something that is expected for how this node is deployed. It is still logged at
	// startup, but doctor shows it as a note instead of asking anyone to act.
	Advisory bool
}

// ValidateStartup returns configuration problems. Fatal issues are only reported when enforcing,
// so an existing install keeps starting and surfaces the same problems as warnings.
func (c *Config) ValidateStartup() []StartupIssue {
	var issues []StartupIssue
	add := func(fatal bool, msg string) {
		issues = append(issues, StartupIssue{Fatal: fatal && c.Enforcing(), Message: msg})
	}

	if !c.Auth.Enabled {
		if !c.Security.CloudflareAccess.Enabled() && !c.Security.AllowUnauthenticated && !isLoopbackAddress(c.ServerAddress) {
			if c.Node.IsPrimary {
				add(true, "AUTH_ENABLED=false on a non-loopback address with no Cloudflare Access verification: "+
					"set AUTH_ENABLED=true, or CF_ACCESS_TEAM_DOMAIN and CF_ACCESS_AUD, or ALLOW_UNAUTHENTICATED=true to acknowledge the risk")
			} else {
				// Secondaries are designed to run without user login: the gateway authenticates the
				// user and the network keeps everyone else out. That is a warning, not a startup
				// failure, but the port must never be reachable from outside the private network.
				issues = append(issues, StartupIssue{Advisory: true, Message: "this secondary node serves its API without user authentication and trusts whatever can reach " +
					"its port: keep it on a private network reachable only by the gateway and the primary"})
			}
		}
	} else {
		if len(c.Auth.JWTSecret) < minJWTSecretLength {
			add(true, fmt.Sprintf("JWT_SECRET must be at least %d characters", minJWTSecretLength))
		}
		if c.Auth.GitHub.ClientID == "" || c.Auth.GitHub.ClientSecret == "" {
			add(true, "GITHUB_CLIENT_ID and GITHUB_CLIENT_SECRET are required when AUTH_ENABLED=true")
		}
		if len(c.Auth.GitHub.AllowedUsers) == 0 {
			add(true, "GITHUB_ALLOWED_USERS is empty: every login would be rejected")
		}
		if !c.Auth.SecureCookie {
			add(false, "AUTH_SECURE_COOKIE is false: session cookies will be sent over plain HTTP")
		}
	}
	if !c.Node.IsPrimary && c.Node.PrimaryNodeURL == "" {
		add(false, "NODE_IS_PRIMARY=false but PRIMARY_NODE_URL is empty: this node cannot register itself")
	}
	if c.Node.UsesLink() {
		if c.Node.PrimaryNodeURL == "" {
			add(true, "NODE_TRANSPORT=tunnel needs PRIMARY_NODE_URL: it is the address this node dials out to")
		}
		if !isLoopbackAddress(c.ServerAddress) {
			add(false, "NODE_TRANSPORT=tunnel does not need this node's API to be reachable: bind it to loopback "+
				"(SERVER_ADDRESS=127.0.0.1:8082) so nothing else can call it")
		}
	}
	if c.Node.GatewayAPIKey != "" && len(c.Node.GatewayAPIKey) < minSharedSecretLength {
		add(false, fmt.Sprintf("GATEWAY_API_KEY is shorter than %d characters", minSharedSecretLength))
	}
	return issues
}

const (
	minJWTSecretLength    = 32
	minSharedSecretLength = 24
)

func isLoopbackAddress(addr string) bool {
	host := addr
	if i := strings.LastIndex(addr, ":"); i >= 0 {
		host = addr[:i]
	}
	host = strings.Trim(host, "[]")
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func boolString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return def
}

// loadOrCreatePersisted returns the value stored in dir/name, creating it with gen when absent,
// and says which happened. Persistence is best effort: if the directory is not writable the
// generated value is still used for this process, matching the previous behaviour.
func loadOrCreatePersisted(dir, name string, gen func() string) (value, source string) {
	path := filepath.Join(dir, name)
	if b, err := os.ReadFile(path); err == nil {
		if v := strings.TrimSpace(string(b)); v != "" {
			return v, "saved file " + path
		}
	}
	v := gen()
	if err := os.MkdirAll(dir, 0o755); err == nil {
		if err := os.WriteFile(path, []byte(v+"\n"), constants.SecretFileMode); err != nil {
			slog.Warn("could not persist generated value", "file", path, "error", err)
			return v, "generated this start (could not be saved: it will change on restart)"
		}
		return v, "generated and saved to " + path
	}
	return v, "generated this start (could not be saved: it will change on restart)"
}

// AdoptPrimaryIdentity makes a primary node use the ID already recorded for it in the database
// when nothing pinned one explicitly. This replaces a hard-coded default in the compose file: the
// database is the source of truth for who this node is, and an ID that disagrees with it would
// orphan every app. An explicit NODE_ID always wins (and is reported by `doctor` if it disagrees).
// It reports whether the ID changed. With persist set it also saves the ID next to the database so
// later starts agree; `doctor` passes false because it must not change anything.
func (c *Config) AdoptPrimaryIdentity(primaryID string, persist bool) bool {
	if !c.Node.IsPrimary || primaryID == "" || primaryID == c.Node.ID {
		return false
	}
	if strings.HasPrefix(c.Sources["node_id"], "environment") {
		return false
	}
	c.Node.ID = primaryID
	c.Sources["node_id"] = "adopted from the database's primary node"
	if persist {
		dir := filepath.Dir(c.DatabasePath)
		if err := os.MkdirAll(dir, 0o755); err == nil {
			if err := os.WriteFile(filepath.Join(dir, constants.NodeIDFileName), []byte(primaryID+"\n"), constants.SecretFileMode); err != nil {
				slog.Warn("could not save the adopted node id", "error", err)
			}
		}
	}
	return true
}

// resolveNodeTransport accepts "tunnel" or "direct"; anything else (including unset) is direct, so an
// existing secondary keeps working exactly as before.
func resolveNodeTransport(v string) string {
	if strings.EqualFold(strings.TrimSpace(v), constants.NodeTransportTunnel) {
		return constants.NodeTransportTunnel
	}
	return constants.NodeTransportDirect
}

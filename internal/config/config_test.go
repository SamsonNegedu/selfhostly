package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	t.Chdir(t.TempDir())
	// Store original env vars
	origServerAddr := os.Getenv("SERVER_ADDRESS")
	origDbPath := os.Getenv("DATABASE_PATH")
	origAppsDir := os.Getenv("APPS_DIR")
	origCORSOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
	origAutoStart := os.Getenv("AUTO_START_APPS")
	origCloudflareToken := os.Getenv("CLOUDFLARE_API_TOKEN")
	origCloudflareAccount := os.Getenv("CLOUDFLARE_ACCOUNT_ID")
	origAuthEnabled := os.Getenv("AUTH_ENABLED")
	origSecureCookie := os.Getenv("AUTH_SECURE_COOKIE")
	origBaseURL := os.Getenv("NODE_API_ENDPOINT")
	origGitHubClientID := os.Getenv("GITHUB_CLIENT_ID")
	origGitHubSecret := os.Getenv("GITHUB_CLIENT_SECRET")

	// Restore env vars after test
	defer func() {
		os.Setenv("SERVER_ADDRESS", origServerAddr)
		os.Setenv("DATABASE_PATH", origDbPath)
		os.Setenv("APPS_DIR", origAppsDir)
		os.Setenv("CORS_ALLOWED_ORIGINS", origCORSOrigins)
		os.Setenv("AUTO_START_APPS", origAutoStart)
		os.Setenv("CLOUDFLARE_API_TOKEN", origCloudflareToken)
		os.Setenv("CLOUDFLARE_ACCOUNT_ID", origCloudflareAccount)
		os.Setenv("AUTH_ENABLED", origAuthEnabled)
		os.Setenv("AUTH_SECURE_COOKIE", origSecureCookie)
		os.Setenv("NODE_API_ENDPOINT", origBaseURL)
		os.Setenv("GITHUB_CLIENT_ID", origGitHubClientID)
		os.Setenv("GITHUB_CLIENT_SECRET", origGitHubSecret)
	}()

	// Clear env vars
	os.Unsetenv("SERVER_ADDRESS")
	os.Unsetenv("DATABASE_PATH")
	os.Unsetenv("APPS_DIR")
	os.Unsetenv("CORS_ALLOWED_ORIGINS")
	os.Unsetenv("AUTO_START_APPS")
	os.Unsetenv("CLOUDFLARE_API_TOKEN")
	os.Unsetenv("CLOUDFLARE_ACCOUNT_ID")
	os.Unsetenv("AUTH_ENABLED")
	os.Unsetenv("AUTH_SECURE_COOKIE")
	os.Unsetenv("NODE_API_ENDPOINT")
	os.Unsetenv("GITHUB_CLIENT_ID")
	os.Unsetenv("GITHUB_CLIENT_SECRET")

	// Test with default values
	config, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Check default values
	if config.ServerAddress != ":8080" {
		t.Errorf("Expected ServerAddress to be :8080, got %s", config.ServerAddress)
	}

	if config.DatabasePath != "./data/selfhostly.db" {
		t.Errorf("Expected DatabasePath to be ./data/selfhostly.db, got %s", config.DatabasePath)
	}

	if config.AppsDir != "./apps" {
		t.Errorf("Expected AppsDir to be ./apps, got %s", config.AppsDir)
	}

	if config.AutoStart != false {
		t.Errorf("Expected AutoStart to be false, got %v", config.AutoStart)
	}

	if config.Auth.Enabled != false {
		t.Errorf("Expected Auth.Enabled to be false, got %v", config.Auth.Enabled)
	}

	if config.Auth.JWTSecret != "" {
		t.Errorf("Expected JWTSecret to be empty when auth disabled, got %s", config.Auth.JWTSecret)
	}

	if config.Auth.SecureCookie != false {
		t.Errorf("Expected SecureCookie to be false, got %v", config.Auth.SecureCookie)
	}

	if config.Auth.BaseURL != "http://localhost:8080" {
		t.Errorf("Expected BaseURL to be 'http://localhost:8080', got %s", config.Auth.BaseURL)
	}

	// Check default CORS origins
	expectedOrigins := []string{"http://localhost:5173", "http://localhost:3000", "http://localhost:8080"}
	if len(config.CORS.AllowedOrigins) != len(expectedOrigins) {
		t.Errorf("Expected %d CORS origins, got %d", len(expectedOrigins), len(config.CORS.AllowedOrigins))
	}

	for i, origin := range expectedOrigins {
		if config.CORS.AllowedOrigins[i] != origin {
			t.Errorf("Expected CORS origin %s at index %d, got %s", origin, i, config.CORS.AllowedOrigins[i])
		}
	}
}

func TestLoadWithCustomEnv(t *testing.T) {
	t.Chdir(t.TempDir())
	// Store original env vars
	origServerAddr := os.Getenv("SERVER_ADDRESS")
	origDbPath := os.Getenv("DATABASE_PATH")
	origAppsDir := os.Getenv("APPS_DIR")
	origCORSOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
	origAutoStart := os.Getenv("AUTO_START_APPS")
	origCloudflareToken := os.Getenv("CLOUDFLARE_API_TOKEN")
	origCloudflareAccount := os.Getenv("CLOUDFLARE_ACCOUNT_ID")
	origAuthEnabled := os.Getenv("AUTH_ENABLED")
	origJWTSecret := os.Getenv("JWT_SECRET")
	origSecureCookie := os.Getenv("AUTH_SECURE_COOKIE")
	origBaseURL := os.Getenv("NODE_API_ENDPOINT")
	origGitHubClientID := os.Getenv("GITHUB_CLIENT_ID")
	origGitHubSecret := os.Getenv("GITHUB_CLIENT_SECRET")

	// Restore env vars after test
	defer func() {
		os.Setenv("SERVER_ADDRESS", origServerAddr)
		os.Setenv("DATABASE_PATH", origDbPath)
		os.Setenv("APPS_DIR", origAppsDir)
		os.Setenv("CORS_ALLOWED_ORIGINS", origCORSOrigins)
		os.Setenv("AUTO_START_APPS", origAutoStart)
		os.Setenv("CLOUDFLARE_API_TOKEN", origCloudflareToken)
		os.Setenv("CLOUDFLARE_ACCOUNT_ID", origCloudflareAccount)
		os.Setenv("AUTH_ENABLED", origAuthEnabled)
		os.Setenv("JWT_SECRET", origJWTSecret)
		os.Setenv("AUTH_SECURE_COOKIE", origSecureCookie)
		os.Setenv("NODE_API_ENDPOINT", origBaseURL)
		os.Setenv("GITHUB_CLIENT_ID", origGitHubClientID)
		os.Setenv("GITHUB_CLIENT_SECRET", origGitHubSecret)
	}()

	// Set custom env vars
	os.Setenv("SERVER_ADDRESS", ":9000")
	os.Setenv("DATABASE_PATH", "/custom/path/db.sqlite")
	os.Setenv("APPS_DIR", "/custom/apps")
	os.Setenv("CORS_ALLOWED_ORIGINS", "https://example.com,https://app.example.com")
	os.Setenv("AUTO_START_APPS", "true")
	os.Setenv("CLOUDFLARE_API_TOKEN", "test-token")
	os.Setenv("CLOUDFLARE_ACCOUNT_ID", "test-account")
	os.Setenv("AUTH_ENABLED", "true")
	os.Setenv("JWT_SECRET", "custom-secret")
	os.Setenv("AUTH_SECURE_COOKIE", "true")
	os.Setenv("NODE_API_ENDPOINT", "https://example.com")
	os.Setenv("GITHUB_CLIENT_ID", "test-client-id")
	os.Setenv("GITHUB_CLIENT_SECRET", "test-client-secret")

	// Load config with custom env
	config, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Check custom values
	if config.ServerAddress != ":9000" {
		t.Errorf("Expected ServerAddress to be :9000, got %s", config.ServerAddress)
	}

	if config.DatabasePath != "/custom/path/db.sqlite" {
		t.Errorf("Expected DatabasePath to be /custom/path/db.sqlite, got %s", config.DatabasePath)
	}

	if config.AppsDir != "/custom/apps" {
		t.Errorf("Expected AppsDir to be /custom/apps, got %s", config.AppsDir)
	}

	if config.AutoStart != true {
		t.Errorf("Expected AutoStart to be true, got %v", config.AutoStart)
	}

	if config.Cloudflare.APIToken != "test-token" {
		t.Errorf("Expected Cloudflare.APIToken to be 'test-token', got %s", config.Cloudflare.APIToken)
	}

	if config.Cloudflare.AccountID != "test-account" {
		t.Errorf("Expected Cloudflare.AccountID to be 'test-account', got %s", config.Cloudflare.AccountID)
	}

	if config.Auth.Enabled != true {
		t.Errorf("Expected Auth.Enabled to be true, got %v", config.Auth.Enabled)
	}

	if config.Auth.JWTSecret != "custom-secret" {
		t.Errorf("Expected JWTSecret to be 'custom-secret', got %s", config.Auth.JWTSecret)
	}

	if config.Auth.SecureCookie != true {
		t.Errorf("Expected SecureCookie to be true, got %v", config.Auth.SecureCookie)
	}

	if config.Auth.BaseURL != "https://example.com" {
		t.Errorf("Expected BaseURL to be 'https://example.com', got %s", config.Auth.BaseURL)
	}

	if config.Auth.GitHub.ClientID != "test-client-id" {
		t.Errorf("Expected GitHub.ClientID to be 'test-client-id', got %s", config.Auth.GitHub.ClientID)
	}

	if config.Auth.GitHub.ClientSecret != "test-client-secret" {
		t.Errorf("Expected GitHub.ClientSecret to be 'test-client-secret', got %s", config.Auth.GitHub.ClientSecret)
	}

	// Check custom CORS origins
	expectedOrigins := []string{"https://example.com", "https://app.example.com"}
	if len(config.CORS.AllowedOrigins) != len(expectedOrigins) {
		t.Errorf("Expected %d CORS origins, got %d", len(expectedOrigins), len(config.CORS.AllowedOrigins))
	}

	for i, origin := range expectedOrigins {
		if config.CORS.AllowedOrigins[i] != origin {
			t.Errorf("Expected CORS origin %s at index %d, got %s", origin, i, config.CORS.AllowedOrigins[i])
		}
	}
}

func TestParseCommaSeparatedList(t *testing.T) {
	t.Chdir(t.TempDir())
	// Test with normal comma-separated list
	list := parseCommaSeparatedList("a,b,c")
	expected := []string{"a", "b", "c"}
	if len(list) != len(expected) {
		t.Errorf("Expected %d items, got %d", len(expected), len(list))
	}
	for i, item := range expected {
		if list[i] != item {
			t.Errorf("Expected item %s at index %d, got %s", item, i, list[i])
		}
	}

	// Test with whitespace
	list = parseCommaSeparatedList("a, b , c ")
	if len(list) != len(expected) {
		t.Errorf("Expected %d items, got %d", len(expected), len(list))
	}
	for i, item := range expected {
		if list[i] != item {
			t.Errorf("Expected item %s at index %d, got %s", item, i, list[i])
		}
	}

	// Test with empty string
	list = parseCommaSeparatedList("")
	if len(list) != 0 {
		t.Errorf("Expected 0 items for empty string, got %d", len(list))
	}

	// Test with empty items
	list = parseCommaSeparatedList("a,,c")
	expected = []string{"a", "c"}
	if len(list) != len(expected) {
		t.Errorf("Expected %d items, got %d", len(expected), len(list))
	}
	for i, item := range expected {
		if list[i] != item {
			t.Errorf("Expected item %s at index %d, got %s", item, i, list[i])
		}
	}

	// Test with only empty items
	list = parseCommaSeparatedList(",,")
	if len(list) != 0 {
		t.Errorf("Expected 0 items for string with only empty items, got %d", len(list))
	}

	// Test with single item
	list = parseCommaSeparatedList("single")
	expected = []string{"single"}
	if len(list) != len(expected) {
		t.Errorf("Expected %d items, got %d", len(expected), len(list))
	}
	if list[0] != expected[0] {
		t.Errorf("Expected item %s, got %s", expected[0], list[0])
	}
}

func TestGetEnv(t *testing.T) {
	t.Chdir(t.TempDir())
	// Test with existing env var
	key := "TEST_GET_ENV"
	value := "test-value"
	os.Setenv(key, value)

	result := getEnv(key, "default")
	if result != value {
		t.Errorf("Expected %s, got %s", value, result)
	}

	// Clean up
	os.Unsetenv(key)

	// Test with non-existing env var
	result = getEnv(key, "default")
	if result != "default" {
		t.Errorf("Expected 'default', got %s", result)
	}

	// Test with empty env var
	os.Setenv(key, "")
	result = getEnv(key, "default")
	if result != "default" {
		t.Errorf("Expected 'default' for empty env var, got %s", result)
	}

	// Clean up
	os.Unsetenv(key)
}

func TestLoadWithAuthEnabledButNoJWTSecret(t *testing.T) {
	t.Chdir(t.TempDir())
	// Store original env vars
	origAuthEnabled := os.Getenv("AUTH_ENABLED")
	origJWTSecret := os.Getenv("JWT_SECRET")

	// Restore env vars after test
	defer func() {
		os.Setenv("AUTH_ENABLED", origAuthEnabled)
		os.Setenv("JWT_SECRET", origJWTSecret)
	}()

	// Enable auth but don't set JWT secret
	os.Setenv("AUTH_ENABLED", "true")
	os.Unsetenv("JWT_SECRET")

	// Should return an error
	_, err := Load()
	if err == nil {
		t.Fatal("Expected error when AUTH_ENABLED is true but JWT_SECRET is not set, got nil")
	}

	expectedError := "JWT_SECRET environment variable is required when AUTH_ENABLED is true"
	if err.Error() != expectedError {
		t.Errorf("Expected error message '%s', got '%s'", expectedError, err.Error())
	}
}

func TestSecurityModeResolution(t *testing.T) {
	fresh := func(t *testing.T) string { return t.TempDir() + "/selfhostly.db" }
	existing := func(t *testing.T) string {
		p := t.TempDir() + "/selfhostly.db"
		if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}

	t.Setenv("SECURITY_MODE", "")
	if got := resolveSecurityModeOnly(fresh(t), "production"); got != "enforce" {
		t.Errorf("a fresh install must enforce, got %q", got)
	}
	if got := resolveSecurityModeOnly(existing(t), "production"); got != "warn" {
		t.Errorf("an existing install must warn until opted in, got %q", got)
	}
	if got := resolveSecurityModeOnly(fresh(t), "development"); got != "warn" {
		t.Errorf("development must default to warn so local runs need no setup, got %q", got)
	}
	t.Setenv("SECURITY_MODE", "ENFORCE")
	if got := resolveSecurityModeOnly(existing(t), "production"); got != "enforce" {
		t.Errorf("an explicit value must win (case-insensitive), got %q", got)
	}
	t.Setenv("SECURITY_MODE", "warn")
	if got := resolveSecurityModeOnly(fresh(t), "production"); got != "warn" {
		t.Errorf("an explicit warn must win on a fresh install, got %q", got)
	}
}

func TestDerivedSecurityModeIsRememberedAcrossStarts(t *testing.T) {
	t.Setenv("SECURITY_MODE", "")
	db := t.TempDir() + "/selfhostly.db"

	first, src := resolveSecurityMode(db, "production")
	if first != "enforce" {
		t.Fatalf("fresh install must enforce, got %q", first)
	}
	// The first start creates the database. Without remembering, the next start would see an
	// existing database and quietly drop to warn.
	if err := os.WriteFile(db, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	second, src2 := resolveSecurityMode(db, "production")
	if second != "enforce" {
		t.Fatalf("the mode must not change between starts, got %q (%s then %s)", second, src, src2)
	}
	if !strings.Contains(src2, "saved file") {
		t.Fatalf("the source must say it was remembered, got %q", src2)
	}

	t.Setenv("SECURITY_MODE", "warn")
	if got, _ := resolveSecurityMode(db, "production"); got != "warn" {
		t.Fatal("an explicit SECURITY_MODE must override the remembered value")
	}
}

func TestValidateStartup(t *testing.T) {
	base := func() *Config {
		return &Config{
			ServerAddress: ":8080",
			Security:      SecurityConfig{Mode: "enforce"},
			Node:          NodeConfig{IsPrimary: true},
		}
	}
	fatal := func(c *Config) bool {
		for _, i := range c.ValidateStartup() {
			if i.Fatal {
				return true
			}
		}
		return false
	}

	c := base()
	if !fatal(c) {
		t.Error("no auth on a public bind must be fatal when enforcing")
	}
	c.ServerAddress = "127.0.0.1:8080"
	if fatal(c) {
		t.Error("no auth on loopback is acceptable")
	}
	c = base()
	c.Node.IsPrimary = false
	c.Node.PrimaryNodeURL = "http://primary:8082"
	if fatal(c) {
		t.Error("a secondary is designed to run without user login and must still start")
	}
	if len(c.ValidateStartup()) == 0 {
		t.Error("a secondary without user auth must still be warned about")
	}
	c = base()
	c.Security.AllowUnauthenticated = true
	if fatal(c) {
		t.Error("ALLOW_UNAUTHENTICATED acknowledges the risk")
	}
	c = base()
	c.Security.CloudflareAccess = CloudflareAccessConfig{TeamDomain: "t.cloudflareaccess.com", Audience: "aud"}
	if fatal(c) {
		t.Error("verified Cloudflare Access is an acceptable auth mode")
	}
	c = base()
	c.Security.Mode = "warn"
	if fatal(c) {
		t.Error("warn mode must report the same problem without refusing to start")
	}
	if len(c.ValidateStartup()) == 0 {
		t.Error("warn mode must still surface the problem")
	}

	c = base()
	c.Auth = AuthConfig{Enabled: true, JWTSecret: "short", SecureCookie: true}
	issues := c.ValidateStartup()
	if len(issues) < 3 {
		t.Errorf("weak secret, missing OAuth credentials and empty allow-list must each be reported, got %d", len(issues))
	}
}

func TestPersistedIdentityIsStable(t *testing.T) {
	t.Setenv("NODE_ID", "auto")
	t.Setenv("NODE_API_KEY", "")
	t.Setenv("REGISTRATION_TOKEN", "")
	t.Setenv("NODE_IS_PRIMARY", "true")
	t.Setenv("DATABASE_PATH", t.TempDir()+"/db/selfhostly.db")

	first, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	second, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if first.Node.ID != second.Node.ID || first.Node.APIKey != second.Node.APIKey || first.Node.RegistrationToken != second.Node.RegistrationToken {
		t.Fatal("generated identity must not change between restarts")
	}

	t.Setenv("NODE_ID", "explicit-id")
	if c, _ := Load(); c.Node.ID != "explicit-id" {
		t.Fatal("an explicit NODE_ID must win over the persisted one")
	}
}

func TestSecureCookieDefaultsFromBaseURL(t *testing.T) {
	t.Setenv("DATABASE_PATH", t.TempDir()+"/selfhostly.db")
	t.Setenv("AUTH_SECURE_COOKIE", "")
	t.Setenv("AUTH_BASE_URL", "https://selfhostly.example.com")
	if c, _ := Load(); !c.Auth.SecureCookie {
		t.Error("an https public URL must default to secure cookies")
	}
	t.Setenv("AUTH_BASE_URL", "http://localhost:8080")
	if c, _ := Load(); c.Auth.SecureCookie {
		t.Error("a plain http URL must not force secure cookies (they would never be sent)")
	}
	t.Setenv("AUTH_SECURE_COOKIE", "true")
	if c, _ := Load(); !c.Auth.SecureCookie {
		t.Error("an explicit value must win")
	}
}

func resolveSecurityModeOnly(path, env string) string {
	mode, _ := resolveSecurityMode(path, env)
	return mode
}

func TestEncryptionIsNotImpliedBySecurityMode(t *testing.T) {
	t.Setenv("DATABASE_PATH", t.TempDir()+"/selfhostly.db")
	t.Setenv("ENCRYPT_SECRETS_AT_REST", "")
	t.Setenv("SECURITY_MODE", "enforce")
	if c, _ := Load(); c.Security.EncryptSecretsAtRest {
		t.Fatal("switching to enforce must not silently start encrypting stored secrets")
	}
	t.Setenv("ENCRYPT_SECRETS_AT_REST", "true")
	if c, _ := Load(); !c.Security.EncryptSecretsAtRest {
		t.Fatal("an explicit ENCRYPT_SECRETS_AT_REST=true must turn it on")
	}
}

func TestSourcesExplainEveryImplicitValue(t *testing.T) {
	t.Setenv("DATABASE_PATH", t.TempDir()+"/selfhostly.db")
	t.Setenv("NODE_ID", "")
	t.Setenv("NODE_API_KEY", "")
	t.Setenv("SECURITY_MODE", "")
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"security_mode", "node_id", "node_api_key"} {
		if c.Sources[k] == "" {
			t.Errorf("no source recorded for %s", k)
		}
	}
	t.Setenv("NODE_ID", "explicit")
	if c, _ := Load(); c.Sources["node_id"] != "environment (NODE_ID)" {
		t.Errorf("an explicit value must be reported as such, got %q", c.Sources["node_id"])
	}
}

func TestAdoptPrimaryIdentity(t *testing.T) {
	newCfg := func(t *testing.T, primary bool, idSource string) *Config {
		return &Config{
			DatabasePath: t.TempDir() + "/selfhostly.db",
			Node:         NodeConfig{ID: "generated-id", IsPrimary: primary},
			Sources:      map[string]string{"node_id": idSource},
		}
	}

	c := newCfg(t, true, "generated and saved to x")
	if !c.AdoptPrimaryIdentity("db-id", true) || c.Node.ID != "db-id" {
		t.Fatal("an unpinned primary must adopt the database's ID")
	}
	if b, _ := os.ReadFile(filepath.Join(filepath.Dir(c.DatabasePath), "node-id")); !strings.Contains(string(b), "db-id") {
		t.Fatal("the adopted ID must be saved so later starts agree")
	}

	c = newCfg(t, true, "environment (NODE_ID)")
	if c.AdoptPrimaryIdentity("db-id", true) || c.Node.ID != "generated-id" {
		t.Fatal("an explicit NODE_ID must never be overridden")
	}
	c = newCfg(t, false, "generated and saved to x")
	if c.AdoptPrimaryIdentity("db-id", true) {
		t.Fatal("only a primary adopts the primary's ID")
	}
	c = newCfg(t, true, "generated and saved to x")
	if c.AdoptPrimaryIdentity("", true) || c.AdoptPrimaryIdentity("generated-id", true) {
		t.Fatal("nothing to adopt when the database has no ID or it already matches")
	}

	c = newCfg(t, true, "generated and saved to x")
	c.AdoptPrimaryIdentity("db-id", false)
	if _, err := os.Stat(filepath.Join(filepath.Dir(c.DatabasePath), "node-id")); err == nil {
		t.Fatal("persist=false (doctor) must not write anything")
	}
}

func TestNodeTransport(t *testing.T) {
	t.Setenv("DATABASE_PATH", t.TempDir()+"/selfhostly.db")
	t.Setenv("NODE_IS_PRIMARY", "false")
	t.Setenv("PRIMARY_NODE_URL", "https://selfhostly.example.com")

	t.Setenv("NODE_TRANSPORT", "")
	if c, _ := Load(); c.Node.Transport != "direct" || c.Node.UsesLink() {
		t.Fatal("an existing secondary with no setting must stay direct")
	}
	t.Setenv("NODE_TRANSPORT", " TUNNEL ")
	if c, _ := Load(); c.Node.Transport != "tunnel" || !c.Node.UsesLink() {
		t.Fatal("tunnel must be recognised regardless of case and spaces")
	}
	t.Setenv("NODE_TRANSPORT", "carrier-pigeon")
	if c, _ := Load(); c.Node.Transport != "direct" {
		t.Fatal("an unknown value must fall back to direct, not break startup")
	}

	t.Setenv("NODE_TRANSPORT", "tunnel")
	t.Setenv("NODE_IS_PRIMARY", "true")
	if c, _ := Load(); c.Node.UsesLink() {
		t.Fatal("a primary never uses a link to reach itself")
	}
}

func TestTunnelModeStartupChecks(t *testing.T) {
	base := func() *Config {
		return &Config{
			ServerAddress: "127.0.0.1:8082",
			Security:      SecurityConfig{Mode: "enforce"},
			Node:          NodeConfig{IsPrimary: false, Transport: "tunnel", PrimaryNodeURL: "https://p.example.com"},
		}
	}
	messages := func(c *Config) (fatal bool, joined string) {
		for _, i := range c.ValidateStartup() {
			fatal = fatal || i.Fatal
			joined += i.Message + "|"
		}
		return
	}

	if fatal, msg := messages(base()); fatal || msg != "" {
		t.Fatalf("a correctly configured tunnel node has nothing to report, got %q", msg)
	}
	c := base()
	c.Node.PrimaryNodeURL = ""
	if fatal, _ := messages(c); !fatal {
		t.Fatal("a tunnel node with nowhere to dial is a fatal misconfiguration")
	}
	c = base()
	c.ServerAddress = ":8082"
	fatal, msg := messages(c)
	if fatal || !strings.Contains(msg, "loopback") {
		t.Fatalf("a tunnel node on a public address should be told to bind loopback, not refused: %v %q", fatal, msg)
	}
}

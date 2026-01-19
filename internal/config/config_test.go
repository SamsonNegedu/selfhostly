package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
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
	origCookieDomain := os.Getenv("AUTH_COOKIE_DOMAIN")
	origSecureCookie := os.Getenv("AUTH_SECURE_COOKIE")
	origBaseURL := os.Getenv("AUTH_BASE_URL")
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
		os.Setenv("AUTH_COOKIE_DOMAIN", origCookieDomain)
		os.Setenv("AUTH_SECURE_COOKIE", origSecureCookie)
		os.Setenv("AUTH_BASE_URL", origBaseURL)
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
	os.Unsetenv("JWT_SECRET")
	os.Unsetenv("AUTH_COOKIE_DOMAIN")
	os.Unsetenv("AUTH_SECURE_COOKIE")
	os.Unsetenv("AUTH_BASE_URL")
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
	
	if config.DatabasePath != "./data/automaton.db" {
		t.Errorf("Expected DatabasePath to be ./data/automaton.db, got %s", config.DatabasePath)
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
	
	if config.Auth.JWTSecret != "change-me-in-production-secret-key" {
		t.Errorf("Expected JWTSecret to be 'change-me-in-production-secret-key', got %s", config.Auth.JWTSecret)
	}
	
	if config.Auth.CookieDomain != "localhost" {
		t.Errorf("Expected CookieDomain to be 'localhost', got %s", config.Auth.CookieDomain)
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
	origCookieDomain := os.Getenv("AUTH_COOKIE_DOMAIN")
	origSecureCookie := os.Getenv("AUTH_SECURE_COOKIE")
	origBaseURL := os.Getenv("AUTH_BASE_URL")
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
		os.Setenv("AUTH_COOKIE_DOMAIN", origCookieDomain)
		os.Setenv("AUTH_SECURE_COOKIE", origSecureCookie)
		os.Setenv("AUTH_BASE_URL", origBaseURL)
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
	os.Setenv("AUTH_COOKIE_DOMAIN", "example.com")
	os.Setenv("AUTH_SECURE_COOKIE", "true")
	os.Setenv("AUTH_BASE_URL", "https://example.com")
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
	
	if config.Auth.CookieDomain != "example.com" {
		t.Errorf("Expected CookieDomain to be 'example.com', got %s", config.Auth.CookieDomain)
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
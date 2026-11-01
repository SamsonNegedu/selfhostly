package docker

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/selfhostly/internal/tunnel"
)

func TestParseCompose(t *testing.T) {
	// Valid compose file
	validCompose := `
version: "3.8"
services:
  web:
    image: nginx:latest
    ports:
      - "8080:80"
networks:
  frontend:
    driver: bridge
volumes:
  data:
`

	compose, err := ParseCompose([]byte(validCompose))
	if err != nil {
		t.Fatalf("Failed to parse valid compose: %v", err)
	}

	// compose-go treats 'version' as obsolete and does not populate it
	// This is consistent with Docker Compose v2+ behavior

	if len(compose.Services) != 1 {
		t.Errorf("Expected 1 service, got %d", len(compose.Services))
	}

	// compose-go normalization adds a 'default' network, so we expect at least
	// the 'frontend' network plus the implicit 'default' network
	if len(compose.Networks) < 1 {
		t.Errorf("Expected at least 1 network, got %d", len(compose.Networks))
	}

	if len(compose.Volumes) != 1 {
		t.Errorf("Expected 1 volume, got %d", len(compose.Volumes))
	}

	// Check service details
	webService, exists := compose.Services["web"]
	if !exists {
		t.Fatalf("Expected web service to exist")
	}

	if webService.Image != "nginx:latest" {
		t.Errorf("Expected image nginx:latest, got %s", webService.Image)
	}

	if len(webService.Ports) != 1 || webService.Ports[0] != "8080:80" {
		t.Errorf("Expected ports [8080:80], got %v", webService.Ports)
	}

	// Check network details
	frontendNetwork, exists := compose.Networks["frontend"]
	if !exists {
		t.Fatalf("Expected frontend network to exist")
	}

	if frontendNetwork.Driver != "bridge" {
		t.Errorf("Expected bridge driver, got %s", frontendNetwork.Driver)
	}

	// Check volume details
	_, exists = compose.Volumes["data"]
	if !exists {
		t.Fatalf("Expected data volume to exist")
	}
}

func TestParseComposeNoServices(t *testing.T) {
	// Compose file with no services
	invalidCompose := `
version: "3.8"
networks:
  frontend:
    driver: bridge
`

	_, err := ParseCompose([]byte(invalidCompose))
	if err == nil {
		t.Error("Expected error when parsing compose with no services")
	}
}

func TestParseComposeInvalidYAML(t *testing.T) {
	// Invalid YAML
	invalidCompose := `
version: "3.8"
services:
  web:
    image: nginx:latest
    ports:
      - "8080:80"
    invalid_yaml: [
`

	_, err := ParseCompose([]byte(invalidCompose))
	if err == nil {
		t.Error("Expected error when parsing invalid YAML")
	}
	
	// Verify it's a ComposeParseError with helpful information
	var parseErr *ComposeParseError
	if !errors.As(err, &parseErr) {
		t.Error("Expected ComposeParseError type")
	} else {
		if parseErr.Message == "" {
			t.Error("Expected non-empty error message")
		}
		if parseErr.Suggestion == "" {
			t.Error("Expected non-empty suggestion")
		}
	}
}

func TestParseComposeErrorMessages(t *testing.T) {
	tests := []struct {
		name          string
		compose       string
		expectLine    bool
		expectSuggestion bool
		errorContains string
	}{
		{
			name: "invalid depends_on type",
			compose: `
services:
  web:
    image: nginx
    depends_on: 123
`,
			expectLine: false,
			expectSuggestion: true,
			errorContains: "",
		},
		{
			name: "no services",
			compose: `
version: "3.8"
networks:
  frontend: {}
`,
			expectLine: false,
			expectSuggestion: true,
			errorContains: "no services",
		},
		{
			name: "invalid YAML syntax",
			compose: `
services:
  web:
    image: nginx
    ports: [invalid
`,
			expectLine: true,
			expectSuggestion: true,
			errorContains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseCompose([]byte(tt.compose))
			if err == nil {
				t.Error("Expected error but got none")
				return
			}

			var parseErr *ComposeParseError
			if !errors.As(err, &parseErr) {
				t.Errorf("Expected ComposeParseError, got %T: %v", err, err)
				return
			}

			if tt.expectLine && parseErr.Line == 0 {
				t.Error("Expected line number but got 0")
			}

			if tt.expectSuggestion && parseErr.Suggestion == "" {
				t.Error("Expected suggestion but got empty string")
			}

			if tt.errorContains != "" && !strings.Contains(parseErr.Message, tt.errorContains) {
				t.Errorf("Expected error message to contain %q, got %q", tt.errorContains, parseErr.Message)
			}

			// Verify error message is helpful
			errStr := parseErr.Error()
			if errStr == "" {
				t.Error("Error string should not be empty")
			}
		})
	}
}

func TestParseComposeDependsOnWithConditions(t *testing.T) {
	// Test depends_on with conditions (Docker Compose v3.9+ syntax)
	composeWithConditions := `
services:
  kanba:
    image: nginx:latest
    depends_on:
      postgres:
        condition: service_healthy
  postgres:
    image: postgres:15-alpine
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 10s
      timeout: 5s
      retries: 5
`

	compose, err := ParseCompose([]byte(composeWithConditions))
	if err != nil {
		t.Fatalf("Failed to parse compose with depends_on conditions: %v", err)
	}

	kanbaService, exists := compose.Services["kanba"]
	if !exists {
		t.Fatal("Expected kanba service to exist")
	}

	// Check that depends_on was parsed correctly
	if len(kanbaService.DependsOn.WithConditions) == 0 {
		t.Error("Expected depends_on to be parsed with conditions")
	}

	postgresDep, exists := kanbaService.DependsOn.WithConditions["postgres"]
	if !exists {
		t.Error("Expected postgres dependency to exist")
	}

	if postgresDep.Condition != "service_healthy" {
		t.Errorf("Expected condition 'service_healthy', got '%s'", postgresDep.Condition)
	}
}

func TestParseComposeDependsOnSimpleList(t *testing.T) {
	// Test depends_on as simple list (older syntax)
	composeWithList := `
services:
  web:
    image: nginx:latest
    depends_on:
      - db
      - redis
  db:
    image: postgres:latest
  redis:
    image: redis:latest
`

	compose, err := ParseCompose([]byte(composeWithList))
	if err != nil {
		t.Fatalf("Failed to parse compose with depends_on list: %v", err)
	}

	webService, exists := compose.Services["web"]
	if !exists {
		t.Fatal("Expected web service to exist")
	}

	// Check that depends_on was parsed as a list
	if len(webService.DependsOn.Services) == 0 {
		t.Error("Expected depends_on to be parsed as a list")
	}

	expectedDeps := []string{"db", "redis"}
	if len(webService.DependsOn.Services) != len(expectedDeps) {
		t.Errorf("Expected %d dependencies, got %d", len(expectedDeps), len(webService.DependsOn.Services))
	}

	for _, expectedDep := range expectedDeps {
		found := false
		for _, dep := range webService.DependsOn.Services {
			if dep == expectedDep {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected dependency '%s' not found", expectedDep)
		}
	}
}

func TestParseComposeBuildArgsListFormat(t *testing.T) {
	// Test build args as list format (KEY=value)
	composeWithListArgs := `
services:
  app:
    build:
      context: .
      args:
        - NEXT_PUBLIC_SUPABASE_URL=https://dummy.supabase.co
        - NEXT_PUBLIC_SUPABASE_ANON_KEY=dummy_key
        - DATABASE_URL=postgresql://user:pass@postgres:5432/db
    image: myapp:latest
`

	compose, err := ParseCompose([]byte(composeWithListArgs))
	if err != nil {
		t.Fatalf("Failed to parse compose with build args list: %v", err)
	}

	appService, exists := compose.Services["app"]
	if !exists {
		t.Fatal("Expected app service to exist")
	}

	// Check that args were parsed correctly
	if len(appService.Build.Args) == 0 {
		t.Error("Expected build args to be parsed")
	}

	expectedArgs := map[string]string{
		"NEXT_PUBLIC_SUPABASE_URL":    "https://dummy.supabase.co",
		"NEXT_PUBLIC_SUPABASE_ANON_KEY": "dummy_key",
		"DATABASE_URL":                "postgresql://user:pass@postgres:5432/db",
	}

	if len(appService.Build.Args) != len(expectedArgs) {
		t.Errorf("Expected %d args, got %d", len(expectedArgs), len(appService.Build.Args))
	}

	for key, expectedValue := range expectedArgs {
		actualValue, exists := appService.Build.Args[key]
		if !exists {
			t.Errorf("Expected arg '%s' not found", key)
			continue
		}
		if actualValue != expectedValue {
			t.Errorf("Expected arg '%s' to be '%s', got '%s'", key, expectedValue, actualValue)
		}
	}
}

func TestParseComposeBuildArgsMapFormat(t *testing.T) {
	// Test build args as map format (KEY: value)
	composeWithMapArgs := `
services:
  app:
    build:
      context: .
      args:
        BUILD_VERSION: ${APP_VERSION:-latest}
        NODE_ENV: production
    image: myapp:latest
`

	compose, err := ParseCompose([]byte(composeWithMapArgs))
	if err != nil {
		t.Fatalf("Failed to parse compose with build args map: %v", err)
	}

	appService, exists := compose.Services["app"]
	if !exists {
		t.Fatal("Expected app service to exist")
	}

	// Check that args were parsed correctly
	if len(appService.Build.Args) == 0 {
		t.Error("Expected build args to be parsed")
	}

	expectedArgs := map[string]string{
		"BUILD_VERSION": "${APP_VERSION:-latest}",
		"NODE_ENV":      "production",
	}

	if len(appService.Build.Args) != len(expectedArgs) {
		t.Errorf("Expected %d args, got %d", len(expectedArgs), len(appService.Build.Args))
	}

	for key, expectedValue := range expectedArgs {
		actualValue, exists := appService.Build.Args[key]
		if !exists {
			t.Errorf("Expected arg '%s' not found", key)
			continue
		}
		if actualValue != expectedValue {
			t.Errorf("Expected arg '%s' to be '%s', got '%s'", key, expectedValue, actualValue)
		}
	}
}

func TestParseComposeEnvironmentListFormat(t *testing.T) {
	// Test environment variables as list format (KEY=value or KEY)
	composeWithListEnv := `
services:
  app:
    image: myapp:latest
    environment:
      - NEXT_PUBLIC_DISABLE_SIGN_UP=
      - NEXT_PUBLIC_SUPABASE_URL=https://dummy.supabase.co
      - DATABASE_URL=postgresql://user:pass@postgres:5432/db
      - DEBUG
`

	compose, err := ParseCompose([]byte(composeWithListEnv))
	if err != nil {
		t.Fatalf("Failed to parse compose with environment list: %v", err)
	}

	appService, exists := compose.Services["app"]
	if !exists {
		t.Fatal("Expected app service to exist")
	}

	// Check that environment variables were parsed correctly
	if len(appService.Environment) == 0 {
		t.Error("Expected environment variables to be parsed")
	}

	expectedEnv := map[string]string{
		"NEXT_PUBLIC_DISABLE_SIGN_UP": "",
		"NEXT_PUBLIC_SUPABASE_URL":    "https://dummy.supabase.co",
		"DATABASE_URL":                "postgresql://user:pass@postgres:5432/db",
		"DEBUG":                       "",
	}

	if len(appService.Environment) != len(expectedEnv) {
		t.Errorf("Expected %d env vars, got %d", len(expectedEnv), len(appService.Environment))
	}

	for key, expectedValue := range expectedEnv {
		actualValue, exists := appService.Environment[key]
		if !exists {
			t.Errorf("Expected env var '%s' not found", key)
			continue
		}
		if actualValue != expectedValue {
			t.Errorf("Expected env var '%s' to be '%s', got '%s'", key, expectedValue, actualValue)
		}
	}

	// Specifically test the empty value case (KEY=)
	if val, ok := appService.Environment["NEXT_PUBLIC_DISABLE_SIGN_UP"]; !ok {
		t.Error("NEXT_PUBLIC_DISABLE_SIGN_UP should be present")
	} else if val != "" {
		t.Errorf("NEXT_PUBLIC_DISABLE_SIGN_UP should be empty string, got '%s'", val)
	}
}

func TestParseComposeEnvironmentMapFormat(t *testing.T) {
	// Test environment variables as map format (KEY: value)
	composeWithMapEnv := `
services:
  app:
    image: myapp:latest
    environment:
      DEBUG: "true"
      NODE_ENV: production
      PORT: "3000"
`

	compose, err := ParseCompose([]byte(composeWithMapEnv))
	if err != nil {
		t.Fatalf("Failed to parse compose with environment map: %v", err)
	}

	appService, exists := compose.Services["app"]
	if !exists {
		t.Fatal("Expected app service to exist")
	}

	// Check that environment variables were parsed correctly
	if len(appService.Environment) == 0 {
		t.Error("Expected environment variables to be parsed")
	}

	expectedEnv := map[string]string{
		"DEBUG":   "true",
		"NODE_ENV": "production",
		"PORT":    "3000",
	}

	if len(appService.Environment) != len(expectedEnv) {
		t.Errorf("Expected %d env vars, got %d", len(expectedEnv), len(appService.Environment))
	}

	for key, expectedValue := range expectedEnv {
		actualValue, exists := appService.Environment[key]
		if !exists {
			t.Errorf("Expected env var '%s' not found", key)
			continue
		}
		if actualValue != expectedValue {
			t.Errorf("Expected env var '%s' to be '%s', got '%s'", key, expectedValue, actualValue)
		}
	}
}

// cloudflaredLikeContainerConfig returns a tunnel.ContainerConfig that mimics cloudflared for tests.
func cloudflaredLikeContainerConfig(tunnelToken string) *tunnel.ContainerConfig {
	return &tunnel.ContainerConfig{
		Image:   "cloudflare/cloudflared:latest",
		Command: []string{"tunnel", "run"},
		Environment: map[string]string{
			"TUNNEL_TOKEN": tunnelToken,
		},
	}
}

// quickTunnelContainerConfig returns a tunnel.ContainerConfig for Quick Tunnel (--url and metrics port).
// metricsHostPort is the host port for metrics (e.g. 2000); container port is always 2000.
func quickTunnelContainerConfig(metricsHostPort int) *tunnel.ContainerConfig {
	if metricsHostPort < 1 {
		metricsHostPort = 2000
	}
	return &tunnel.ContainerConfig{
		Image:   "cloudflare/cloudflared:latest",
		Command: []string{"tunnel", "--url", "http://web:80", "--metrics", "0.0.0.0:2000"},
		Ports:   []string{fmt.Sprintf("%d:2000", metricsHostPort)},
	}
}

func TestExtractQuickTunnelTargetFromCompose(t *testing.T) {
	composeWithQuickTunnel := `
version: "3.8"
services:
  web:
    image: nginx:latest
    networks: [default]
  tunnel:
    image: cloudflare/cloudflared:latest
    command: tunnel --url http://web:80 --metrics 0.0.0.0:2000
    ports: ["2000:2000"]
    networks: [default]
networks:
  default: { driver: bridge }
`
	service, port, ok := ExtractQuickTunnelTargetFromCompose(composeWithQuickTunnel)
	if !ok {
		t.Fatal("ExtractQuickTunnelTargetFromCompose() ok = false, want true")
	}
	if service != "web" || port != 80 {
		t.Errorf("ExtractQuickTunnelTargetFromCompose() = %q, %d; want web, 80", service, port)
	}

	// No cloudflared service
	noTunnel := `version: "3.8"
services:
  web:
    image: nginx:latest
`
	_, _, ok = ExtractQuickTunnelTargetFromCompose(noTunnel)
	if ok {
		t.Error("ExtractQuickTunnelTargetFromCompose(no tunnel) ok = true, want false")
	}
}

func TestExtractQuickTunnelMetricsHostPort(t *testing.T) {
	composeWithPort := `
services:
  tunnel:
    image: cloudflare/cloudflared:latest
    ports: ["2001:2000"]
  web:
    image: nginx:latest
`
	port, ok := ExtractQuickTunnelMetricsHostPort(composeWithPort)
	if !ok {
		t.Fatal("ExtractQuickTunnelMetricsHostPort() ok = false, want true")
	}
	if port != 2001 {
		t.Errorf("ExtractQuickTunnelMetricsHostPort() = %d, want 2001", port)
	}

	noTunnel := `services: { web: { image: nginx } }`
	_, ok = ExtractQuickTunnelMetricsHostPort(noTunnel)
	if ok {
		t.Error("ExtractQuickTunnelMetricsHostPort(no tunnel) ok = true, want false")
	}
}

// env_file contents are resolved by Docker at deploy time, from the app directory.
// Parsing must not read them from the server's working directory.
func TestParseComposeEnvFileMissingFromWorkingDir(t *testing.T) {
	content := `services:
  web:
    image: nginx
    env_file: .env
`
	compose, err := ParseCompose([]byte(content))
	if err != nil {
		t.Fatalf("ParseCompose() error = %v, want nil", err)
	}
	if _, ok := compose.Services["web"]; !ok {
		t.Fatalf("service web missing from parsed compose")
	}
}

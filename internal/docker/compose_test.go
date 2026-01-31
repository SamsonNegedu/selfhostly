package docker

import (
	"fmt"
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

	if compose.Version != "3.8" {
		t.Errorf("Expected version 3.8, got %s", compose.Version)
	}

	if len(compose.Services) != 1 {
		t.Errorf("Expected 1 service, got %d", len(compose.Services))
	}

	if len(compose.Networks) != 1 {
		t.Errorf("Expected 1 network, got %d", len(compose.Networks))
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

func TestInjectTunnelContainer(t *testing.T) {
	// Test with existing network
	compose := &ComposeFile{
		Services: map[string]Service{
			"web": {
				Image:    "nginx:latest",
				Networks: []string{"webnet"},
			},
		},
		Networks: map[string]Network{
			"webnet": {Driver: "bridge"},
		},
	}

	appName := "test-app"
	tunnelToken := "test-token"
	network := "webnet"

	injected, err := InjectTunnelContainer(compose, appName, cloudflaredLikeContainerConfig(tunnelToken), network)
	if err != nil {
		t.Fatalf("Failed to inject tunnel container: %v", err)
	}
	if !injected {
		t.Fatalf("Expected container to be injected")
	}

	// Check if tunnel service was added
	tunnelService, exists := compose.Services["tunnel"]
	if !exists {
		t.Fatalf("Expected tunnel service to be added")
	}

	if tunnelService.Image != "cloudflare/cloudflared:latest" {
		t.Errorf("Expected image cloudflare/cloudflared:latest, got %s", tunnelService.Image)
	}

	expectedContainerName := "test-app-tunnel"
	if tunnelService.ContainerName != expectedContainerName {
		t.Errorf("Expected container name %s, got %s", expectedContainerName, tunnelService.ContainerName)
	}

	if tunnelService.Restart != "always" {
		t.Errorf("Expected restart policy always, got %s", tunnelService.Restart)
	}

	if len(tunnelService.Networks) != 1 || tunnelService.Networks[0] != network {
		t.Errorf("Expected networks [%s], got %v", network, tunnelService.Networks)
	}

	if tunnelService.Environment["TUNNEL_TOKEN"] != tunnelToken {
		t.Errorf("Expected environment TUNNEL_TOKEN=%s, got %s", tunnelToken, tunnelService.Environment["TUNNEL_TOKEN"])
	}

	if tunnelService.Command != "tunnel run" {
		t.Errorf("Expected command tunnel run, got %s", tunnelService.Command)
	}
}

func TestInjectTunnelContainerNoNetwork(t *testing.T) {
	// Test with no existing network
	compose := &ComposeFile{
		Services: map[string]Service{
			"web": {
				Image: "nginx:latest",
			},
		},
	}

	appName := "test-app"
	tunnelToken := "test-token"

	injected, err := InjectTunnelContainer(compose, appName, cloudflaredLikeContainerConfig(tunnelToken), "")
	if err != nil {
		t.Fatalf("Failed to inject tunnel container: %v", err)
	}
	if !injected {
		t.Fatalf("Expected container to be injected")
	}

	// Check if default network was created
	network, exists := compose.Networks["selfhostly-network"]
	if !exists {
		t.Fatalf("Expected default network to be created")
	}

	if network.Driver != "bridge" {
		t.Errorf("Expected bridge driver, got %s", network.Driver)
	}

	// Check if tunnel service was added with the default network
	tunnelService, exists := compose.Services["tunnel"]
	if !exists {
		t.Fatalf("Expected tunnel service to be added")
	}

	if len(tunnelService.Networks) != 1 || tunnelService.Networks[0] != "selfhostly-network" {
		t.Errorf("Expected networks [selfhostly-network], got %v", tunnelService.Networks)
	}
}

func TestInjectTunnelContainerQuickTunnel(t *testing.T) {
	compose := &ComposeFile{
		Services: map[string]Service{
			"web": {
				Image:    "nginx:latest",
				Networks: []string{"webnet"},
			},
		},
		Networks: map[string]Network{
			"webnet": {Driver: "bridge"},
		},
	}
	injected, err := InjectTunnelContainer(compose, "myapp", quickTunnelContainerConfig(2000), "webnet")
	if err != nil {
		t.Fatalf("Failed to inject Quick Tunnel container: %v", err)
	}
	if !injected {
		t.Fatal("Expected container to be injected")
	}
	tunnelService, exists := compose.Services["tunnel"]
	if !exists {
		t.Fatal("Expected tunnel service to be added")
	}
	if tunnelService.Command != "tunnel --url http://web:80 --metrics 0.0.0.0:2000" {
		t.Errorf("Expected Quick Tunnel command, got %s", tunnelService.Command)
	}
	if len(tunnelService.Ports) != 1 || tunnelService.Ports[0] != "2000:2000" {
		t.Errorf("Expected ports [2000:2000], got %v", tunnelService.Ports)
	}
}

func TestExtractNetworks(t *testing.T) {
	// Test with services using networks
	compose := &ComposeFile{
		Services: map[string]Service{
			"web": {
				Image:    "nginx:latest",
				Networks: []string{"frontend"},
			},
			"db": {
				Image:    "postgres:latest",
				Networks: []string{"backend"},
			},
			"app": {
				Image:    "myapp:latest",
				Networks: []string{"frontend", "backend"},
			},
		},
		Networks: map[string]Network{
			"frontend": {Driver: "bridge"},
			"backend":  {Driver: "bridge"},
		},
	}

	networks := ExtractNetworks(compose)

	// ExtractNetworks already returns unique networks
	expectedNetworks := []string{"frontend", "backend"}
	if len(networks) != len(expectedNetworks) {
		t.Errorf("Expected %d unique networks, got %d", len(expectedNetworks), len(networks))
	}

	// Verify that we get the expected unique networks
	networkMap := make(map[string]bool)
	for _, network := range networks {
		networkMap[network] = true
	}

	for _, expectedNetwork := range expectedNetworks {
		if !networkMap[expectedNetwork] {
			t.Errorf("Expected network %s not found in extracted networks", expectedNetwork)
		}
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

func TestRemoveTunnelService(t *testing.T) {
	composeWithTunnel := `
version: "3.8"
services:
  web:
    image: nginx:latest
    networks: [default]
  tunnel:
    image: cloudflare/cloudflared:latest
    command: tunnel run
    networks: [default]
networks:
  default: { driver: bridge }
`
	compose, err := ParseCompose([]byte(composeWithTunnel))
	if err != nil {
		t.Fatalf("ParseCompose: %v", err)
	}
	if _, ok := compose.Services["tunnel"]; !ok {
		t.Fatal("compose should have tunnel service before RemoveTunnelService")
	}
	removed := RemoveTunnelService(compose)
	if !removed {
		t.Error("RemoveTunnelService() = false, want true")
	}
	if _, ok := compose.Services["tunnel"]; ok {
		t.Error("tunnel service should be removed from compose")
	}
	if _, ok := compose.Services["web"]; !ok {
		t.Error("web service should still be present")
	}

	// No tunnel service: RemoveTunnelService returns false
	composeNoTunnel := &ComposeFile{
		Services: map[string]Service{"web": {Image: "nginx"}},
	}
	removed = RemoveTunnelService(composeNoTunnel)
	if removed {
		t.Error("RemoveTunnelService(no tunnel) = true, want false")
	}
}

func TestMarshalComposeFile(t *testing.T) {
	// Create a compose file
	compose := &ComposeFile{
		Version: "3.8",
		Services: map[string]Service{
			"web": {
				Image: "nginx:latest",
				Ports: []string{"8080:80"},
			},
		},
	}

	// Marshal to YAML
	data, err := MarshalComposeFile(compose)
	if err != nil {
		t.Fatalf("Failed to marshal compose file: %v", err)
	}

	if len(data) == 0 {
		t.Error("Expected non-empty YAML output")
	}

	// Parse the marshaled data to verify it's valid
	parsed, err := ParseCompose(data)
	if err != nil {
		t.Fatalf("Failed to parse marshaled compose file: %v", err)
	}

	if parsed.Version != compose.Version {
		t.Errorf("Expected version %s, got %s", compose.Version, parsed.Version)
	}

	if len(parsed.Services) != len(compose.Services) {
		t.Errorf("Expected %d services, got %d", len(compose.Services), len(parsed.Services))
	}
}

func TestInjectTunnelContainerAndMarshal(t *testing.T) {
	// Create original compose, inject tunnel container, marshal and verify
	original := &ComposeFile{
		Version: "3.8",
		Services: map[string]Service{
			"web": {
				Image:    "nginx:latest",
				Networks: []string{"frontend"},
			},
		},
		Networks: map[string]Network{
			"frontend": {Driver: "bridge"},
		},
	}

	injected, err := InjectTunnelContainer(original, "web", cloudflaredLikeContainerConfig("test-token"), "frontend")
	if err != nil {
		t.Fatalf("Failed to inject tunnel container: %v", err)
	}
	if !injected {
		t.Fatalf("Expected container to be injected")
	}

	data, err := MarshalComposeFile(original)
	if err != nil {
		t.Fatalf("Failed to marshal compose file: %v", err)
	}
	if len(data) == 0 {
		t.Error("Expected non-empty compose data")
	}

	merged, err := ParseCompose(data)
	if err != nil {
		t.Fatalf("Failed to parse merged compose file: %v", err)
	}

	if len(merged.Services) != 2 {
		t.Errorf("Expected 2 services, got %d", len(merged.Services))
	}

	if _, exists := merged.Services["web"]; !exists {
		t.Error("Expected web service to exist in merged compose")
	}

	if _, exists := merged.Services["tunnel"]; !exists {
		t.Error("Expected tunnel service to exist in merged compose")
	}

	if len(merged.Networks) != 1 {
		t.Errorf("Expected 1 network, got %d", len(merged.Networks))
	}

	if _, exists := merged.Networks["frontend"]; !exists {
		t.Error("Expected frontend network to exist in merged compose")
	}
}

func TestUniqueStrings(t *testing.T) {
	// Test with duplicates
	input := []string{"a", "b", "c", "a", "b", "d"}
	result := uniqueStrings(input)

	expected := []string{"a", "b", "c", "d"}
	if len(result) != len(expected) {
		t.Errorf("Expected %d unique strings, got %d", len(expected), len(result))
	}

	// Check if all expected strings are present
	expectedMap := make(map[string]bool)
	for _, s := range expected {
		expectedMap[s] = true
	}

	for _, s := range result {
		if !expectedMap[s] {
			t.Errorf("Unexpected string in result: %s", s)
		}
	}

	// Test with empty slice
	emptyResult := uniqueStrings([]string{})
	if len(emptyResult) != 0 {
		t.Errorf("Expected empty result for empty input, got %d elements", len(emptyResult))
	}

	// Test with no duplicates
	noDuplicates := []string{"a", "b", "c"}
	noDuplicatesResult := uniqueStrings(noDuplicates)
	if len(noDuplicatesResult) != len(noDuplicates) {
		t.Errorf("Expected %d strings for input with no duplicates, got %d", len(noDuplicates), len(noDuplicatesResult))
	}
}

func TestInjectTunnelContainerWithServiceWithoutNetwork(t *testing.T) {
	// Test with a service that has no network defined (like uptime-kuma)
	// This is the real-world scenario where services use default bridge network
	compose := &ComposeFile{
		Services: map[string]Service{
			"uptime-kuma": {
				Image:         "louislam/uptime-kuma:2",
				ContainerName: "uptime-kuma",
				Ports:         []string{"3001"},
				Volumes:       []string{"/uptime-kuma/data:/app/data"},
				Restart:       "unless-stopped",
				// No Networks field defined!
			},
		},
		// No Networks section either
	}

	appName := "uptime-kuma"
	tunnelToken := "test-token"

	injected, err := InjectTunnelContainer(compose, appName, cloudflaredLikeContainerConfig(tunnelToken), "")
	if err != nil {
		t.Fatalf("Failed to inject tunnel container: %v", err)
	}
	if !injected {
		t.Fatalf("Expected container to be injected")
	}

	// Check if default network was created
	network, exists := compose.Networks["selfhostly-network"]
	if !exists {
		t.Fatalf("Expected default network to be created")
	}

	if network.Driver != "bridge" {
		t.Errorf("Expected bridge driver, got %s", network.Driver)
	}

	// Check if tunnel service was added with the default network
	tunnelService, exists := compose.Services["tunnel"]
	if !exists {
		t.Fatalf("Expected tunnel service to be added")
	}

	if len(tunnelService.Networks) != 1 || tunnelService.Networks[0] != "selfhostly-network" {
		t.Errorf("Expected networks [selfhostly-network], got %v", tunnelService.Networks)
	}

	// THE CRITICAL CHECK: The original service should also be updated to use the same network!
	uptimeKumaService := compose.Services["uptime-kuma"]
	if len(uptimeKumaService.Networks) == 0 {
		t.Error("Original service should be updated to use the same network as tunnel")
	}

	if len(uptimeKumaService.Networks) != 1 || uptimeKumaService.Networks[0] != "selfhostly-network" {
		t.Errorf("Expected original service to use [selfhostly-network], got %v", uptimeKumaService.Networks)
	}
}

func TestInjectTunnelContainerPreservesExistingNetwork(t *testing.T) {
	// Test that if a service already has a network, we use it and don't override
	compose := &ComposeFile{
		Services: map[string]Service{
			"web": {
				Image:    "nginx:latest",
				Networks: []string{"my-custom-network"},
			},
		},
		Networks: map[string]Network{
			"my-custom-network": {Driver: "bridge"},
		},
	}

	appName := "web"
	tunnelToken := "test-token"

	injected, err := InjectTunnelContainer(compose, appName, cloudflaredLikeContainerConfig(tunnelToken), "")
	if err != nil {
		t.Fatalf("Failed to inject tunnel container: %v", err)
	}
	if !injected {
		t.Fatalf("Expected container to be injected")
	}

	// tunnel should use the existing network
	tunnelService := compose.Services["tunnel"]
	if len(tunnelService.Networks) != 1 || tunnelService.Networks[0] != "my-custom-network" {
		t.Errorf("Expected tunnel to use [my-custom-network], got %v", tunnelService.Networks)
	}

	// Original service should keep its network unchanged
	webService := compose.Services["web"]
	if len(webService.Networks) != 1 || webService.Networks[0] != "my-custom-network" {
		t.Errorf("Expected web service to keep [my-custom-network], got %v", webService.Networks)
	}
}

package docker

import (
	"testing"
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

func TestInjectCloudflared(t *testing.T) {
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

	err := InjectCloudflared(compose, appName, tunnelToken, network)
	if err != nil {
		t.Fatalf("Failed to inject cloudflared: %v", err)
	}

	// Check if cloudflared service was added
	cloudflaredService, exists := compose.Services["cloudflared"]
	if !exists {
		t.Fatalf("Expected cloudflared service to be added")
	}

	if cloudflaredService.Image != "cloudflare/cloudflared:latest" {
		t.Errorf("Expected image cloudflare/cloudflared:latest, got %s", cloudflaredService.Image)
	}

	expectedContainerName := "test-app-cloudflared"
	if cloudflaredService.ContainerName != expectedContainerName {
		t.Errorf("Expected container name %s, got %s", expectedContainerName, cloudflaredService.ContainerName)
	}

	if cloudflaredService.Restart != "always" {
		t.Errorf("Expected restart policy always, got %s", cloudflaredService.Restart)
	}

	if len(cloudflaredService.Networks) != 1 || cloudflaredService.Networks[0] != network {
		t.Errorf("Expected networks [%s], got %v", network, cloudflaredService.Networks)
	}

	if cloudflaredService.Environment["TUNNEL_TOKEN"] != tunnelToken {
		t.Errorf("Expected environment TUNNEL_TOKEN=%s, got %s", tunnelToken, cloudflaredService.Environment["TUNNEL_TOKEN"])
	}

	if cloudflaredService.Command != "tunnel run" {
		t.Errorf("Expected command tunnel run, got %s", cloudflaredService.Command)
	}
}

func TestInjectCloudflaredNoNetwork(t *testing.T) {
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

	err := InjectCloudflared(compose, appName, tunnelToken, "")
	if err != nil {
		t.Fatalf("Failed to inject cloudflared: %v", err)
	}

	// Check if default network was created
	network, exists := compose.Networks["automaton-network"]
	if !exists {
		t.Fatalf("Expected default network to be created")
	}

	if network.Driver != "bridge" {
		t.Errorf("Expected bridge driver, got %s", network.Driver)
	}

	// Check if cloudflared service was added with the default network
	cloudflaredService, exists := compose.Services["cloudflared"]
	if !exists {
		t.Fatalf("Expected cloudflared service to be added")
	}

	if len(cloudflaredService.Networks) != 1 || cloudflaredService.Networks[0] != "automaton-network" {
		t.Errorf("Expected networks [automaton-network], got %v", cloudflaredService.Networks)
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

func TestMergeServices(t *testing.T) {
	// Create original compose
	original := &ComposeFile{
		Version: "3.8",
		Services: map[string]Service{
			"web": {
				Image: "nginx:latest",
			},
		},
		Networks: map[string]Network{
			"frontend": {Driver: "bridge"},
		},
	}

	// Create cloudflared compose
	cloudflared := &ComposeFile{
		Services: map[string]Service{
			"cloudflared": {
				Image:    "cloudflare/cloudflared:latest",
				Networks: []string{"tunnel"},
			},
		},
		Networks: map[string]Network{
			"tunnel": {Driver: "bridge"},
		},
	}

	// Merge services
	data := MergeServices(original, cloudflared)
	if len(data) == 0 {
		t.Error("Expected non-empty merged compose data")
	}

	// Parse the merged data to verify
	merged, err := ParseCompose(data)
	if err != nil {
		t.Fatalf("Failed to parse merged compose file: %v", err)
	}

	// Check if all services were merged
	if len(merged.Services) != 2 {
		t.Errorf("Expected 2 services, got %d", len(merged.Services))
	}

	if _, exists := merged.Services["web"]; !exists {
		t.Error("Expected web service to exist in merged compose")
	}

	if _, exists := merged.Services["cloudflared"]; !exists {
		t.Error("Expected cloudflared service to exist in merged compose")
	}

	// Check if all networks were merged
	if len(merged.Networks) != 2 {
		t.Errorf("Expected 2 networks, got %d", len(merged.Networks))
	}

	if _, exists := merged.Networks["frontend"]; !exists {
		t.Error("Expected frontend network to exist in merged compose")
	}

	if _, exists := merged.Networks["tunnel"]; !exists {
		t.Error("Expected tunnel network to exist in merged compose")
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

func TestInjectCloudflaredWithServiceWithoutNetwork(t *testing.T) {
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

	err := InjectCloudflared(compose, appName, tunnelToken, "")
	if err != nil {
		t.Fatalf("Failed to inject cloudflared: %v", err)
	}

	// Check if default network was created
	network, exists := compose.Networks["automaton-network"]
	if !exists {
		t.Fatalf("Expected default network to be created")
	}

	if network.Driver != "bridge" {
		t.Errorf("Expected bridge driver, got %s", network.Driver)
	}

	// Check if cloudflared service was added with the default network
	cloudflaredService, exists := compose.Services["cloudflared"]
	if !exists {
		t.Fatalf("Expected cloudflared service to be added")
	}

	if len(cloudflaredService.Networks) != 1 || cloudflaredService.Networks[0] != "automaton-network" {
		t.Errorf("Expected networks [automaton-network], got %v", cloudflaredService.Networks)
	}

	// THE CRITICAL CHECK: The original service should also be updated to use the same network!
	uptimeKumaService := compose.Services["uptime-kuma"]
	if len(uptimeKumaService.Networks) == 0 {
		t.Error("Original service should be updated to use the same network as cloudflared")
	}

	if len(uptimeKumaService.Networks) != 1 || uptimeKumaService.Networks[0] != "automaton-network" {
		t.Errorf("Expected original service to use [automaton-network], got %v", uptimeKumaService.Networks)
	}
}

func TestInjectCloudflaredPreservesExistingNetwork(t *testing.T) {
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

	err := InjectCloudflared(compose, appName, tunnelToken, "")
	if err != nil {
		t.Fatalf("Failed to inject cloudflared: %v", err)
	}

	// cloudflared should use the existing network
	cloudflaredService := compose.Services["cloudflared"]
	if len(cloudflaredService.Networks) != 1 || cloudflaredService.Networks[0] != "my-custom-network" {
		t.Errorf("Expected cloudflared to use [my-custom-network], got %v", cloudflaredService.Networks)
	}

	// Original service should keep its network unchanged
	webService := compose.Services["web"]
	if len(webService.Networks) != 1 || webService.Networks[0] != "my-custom-network" {
		t.Errorf("Expected web service to keep [my-custom-network], got %v", webService.Networks)
	}
}

package docker

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// ComposeFile represents a docker-compose.yml structure
type ComposeFile struct {
	Version  string             `yaml:"version"`
	Services map[string]Service `yaml:"services"`
	Networks map[string]Network `yaml:"networks"`
	Volumes  map[string]Volume  `yaml:"volumes"`
}

// Service represents a docker-compose service
type Service struct {
	Image         string   `yaml:"image"`
	ContainerName string   `yaml:"container_name,omitempty"`
	Command       string   `yaml:"command,omitempty"`
	Environment   []string `yaml:"environment,omitempty"`
	Ports         []string `yaml:"ports,omitempty"`
	Volumes       []string `yaml:"volumes,omitempty"`
	Networks      []string `yaml:"networks,omitempty"`
	DependsOn     []string `yaml:"depends_on,omitempty"`
	Restart       string   `yaml:"restart,omitempty"`
}

// Network represents a docker-compose network
type Network struct {
	Driver string `yaml:"driver,omitempty"`
}

// Volume represents a docker-compose volume
type Volume struct{}

// ParseCompose parses and validates docker-compose YAML content
func ParseCompose(content []byte) (*ComposeFile, error) {
	var compose ComposeFile
	if err := yaml.Unmarshal(content, &compose); err != nil {
		return nil, fmt.Errorf("failed to parse compose file: %w", err)
	}

	if len(compose.Services) == 0 {
		return nil, fmt.Errorf("no services defined in compose file")
	}

	return &compose, nil
}

// InjectCloudflared injects the cloudflared service into the compose file
func InjectCloudflared(compose *ComposeFile, appName, tunnelToken string, network string) error {
	if compose.Services == nil {
		compose.Services = make(map[string]Service)
	}

	cloudflaredService := Service{
		Image:   "cloudflare/cloudflared:latest",
		Command: fmt.Sprintf("tunnel --no-autoupdate run --token %s", tunnelToken),
		Restart: "unless-stopped",
	}

	if network != "" {
		cloudflaredService.Networks = []string{network}
	}

	compose.Services["cloudflared"] = cloudflaredService
	return nil
}

// ExtractNetworks extracts network names from services
func ExtractNetworks(compose *ComposeFile) []string {
	var networks []string

	for _, service := range compose.Services {
		for _, network := range service.Networks {
			networks = append(networks, network)
		}
	}

	// Also check if there are defined networks
	if len(compose.Networks) > 0 {
		for name := range compose.Networks {
			networks = append(networks, name)
		}
	}

	return uniqueStrings(networks)
}

// MarshalComposeFile marshals a ComposeFile to YAML bytes
func MarshalComposeFile(compose *ComposeFile) ([]byte, error) {
	data, err := yaml.Marshal(compose)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// MergeServices merges the cloudflared service with existing services
// Deprecated: Use InjectCloudflared followed by MarshalComposeFile instead
func MergeServices(original, cloudflared *ComposeFile) []byte {
	if original.Services == nil {
		original.Services = make(map[string]Service)
	}

	// Merge cloudflared services
	for name, service := range cloudflared.Services {
		original.Services[name] = service
	}

	// Merge networks if needed
	if cloudflared.Networks != nil {
		if original.Networks == nil {
			original.Networks = make(map[string]Network)
		}
		for name, network := range cloudflared.Networks {
			original.Networks[name] = network
		}
	}

	data, err := yaml.Marshal(original)
	if err != nil {
		return nil
	}
	return data
}

func uniqueStrings(slice []string) []string {
	keys := make(map[string]bool)
	var result []string
	for _, s := range slice {
		if !keys[s] {
			keys[s] = true
			result = append(result, s)
		}
	}
	return result
}

package docker

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// ComposeFile represents a docker-compose.yml structure
type ComposeFile struct {
	Version  string             `yaml:"version,omitempty"`
	Services map[string]Service `yaml:"services"`
	Networks map[string]Network `yaml:"networks,omitempty"`
	Volumes  map[string]Volume  `yaml:"volumes,omitempty"`
}

// Service represents a docker-compose service
type Service struct {
	Image            string                 `yaml:"image"`
	ContainerName    string                 `yaml:"container_name,omitempty"`
	Command          string                 `yaml:"command,omitempty"`
	Build            BuildConfig            `yaml:"build,omitempty"`
	Environment      map[string]string      `yaml:"environment,omitempty"`
	EnvironmentFiles []string               `yaml:"env_file,omitempty"`
	Ports            []string               `yaml:"ports,omitempty"`
	Volumes          []string               `yaml:"volumes,omitempty"`
	Networks         []string               `yaml:"networks,omitempty"`
	DependsOn        []string               `yaml:"depends_on,omitempty"`
	Restart          string                 `yaml:"restart,omitempty"`
	Healthcheck      HealthcheckConfig      `yaml:"healthcheck,omitempty"`
	Labels           map[string]string      `yaml:"labels,omitempty"`
	Logging          LoggingConfig          `yaml:"logging,omitempty"`
	Deploy           DeployConfig           `yaml:"deploy,omitempty"`
	ExtraHosts       []string               `yaml:"extra_hosts,omitempty"`
	StopSignal       string                 `yaml:"stop_signal,omitempty"`
	Timeout          string                 `yaml:"timeout,omitempty"`
	WorkingDir       string                 `yaml:"working_dir,omitempty"`
	User             string                 `yaml:"user,omitempty"`
	Hostname         string                 `yaml:"hostname,omitempty"`
	Domainname       string                 `yaml:"domainname,omitempty"`
	ExposedPorts     map[string]interface{} `yaml:"expose,omitempty"`
	Tmpfs            []string               `yaml:"tmpfs,omitempty"`
	Devices          []string               `yaml:"devices,omitempty"`
	Privileged       bool                   `yaml:"privileged,omitempty"`
	ReadonlyRootfs   bool                   `yaml:"read_only,omitempty"`
	Init             bool                   `yaml:"init,omitempty"`
}

// Network represents a docker-compose network
type Network struct {
	Driver string `yaml:"driver,omitempty"`
}

// Volume represents a docker-compose volume
type Volume struct{}

// BuildConfig represents a docker-compose build configuration
type BuildConfig struct {
	Context    string            `yaml:"context,omitempty"`
	Dockerfile string            `yaml:"dockerfile,omitempty"`
	Args       map[string]string `yaml:"args,omitempty"`
	Target     string            `yaml:"target,omitempty"`
	Network    string            `yaml:"network,omitempty"`
	Labels     map[string]string `yaml:"labels,omitempty"`
	CacheFrom  string            `yaml:"cache_from,omitempty"`
	NoCache    bool              `yaml:"no_cache,omitempty"`
	Pull       bool              `yaml:"pull,omitempty"`
	SSH        map[string]string `yaml:"ssh,omitempty"`
}

// HealthcheckConfig represents a docker-compose healthcheck configuration
type HealthcheckConfig struct {
	Test        []string `yaml:"test,omitempty"`
	Interval    string   `yaml:"interval,omitempty"`
	Timeout     string   `yaml:"timeout,omitempty"`
	Retries     int      `yaml:"retries,omitempty"`
	StartPeriod string   `yaml:"start_period,omitempty"`
	Disable     bool     `yaml:"disable,omitempty"`
}

// LoggingConfig represents a docker-compose logging configuration
type LoggingConfig struct {
	Driver  string            `yaml:"driver,omitempty"`
	Options map[string]string `yaml:"options,omitempty"`
}

// DeployConfig represents a docker-compose deploy configuration
type DeployConfig struct {
	Replicas      *int              `yaml:"replicas,omitempty"`
	Mode          string            `yaml:"mode,omitempty"`
	Labels        map[string]string `yaml:"labels,omitempty"`
	UpdateConfig  UpdateConfig      `yaml:"update_config,omitempty"`
	Resources     Resources         `yaml:"resources,omitempty"`
	RestartPolicy RestartPolicy     `yaml:"restart_policy,omitempty"`
	Placement     Placement         `yaml:"placement,omitempty"`
}

// UpdateConfig represents docker-compose update configuration
type UpdateConfig struct {
	Parallelism     *int   `yaml:"parallelism,omitempty"`
	Delay           string `yaml:"delay,omitempty"`
	FailureAction   string `yaml:"failure_action,omitempty"`
	Monitor         string `yaml:"monitor,omitempty"`
	MaxFailureRatio string `yaml:"max_failure_ratio,omitempty"`
	Order           string `yaml:"order,omitempty"`
}

// Resources represents docker-compose resource limits
type Resources struct {
	Limits       ResourceLimits       `yaml:"limits,omitempty"`
	Reservations ResourceReservations `yaml:"reservations,omitempty"`
}

// ResourceLimits represents resource limits
type ResourceLimits struct {
	Cpus    string   `yaml:"cpus,omitempty"`
	Memory  string   `yaml:"memory,omitempty"`
	Pids    *int     `yaml:"pids,omitempty"`
	Devices []string `yaml:"devices,omitempty"`
}

// ResourceReservations represents resource reservations
type ResourceReservations struct {
	Cpus   string `yaml:"cpus,omitempty"`
	Memory string `yaml:"memory,omitempty"`
	Gpus   string `yaml:"gpus,omitempty"`
}

// RestartPolicy represents restart policy
type RestartPolicy struct {
	Condition   string `yaml:"condition,omitempty"`
	Delay       string `yaml:"delay,omitempty"`
	MaxAttempts *int   `yaml:"max_attempts,omitempty"`
	Window      string `yaml:"window,omitempty"`
}

// Placement represents placement constraints
type Placement struct {
	Constraints        []string `yaml:"constraints,omitempty"`
	Preferences        []string `yaml:"preferences,omitempty"`
	MaxReplicasPerNode *int     `yaml:"maxreplicaspernode,omitempty"`
}

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
	if compose.Networks == nil {
		compose.Networks = make(map[string]Network)
	}

	// If no specific network is provided, extract networks from existing services
	if network == "" {
		networks := ExtractNetworks(compose)
		if len(networks) > 0 {
			// Use the first network found or create a default one
			network = networks[0]
			if len(networks) > 1 {
				// If multiple networks exist, check if any service matches the appName
				for serviceName, service := range compose.Services {
					if serviceName == appName && len(service.Networks) > 0 {
						network = service.Networks[0]
						break
					}
				}
			}
		} else {
			// Default network name
			network = "automaton-network"
		}
	}

	// Check if the network exists in the compose file, if not create it
	if _, networkExists := compose.Networks[network]; !networkExists {
		compose.Networks[network] = Network{Driver: "bridge"}
	}

	// Add all services without an explicit network to the same network
	// This ensures cloudflared can communicate with all services
	for serviceName, service := range compose.Services {
		if len(service.Networks) == 0 {
			// Service has no network defined, add it to the same network as cloudflared
			service.Networks = []string{network}
			compose.Services[serviceName] = service
		}
	}

	cloudflaredService := Service{
		Image:         "cloudflare/cloudflared:latest",
		ContainerName: fmt.Sprintf("%s-cloudflared", appName),
		Restart:       "always",
		Networks:      []string{network},
		Environment: map[string]string{
			"TUNNEL_TOKEN": tunnelToken,
		},
		Command: "tunnel run",
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

package docker

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/compose-spec/compose-go/v2/loader"
	composetypes "github.com/compose-spec/compose-go/v2/types"
	"github.com/selfhostly/internal/constants"
	"github.com/selfhostly/internal/tunnel"
	"gopkg.in/yaml.v3"
)

// ComposeParseError represents a detailed compose file parsing error
type ComposeParseError struct {
	Line        int    // Line number where error occurred (0 if unknown)
	Column      int    // Column number where error occurred (0 if unknown)
	Message     string // Human-readable error message
	Suggestion  string // Optional suggestion for fixing the error
	OriginalErr error  // Original error from YAML parser
}

func (e *ComposeParseError) Error() string {
	if e.Line > 0 {
		if e.Suggestion != "" {
			return fmt.Sprintf("line %d: %s\nSuggestion: %s", e.Line, e.Message, e.Suggestion)
		}
		return fmt.Sprintf("line %d: %s", e.Line, e.Message)
	}
	if e.Suggestion != "" {
		return fmt.Sprintf("%s\nSuggestion: %s", e.Message, e.Suggestion)
	}
	return e.Message
}

func (e *ComposeParseError) Unwrap() error {
	return e.OriginalErr
}

// ComposeFile represents a docker-compose.yml structure
type ComposeFile struct {
	Version  string             `yaml:"version,omitempty"`
	Services map[string]Service `yaml:"services"`
	Networks map[string]Network `yaml:"networks,omitempty"`
	Volumes  map[string]Volume  `yaml:"volumes,omitempty"`
}

// DependsOnConfig represents a dependency configuration for a service
// Supports both formats:
//   - Simple list: ["service1", "service2"]
//   - With conditions: {service1: {condition: "service_healthy"}}
type DependsOnConfig struct {
	// For simple list format
	Services []string
	// For map format with conditions
	WithConditions map[string]DependsOnCondition
}

// DependsOnCondition represents a dependency condition
type DependsOnCondition struct {
	Condition string `yaml:"condition,omitempty"`
}

// MarshalYAML implements custom marshaling for DependsOnConfig
func (d DependsOnConfig) MarshalYAML() (interface{}, error) {
	if len(d.WithConditions) > 0 {
		return d.WithConditions, nil
	}
	return d.Services, nil
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
	DependsOn        DependsOnConfig        `yaml:"depends_on,omitempty"`
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
	Name     string `yaml:"name,omitempty"`
	Driver   string `yaml:"driver,omitempty"`
	External bool   `yaml:"external,omitempty"`
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

// enhanceComposeGoError wraps a compose-go error with our custom ComposeParseError
func enhanceComposeGoError(err error, content []byte) *ComposeParseError {
	line, column := extractLineNumberFromError(err)
	errStr := err.Error()

	parseErr := &ComposeParseError{
		Line:        line,
		Column:      column,
		OriginalErr: err,
	}

	// Provide specific error messages based on common errors
	switch {
	case strings.Contains(errStr, "yaml:") || strings.Contains(errStr, "cannot unmarshal"):
		parseErr.Message = "YAML syntax error"
		parseErr.Suggestion = "Check for missing colons, incorrect indentation, or unclosed quotes/brackets"
	case strings.Contains(errStr, "services"):
		parseErr.Message = "invalid services configuration"
		parseErr.Suggestion = "Check that all services are properly defined under the 'services:' section"
	case strings.Contains(errStr, "Additional property") || strings.Contains(errStr, "additional properties"):
		parseErr.Message = "unknown property in compose file"
		parseErr.Suggestion = "Check for typos in property names. Refer to the Docker Compose specification for valid properties."
	default:
		parseErr.Message = errStr
		parseErr.Suggestion = "Verify your Docker Compose file syntax. Check indentation, quotes, and structure."
	}

	// Add line context if we have a line number
	if line > 0 && len(content) > 0 {
		lines := strings.Split(string(content), "\n")
		if line <= len(lines) {
			contextLine := strings.TrimSpace(lines[line-1])
			if contextLine != "" {
				parseErr.Message += fmt.Sprintf(" (near: %s)", contextLine)
			}
		}
	}

	return parseErr
}

// extractLineNumberFromError attempts to extract line number from YAML error message
func extractLineNumberFromError(err error) (line, column int) {
	errStr := err.Error()

	// Try to match "line X:" pattern
	lineMatch := regexp.MustCompile(`line (\d+):`)
	matches := lineMatch.FindStringSubmatch(errStr)
	if len(matches) >= 2 {
		if lineNum, parseErr := strconv.Atoi(matches[1]); parseErr == nil {
			line = lineNum
		}
	}

	// Try to match "column X:" pattern
	colMatch := regexp.MustCompile(`column (\d+):`)
	colMatches := colMatch.FindStringSubmatch(errStr)
	if len(colMatches) >= 2 {
		if colNum, parseErr := strconv.Atoi(colMatches[1]); parseErr == nil {
			column = colNum
		}
	}

	return line, column
}

// ParseCompose parses and validates docker-compose YAML content using the official compose-go library.
// This handles all Docker Compose formats (list vs map for environment, depends_on, build.args, etc.)
func ParseCompose(content []byte) (*ComposeFile, error) {
	// First, quick YAML syntax check to give better errors
	var raw map[string]interface{}
	if err := yaml.Unmarshal(content, &raw); err != nil {
		return nil, enhanceComposeGoError(err, content)
	}

	// Use compose-go to parse the content
	config := composetypes.ConfigDetails{
		ConfigFiles: []composetypes.ConfigFile{
			{Content: content},
		},
		// Empty environment map - we don't interpolate variables at parse time
		// Docker resolves ${VAR} at container runtime
		Environment: composetypes.Mapping{},
	}

	// Load with skip interpolation (keep ${VAR} as-is) and skip validation
	// (we do our own validation downstream). We keep normalization enabled
	// so compose-go creates implicit default networks, resolves short syntax, etc.
	opts := func(o *loader.Options) {
		o.SkipInterpolation = true
		o.SkipValidation = true
	}

	project, err := loader.LoadWithContext(context.Background(), config, opts)
	if err != nil {
		return nil, enhanceComposeGoError(err, content)
	}

	if project == nil || len(project.Services) == 0 {
		return nil, &ComposeParseError{
			Message:    "no services defined in compose file",
			Suggestion: "Add at least one service under the 'services:' section",
		}
	}

	// Convert compose-go types to our internal types
	return convertProject(project), nil
}

// convertProject converts a compose-go Project to our ComposeFile type
func convertProject(project *composetypes.Project) *ComposeFile {
	compose := &ComposeFile{
		Services: make(map[string]Service),
		Networks: make(map[string]Network),
		Volumes:  make(map[string]Volume),
	}

	// Convert services
	for name, svc := range project.Services {
		compose.Services[name] = convertService(svc)
	}

	// Convert networks
	for name, net := range project.Networks {
		compose.Networks[name] = convertNetwork(net)
	}

	// Convert volumes
	for name := range project.Volumes {
		compose.Volumes[name] = Volume{}
	}

	return compose
}

// convertService converts a compose-go ServiceConfig to our Service type
func convertService(svc composetypes.ServiceConfig) Service {
	service := Service{
		Image:         svc.Image,
		ContainerName: svc.ContainerName,
		Restart:       svc.Restart,
		User:          svc.User,
		Hostname:      svc.Hostname,
		Domainname:    svc.DomainName,
		WorkingDir:    svc.WorkingDir,
		StopSignal:    svc.StopSignal,
		Privileged:    svc.Privileged,
		ReadonlyRootfs: svc.ReadOnly,
		Init:          derefBool(svc.Init),
	}

	// Command ([]string → joined string)
	if len(svc.Command) > 0 {
		service.Command = strings.Join(svc.Command, " ")
	}

	// Environment (MappingWithEquals → map[string]string)
	if len(svc.Environment) > 0 {
		service.Environment = make(map[string]string)
		for key, val := range svc.Environment {
			if val != nil {
				service.Environment[key] = *val
			} else {
				service.Environment[key] = ""
			}
		}
	}

	// Env files
	for _, ef := range svc.EnvFiles {
		service.EnvironmentFiles = append(service.EnvironmentFiles, ef.Path)
	}

	// Ports ([]ServicePortConfig → []string "host:container")
	for _, p := range svc.Ports {
		service.Ports = append(service.Ports, convertPort(p))
	}

	// Volumes ([]ServiceVolumeConfig → []string "source:target")
	for _, v := range svc.Volumes {
		service.Volumes = append(service.Volumes, convertVolume(v))
	}

	// Networks (map[string]*ServiceNetworkConfig → []string)
	for networkName := range svc.Networks {
		service.Networks = append(service.Networks, networkName)
	}

	// DependsOn (compose-go DependsOnConfig → our DependsOnConfig)
	if len(svc.DependsOn) > 0 {
		service.DependsOn = convertDependsOn(svc.DependsOn)
	}

	// Build
	if svc.Build != nil {
		service.Build = convertBuild(*svc.Build)
	}

	// Healthcheck
	if svc.HealthCheck != nil {
		service.Healthcheck = convertHealthcheck(*svc.HealthCheck)
	}

	// Labels
	if len(svc.Labels) > 0 {
		service.Labels = make(map[string]string)
		for k, v := range svc.Labels {
			service.Labels[k] = v
		}
	}

	// Logging
	if svc.Logging != nil {
		service.Logging = LoggingConfig{
			Driver:  svc.Logging.Driver,
			Options: svc.Logging.Options,
		}
	}

	// ExtraHosts
	if len(svc.ExtraHosts) > 0 {
		for host, ips := range svc.ExtraHosts {
			for _, ip := range ips {
				service.ExtraHosts = append(service.ExtraHosts, host+":"+ip)
			}
		}
	}

	// Tmpfs
	service.Tmpfs = svc.Tmpfs

	// Devices
	for _, d := range svc.Devices {
		devStr := d.Source + ":" + d.Target
		if d.Permissions != "" {
			devStr += ":" + d.Permissions
		}
		service.Devices = append(service.Devices, devStr)
	}

	return service
}

// convertPort converts a compose-go ServicePortConfig to "host:container" string
func convertPort(p composetypes.ServicePortConfig) string {
	hostPort := p.Published
	containerPort := fmt.Sprintf("%d", p.Target)
	protocol := p.Protocol

	if hostPort == "" {
		hostPort = containerPort
	}

	result := fmt.Sprintf("%s:%s", hostPort, containerPort)
	if protocol != "" && protocol != "tcp" {
		result += "/" + protocol
	}
	return result
}

// convertVolume converts a compose-go ServiceVolumeConfig to "source:target" string
func convertVolume(v composetypes.ServiceVolumeConfig) string {
	if v.Source == "" {
		return v.Target
	}
	result := v.Source + ":" + v.Target
	if v.ReadOnly {
		result += ":ro"
	}
	return result
}

// convertDependsOn converts compose-go DependsOnConfig to our DependsOnConfig
func convertDependsOn(deps composetypes.DependsOnConfig) DependsOnConfig {
	result := DependsOnConfig{}

	hasConditions := false
	for _, dep := range deps {
		if dep.Condition != "" && dep.Condition != "service_started" {
			hasConditions = true
			break
		}
	}

	if hasConditions {
		result.WithConditions = make(map[string]DependsOnCondition)
		for name, dep := range deps {
			result.WithConditions[name] = DependsOnCondition{
				Condition: dep.Condition,
			}
		}
	} else {
		for name := range deps {
			result.Services = append(result.Services, name)
		}
	}

	return result
}

// convertBuild converts a compose-go BuildConfig to our BuildConfig
func convertBuild(b composetypes.BuildConfig) BuildConfig {
	build := BuildConfig{
		Context:    b.Context,
		Dockerfile: b.Dockerfile,
		Target:     b.Target,
		Network:    b.Network,
	}

	// Build args (MappingWithEquals → map[string]string)
	if len(b.Args) > 0 {
		build.Args = make(map[string]string)
		for key, val := range b.Args {
			if val != nil {
				build.Args[key] = *val
			} else {
				build.Args[key] = ""
			}
		}
	}

	// Labels
	if len(b.Labels) > 0 {
		build.Labels = make(map[string]string)
		for k, v := range b.Labels {
			build.Labels[k] = v
		}
	}

	return build
}

// convertHealthcheck converts a compose-go HealthCheckConfig to our HealthcheckConfig
func convertHealthcheck(hc composetypes.HealthCheckConfig) HealthcheckConfig {
	result := HealthcheckConfig{
		Disable: hc.Disable,
	}

	if len(hc.Test) > 0 {
		result.Test = hc.Test
	}

	if hc.Interval != nil {
		result.Interval = hc.Interval.String()
	}
	if hc.Timeout != nil {
		result.Timeout = hc.Timeout.String()
	}
	if hc.StartPeriod != nil {
		result.StartPeriod = hc.StartPeriod.String()
	}
	if hc.Retries != nil {
		result.Retries = int(*hc.Retries)
	}

	return result
}

// convertNetwork converts a compose-go NetworkConfig to our Network type
func convertNetwork(net composetypes.NetworkConfig) Network {
	return Network{
		Name:     net.Name,
		Driver:   net.Driver,
		External: bool(net.External),
	}
}

// derefBool safely dereferences a *bool, returning false if nil
func derefBool(b *bool) bool {
	if b != nil {
		return *b
	}
	return false
}

// checkDockerNetworkExists checks if a Docker network exists
func checkDockerNetworkExists(networkName string) bool {
	cmd := exec.Command("docker", "network", "inspect", networkName)
	err := cmd.Run()
	return err == nil
}

// InjectTunnelContainer injects a tunnel provider's container into the compose file.
// This is a generic function that works with any tunnel provider that needs a sidecar container.
// Returns true if a container was injected, false if containerConfig was nil.
func InjectTunnelContainer(compose *ComposeFile, appName string, containerConfig *tunnel.ContainerConfig, network string) (bool, error) {
	// Some providers don't need a container (e.g., certain hosted tunnel services)
	if containerConfig == nil {
		return false, nil
	}

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
			network = constants.CoreAPINetwork
		}
	}

	// Check if the network exists in the compose file, if not create it
	if _, networkExists := compose.Networks[network]; !networkExists {
		compose.Networks[network] = Network{Driver: "bridge"}
	}

	// Add all services without an explicit network to the same network
	// This ensures the tunnel container can communicate with all services
	for serviceName, service := range compose.Services {
		if len(service.Networks) == 0 {
			// Service has no network defined, add it to the same network as tunnel
			service.Networks = []string{network}
			compose.Services[serviceName] = service
		}
	}

	// Build network list for tunnel container
	// Always include the app's network (for reaching the app) and core API network (for being reached by primary)
	networks := containerConfig.Networks
	if len(networks) == 0 {
		networks = []string{network}
	}

	// Ensure core API network is added (for cross-network access from primary backend)
	hasCoreAPINetwork := false
	for _, n := range networks {
		if n == constants.CoreAPINetwork {
			hasCoreAPINetwork = true
			break
		}
	}
	if !hasCoreAPINetwork && network != constants.CoreAPINetwork {
		// Add core API network as an external network for cross-app communication
		networks = append(networks, constants.CoreAPINetwork)
		// Add it to compose.Networks
		if _, exists := compose.Networks[constants.CoreAPINetwork]; !exists {
			// Check if the network exists in Docker
			// If it exists, mark as external (use existing network)
			// If it doesn't exist, create it normally
			networkExists := checkDockerNetworkExists(constants.CoreAPINetwork)
			compose.Networks[constants.CoreAPINetwork] = Network{
				Driver:   "bridge",
				External: networkExists, // Only external if network already exists
			}
		}
	}

	// Build command string from array
	commandStr := ""
	if len(containerConfig.Command) > 0 {
		commandStr = strings.Join(containerConfig.Command, " ")
	}

	// Convert environment map
	env := containerConfig.Environment
	if env == nil {
		env = make(map[string]string)
	}

	tunnelService := Service{
		Image:         containerConfig.Image,
		ContainerName: fmt.Sprintf("%s-tunnel", appName),
		Restart:       "unless-stopped",
		Networks:      networks,
		Environment:   env,
		Command:       commandStr,
	}

	// Add volumes if specified
	if len(containerConfig.Volumes) > 0 {
		tunnelService.Volumes = containerConfig.Volumes
	}

	// Add ports if specified (e.g., metrics port for Quick Tunnel)
	if len(containerConfig.Ports) > 0 {
		tunnelService.Ports = containerConfig.Ports
	}

	compose.Services["tunnel"] = tunnelService
	return true, nil
}

// RemoveTunnelService removes the tunnel service from the compose file (e.g. after tunnel deletion).
// The injected tunnel service is always named "tunnel". Returns true if the service was present and removed.
func RemoveTunnelService(compose *ComposeFile) bool {
	if compose.Services == nil {
		return false
	}
	if _, ok := compose.Services["tunnel"]; !ok {
		return false
	}
	delete(compose.Services, "tunnel")
	return true
}

// ExtractQuickTunnelTargetFromCompose parses compose content and extracts the Quick Tunnel target
// (service name and port) from the tunnel service's command (e.g. --url http://web:80).
// Returns ("", 0, false) if not found. Used when updating an app to re-inject the Quick Tunnel container.
func ExtractQuickTunnelTargetFromCompose(composeContent string) (service string, port int, ok bool) {
	compose, err := ParseCompose([]byte(composeContent))
	if err != nil || compose.Services == nil {
		return "", 0, false
	}
	// Look for the tunnel service (injected by us) or any cloudflared service with --url
	for _, svc := range compose.Services {
		if !strings.Contains(svc.Image, "cloudflared") {
			continue
		}
		cmd := svc.Command
		// Command may be "tunnel --url http://serviceName:port --metrics ..."
		re := regexp.MustCompile(`--url\s+http://([^:\s]+):(\d+)`)
		matches := re.FindStringSubmatch(cmd)
		if len(matches) >= 3 {
			p, err := strconv.Atoi(matches[2])
			if err != nil || p < 1 || p > 65535 {
				continue
			}
			return strings.TrimSpace(matches[1]), p, true
		}
	}
	return "", 0, false
}

// ExtractQuickTunnelMetricsHostPort parses compose content and returns the host port used for Quick Tunnel metrics
// (the host side of the "HOST:CONTAINER" mapping on the tunnel service). Returns (0, false) if not found.
func ExtractQuickTunnelMetricsHostPort(composeContent string) (hostPort int, ok bool) {
	compose, err := ParseCompose([]byte(composeContent))
	if err != nil || compose.Services == nil {
		return 0, false
	}
	for _, svc := range compose.Services {
		if !strings.Contains(svc.Image, "cloudflared") || len(svc.Ports) == 0 {
			continue
		}
		// Extract container port from command first to be more robust
		containerPort := ExtractQuickTunnelMetricsContainerPort(svc.Command)
		if containerPort == 0 {
			containerPort = constants.QuickTunnelMetricsPort // Default fallback
		}

		// Port format is "hostPort:containerPort"
		for _, p := range svc.Ports {
			parts := strings.Split(p, ":")
			if len(parts) != 2 {
				continue
			}
			cp, err := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err != nil || cp != containerPort {
				continue
			}
			hp, err := strconv.Atoi(strings.TrimSpace(parts[0]))
			if err != nil || hp < constants.MinPort || hp > constants.MaxPort {
				continue
			}
			return hp, true
		}
	}
	return 0, false
}

// ExtractQuickTunnelMetricsContainerPort extracts the container port from the cloudflared command.
// Looks for --metrics 0.0.0.0:PORT or --metrics localhost:PORT in the command string.
// Returns 0 if not found (caller should use default 2000).
func ExtractQuickTunnelMetricsContainerPort(command string) int {
	if command == "" {
		return 0
	}
	// Match --metrics 0.0.0.0:PORT or --metrics localhost:PORT or --metrics :PORT
	re := regexp.MustCompile(`--metrics\s+(?:0\.0\.0\.0|localhost|127\.0\.0\.1)?:?(\d+)`)
	matches := re.FindStringSubmatch(command)
	if len(matches) >= 2 {
		port, err := strconv.Atoi(matches[1])
		if err == nil && port >= constants.MinPort && port <= constants.MaxPort {
			return port
		}
	}
	return 0
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

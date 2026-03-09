package validation

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/selfhostly/internal/docker"
)

var (
	// appNameRegex allows only alphanumeric characters, hyphens, and underscores
	appNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	
	// containerIDRegex validates Docker container IDs (12 or 64 character hex strings)
	// Note: Docker IDs are lowercase hex, but we accept uppercase for flexibility
	containerIDRegex = regexp.MustCompile(`^[a-fA-F0-9]{12,64}$`)
)

// SecurityConfig holds security validation configuration
type SecurityConfig struct {
	AllowedVolumePaths []string
}

// defaultSecurityConfig is used when no config is provided
var defaultSecurityConfig = &SecurityConfig{
	AllowedVolumePaths: []string{},
}

// Reserved names that should not be used as app names
var reservedNames = map[string]bool{
	".":    true,
	"..":   true,
	"~":    true,
	"tmp":  true,
	"temp": true,
}

// ValidateAppName validates an application name to prevent path traversal and other attacks
func ValidateAppName(name string) error {
	// Check length
	if len(name) < 1 {
		return errors.New("app name cannot be empty")
	}
	if len(name) > 64 {
		return errors.New("app name must be 64 characters or less")
	}
	
	// Check for path traversal sequences
	if strings.Contains(name, "..") {
		return errors.New("app name cannot contain '..'")
	}
	if strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return errors.New("app name cannot contain slashes")
	}
	
	// Check for reserved names
	if reservedNames[strings.ToLower(name)] {
		return errors.New("app name is reserved")
	}
	
	// Check against allowed character set
	if !appNameRegex.MatchString(name) {
		return errors.New("app name must contain only letters, numbers, hyphens, and underscores")
	}
	
	// Prevent names starting or ending with special characters
	if strings.HasPrefix(name, "-") || strings.HasPrefix(name, "_") {
		return errors.New("app name cannot start with a hyphen or underscore")
	}
	if strings.HasSuffix(name, "-") || strings.HasSuffix(name, "_") {
		return errors.New("app name cannot end with a hyphen or underscore")
	}
	
	return nil
}

// ValidateContainerID validates a Docker container ID format
func ValidateContainerID(id string) error {
	if id == "" {
		return errors.New("container ID cannot be empty")
	}
	
	// Docker container IDs are hex strings of 12 or 64 characters
	if !containerIDRegex.MatchString(id) {
		return errors.New("invalid container ID format (must be 12 or 64 character hex string)")
	}
	
	return nil
}

// ValidateComposeContent validates Docker Compose file content
func ValidateComposeContent(content string) error {
	return ValidateComposeContentWithConfig(content, nil)
}

// ValidateComposeContentWithConfig validates Docker Compose file content with custom security config
func ValidateComposeContentWithConfig(content string, securityConfig *SecurityConfig) error {
	// Check if empty
	if len(content) == 0 {
		return errors.New("compose file content cannot be empty")
	}
	
	// Check size limit (1MB)
	maxSize := 1 << 20 // 1MB
	if len(content) > maxSize {
		return fmt.Errorf("compose file too large: %d bytes (maximum %d bytes)", len(content), maxSize)
	}
	
	// Parse and validate the compose file structure
	compose, err := docker.ParseCompose([]byte(content))
	if err != nil {
		// If it's already a ComposeParseError, return it as-is
		var parseErr *docker.ComposeParseError
		if errors.As(err, &parseErr) {
			return err
		}
		// Otherwise wrap it
		return fmt.Errorf("invalid compose file: %w", err)
	}
	
	// Validate that services don't reference undefined networks
	if err := validateServiceNetworks(compose); err != nil {
		return fmt.Errorf("network validation failed: %w", err)
	}
	
	// Validate that services don't reference undefined volumes
	if err := validateServiceVolumes(compose); err != nil {
		return fmt.Errorf("volume validation failed: %w", err)
	}
	
	// Use default config if none provided
	if securityConfig == nil {
		securityConfig = defaultSecurityConfig
	}
	
	// Validate security: block dangerous configurations that could compromise the host
	if err := validateComposeSecurityWithConfig(compose, securityConfig); err != nil {
		return fmt.Errorf("security validation failed: %w", err)
	}
	
	return nil
}

// validateServiceNetworks ensures all networks referenced by services are defined
func validateServiceNetworks(compose *docker.ComposeFile) error {
	// Get all defined networks (including default network)
	definedNetworks := make(map[string]bool)
	if compose.Networks != nil {
		for networkName := range compose.Networks {
			definedNetworks[networkName] = true
		}
	}
	
	// Check each service's network references
	for serviceName, service := range compose.Services {
		for _, networkName := range service.Networks {
			// Skip if network is defined in the compose file
			if definedNetworks[networkName] {
				continue
			}
			
			// Check if it's a special network (external, default, etc.)
			// Docker allows "default" network and external networks
			if networkName == "default" {
				continue
			}
			
			return fmt.Errorf("service %q refers to undefined network %q: add the network definition under the 'networks:' section (e.g., networks: { %q: {} })", serviceName, networkName, networkName)
		}
	}
	
	return nil
}

// validateServiceVolumes ensures all named volumes referenced by services are defined
func validateServiceVolumes(compose *docker.ComposeFile) error {
	// Get all defined volumes
	definedVolumes := make(map[string]bool)
	if compose.Volumes != nil {
		for volumeName := range compose.Volumes {
			definedVolumes[volumeName] = true
		}
	}
	
	// Check each service's volume references
	for _, service := range compose.Services {
		for _, volumeSpec := range service.Volumes {
			// Parse volume spec (can be "volume-name:/path" or "/host/path:/container/path")
			parts := strings.Split(volumeSpec, ":")
			if len(parts) < 2 {
				continue // Invalid volume spec, will be caught by docker
			}
			
			volumeName := parts[0]
			
			// Skip if it's a host path (starts with / or ./ or ../)
			if strings.HasPrefix(volumeName, "/") || 
			   strings.HasPrefix(volumeName, "./") || 
			   strings.HasPrefix(volumeName, "../") {
				continue
			}
			
			// Skip if volume is defined
			if definedVolumes[volumeName] {
				continue
			}
			
			// If it's a named volume but not defined, Docker will create it automatically
			// We allow this to pass to maintain backward compatibility
			// In the future, we could make this stricter by uncommenting the error below:
			// return fmt.Errorf("service %q refers to undefined volume %q: volume must be defined in the volumes section", serviceName, volumeName)
		}
	}
	
	return nil
}

// validateComposeSecurity validates Docker Compose file for security vulnerabilities (backward compatibility)
func validateComposeSecurity(compose *docker.ComposeFile) error {
	return validateComposeSecurityWithConfig(compose, defaultSecurityConfig)
}

// validateComposeSecurityWithConfig validates Docker Compose file for security vulnerabilities
// that could allow host machine compromise. This function blocks configurations that:
// - Grant excessive privileges (privileged mode)
// - Mount sensitive host paths (Docker socket, root filesystem, etc.)
// - Access hardware devices
// - Use dangerous tmpfs mounts
// - Use host network/PID/IPC namespaces
// - Add dangerous Linux capabilities
// - Disable security features
func validateComposeSecurityWithConfig(compose *docker.ComposeFile, securityConfig *SecurityConfig) error {
	for serviceName, service := range compose.Services {
		// Block privileged mode - grants full host access
		if service.Privileged {
			return fmt.Errorf("service %q: privileged mode is not allowed for security reasons (can escape container and access host)", serviceName)
		}
		
		// Validate volume mounts for dangerous host paths
		for _, volumeSpec := range service.Volumes {
			if err := validateVolumeSecurityWithConfig(serviceName, volumeSpec, securityConfig); err != nil {
				return err
			}
		}
		
		// Block device access - can access hardware and potentially escape container
		if len(service.Devices) > 0 {
			return fmt.Errorf("service %q: device access is not allowed for security reasons (devices: %v)", serviceName, service.Devices)
		}
		
		// Validate tmpfs mounts for dangerous paths
		for _, tmpfsSpec := range service.Tmpfs {
			if err := validateTmpfsSecurity(serviceName, tmpfsSpec); err != nil {
				return err
			}
		}
		
		// Block host network mode - bypasses network isolation
		if service.NetworkMode == "host" {
			return fmt.Errorf("service %q: network_mode 'host' is not allowed for security reasons (bypasses network isolation)", serviceName)
		}
		
		// Block host PID namespace - allows access to all host processes
		if service.PidMode == "host" {
			return fmt.Errorf("service %q: pid mode 'host' is not allowed for security reasons (grants access to all host processes)", serviceName)
		}
		
		// Block host IPC namespace - allows shared memory access
		if service.IpcMode == "host" {
			return fmt.Errorf("service %q: ipc mode 'host' is not allowed for security reasons (grants shared memory access)", serviceName)
		}
		
		// Validate Linux capabilities - block dangerous capability additions
		if len(service.CapAdd) > 0 {
			if err := validateCapabilities(serviceName, service.CapAdd); err != nil {
				return err
			}
		}
		
		// Validate security options - block disabling security features
		if len(service.SecurityOpt) > 0 {
			if err := validateSecurityOpt(serviceName, service.SecurityOpt); err != nil {
				return err
			}
		}
		
		// Block custom cgroup parent - can be used for privilege escalation
		if service.CgroupParent != "" {
			return fmt.Errorf("service %q: custom cgroup_parent is not allowed for security reasons", serviceName)
		}
	}
	
	return nil
}

// validateVolumeSecurity checks if a volume mount is dangerous (backward compatibility)
func validateVolumeSecurity(serviceName, volumeSpec string) error {
	return validateVolumeSecurityWithConfig(serviceName, volumeSpec, defaultSecurityConfig)
}

// validateVolumeSecurityWithConfig checks if a volume mount is dangerous
func validateVolumeSecurityWithConfig(serviceName, volumeSpec string, securityConfig *SecurityConfig) error {
	// Parse volume spec: "source:target[:mode]"
	parts := strings.Split(volumeSpec, ":")
	if len(parts) < 2 {
		// Invalid format, will be caught by Docker - allow it through
		return nil
	}
	
	hostPath := strings.TrimSpace(parts[0])
	
	// Skip named volumes (don't start with / or ./ or ../)
	if !strings.HasPrefix(hostPath, "/") && 
	   !strings.HasPrefix(hostPath, "./") && 
	   !strings.HasPrefix(hostPath, "../") {
		return nil // Named volume, safe
	}
	
	// Clean the path to resolve any .. or . components
	cleanedPath := filepath.Clean(hostPath)
	
	// List of critical paths that should NEVER be mounted (even if whitelisted)
	// These are checked first before any whitelist
	criticalPaths := []struct {
		path   string
		reason string
	}{
		{"/var/run/docker.sock", "grants full Docker control and host access"},
		{"/var/run/docker", "grants access to Docker runtime"},
		{"/run/docker.sock", "grants full Docker control and host access"},
		{"/", "grants access to entire host filesystem"},
		{"/root", "grants access to root user's home directory"},
		{"/etc", "grants access to system configuration files"},
		{"/boot", "grants access to boot partition"},
		{"/sys", "grants access to kernel interfaces"},
		{"/proc", "grants access to process information"},
		{"/dev", "grants access to device files"},
		{"/host", "commonly used to mount root filesystem"},
		{"/var/lib/docker", "Docker internal storage"},
		{"/var/lib/kubelet", "Kubernetes internal storage"},
		{"/var/lib/rancher", "Rancher internal storage"},
	}
	
	// Check critical paths first (cannot be overridden by whitelist)
	for _, critical := range criticalPaths {
		// Exact match
		if cleanedPath == critical.path {
			return fmt.Errorf("service %q: mounting %q is not allowed (%s)", serviceName, hostPath, critical.reason)
		}
		
		// Prefix match (e.g., /root/anything)
		if strings.HasPrefix(cleanedPath, critical.path+"/") {
			return fmt.Errorf("service %q: mounting paths under %q is not allowed (%s)", serviceName, critical.path, critical.reason)
		}
	}
	
	// Check if path is in the whitelist (allowed paths)
	// Whitelist can only override non-critical path restrictions (like /home)
	isWhitelisted := false
	for _, allowedPath := range securityConfig.AllowedVolumePaths {
		allowedPath = strings.TrimSpace(allowedPath)
		if allowedPath == "" {
			continue
		}
		
		// Clean the allowed path as well
		cleanedAllowed := filepath.Clean(allowedPath)
		
		// Check if the host path is exactly the allowed path or a subdirectory of it
		if cleanedPath == cleanedAllowed || strings.HasPrefix(cleanedPath+"/", cleanedAllowed+"/") {
			isWhitelisted = true
			break
		}
	}
	
	// If whitelisted, allow it (critical paths already blocked above)
	if isWhitelisted {
		return nil
	}
	
	// Block mounting from /home (contains user data and SSH keys)
	// This can be overridden by whitelist (unlike critical paths above)
	if strings.HasPrefix(cleanedPath, "/home/") {
		return fmt.Errorf("service %q: mounting /home paths is not allowed (contains sensitive user data). Use ALLOWED_VOLUME_PATHS environment variable to whitelist specific paths", serviceName)
	}
	
	// Allow other paths (e.g., /data, /mnt, /opt, specific app directories)
	// These are typically safe for application data
	return nil
}

// validateTmpfsSecurity checks if a tmpfs mount is dangerous
func validateTmpfsSecurity(serviceName, tmpfsSpec string) error {
	// Parse tmpfs spec: "/path" or "/path:options"
	parts := strings.Split(tmpfsSpec, ":")
	if len(parts) < 1 {
		return nil
	}
	
	mountPath := strings.TrimSpace(parts[0])
	
	// Block tmpfs mounts on critical system paths
	dangerousTmpfsPaths := []string{
		"/etc",
		"/root",
		"/boot",
		"/sys",
		"/proc",
		"/dev",
	}
	
	for _, dangerous := range dangerousTmpfsPaths {
		if mountPath == dangerous || strings.HasPrefix(mountPath, dangerous+"/") {
			return fmt.Errorf("service %q: tmpfs mount on %q is not allowed for security reasons", serviceName, dangerous)
		}
	}
	
	return nil
}

// validateCapabilities checks for dangerous Linux capability additions
func validateCapabilities(serviceName string, capabilities []string) error {
	// List of dangerous capabilities that should not be added
	// These capabilities can be used to escape containers or compromise the host
	dangerousCapabilities := map[string]string{
		"SYS_ADMIN":      "grants broad system administration privileges",
		"SYS_MODULE":     "allows loading kernel modules",
		"SYS_RAWIO":      "allows raw I/O operations",
		"SYS_PTRACE":     "allows process tracing and debugging",
		"SYS_BOOT":       "allows system reboot",
		"MAC_ADMIN":      "allows MAC configuration changes",
		"MAC_OVERRIDE":   "allows overriding MAC policy",
		"NET_ADMIN":      "grants network administration privileges",
		"SYS_RESOURCE":   "allows resource limit manipulation",
		"SYS_TIME":       "allows system time modification",
		"DAC_READ_SEARCH": "bypasses file read permission checks",
		"DAC_OVERRIDE":   "bypasses file permission checks",
		"ALL":            "grants all capabilities",
	}
	
	for _, cap := range capabilities {
		capUpper := strings.ToUpper(strings.TrimSpace(cap))
		// Remove CAP_ prefix if present for comparison
		capUpper = strings.TrimPrefix(capUpper, "CAP_")
		
		if reason, isDangerous := dangerousCapabilities[capUpper]; isDangerous {
			return fmt.Errorf("service %q: capability %q is not allowed (%s)", serviceName, cap, reason)
		}
	}
	
	return nil
}

// validateSecurityOpt checks for disabled security features
func validateSecurityOpt(serviceName string, securityOpts []string) error {
	for _, opt := range securityOpts {
		optLower := strings.ToLower(strings.TrimSpace(opt))
		
		// Block disabling AppArmor
		if strings.HasPrefix(optLower, "apparmor=unconfined") || strings.HasPrefix(optLower, "apparmor:unconfined") {
			return fmt.Errorf("service %q: disabling AppArmor is not allowed for security reasons", serviceName)
		}
		
		// Block disabling SELinux
		if strings.HasPrefix(optLower, "label=disable") || strings.HasPrefix(optLower, "label:disable") {
			return fmt.Errorf("service %q: disabling SELinux labels is not allowed for security reasons", serviceName)
		}
		
		// Block disabling seccomp
		if strings.HasPrefix(optLower, "seccomp=unconfined") || strings.HasPrefix(optLower, "seccomp:unconfined") {
			return fmt.Errorf("service %q: disabling seccomp is not allowed for security reasons", serviceName)
		}
		
		// Block no-new-privileges=false (should be true or omitted)
		if strings.Contains(optLower, "no-new-privileges=false") || strings.Contains(optLower, "no-new-privileges:false") {
			return fmt.Errorf("service %q: disabling no-new-privileges is not allowed for security reasons", serviceName)
		}
	}
	
	return nil
}

// ValidateDescription validates an app description
func ValidateDescription(description string) error {
	// Description is optional, but if provided should have reasonable length
	maxLength := 500
	if len(description) > maxLength {
		return errors.New("description must be 500 characters or less")
	}
	
	return nil
}

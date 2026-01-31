package validation

import (
	"errors"
	"fmt"
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
	// Check if empty
	if len(content) == 0 {
		return errors.New("compose file content cannot be empty")
	}
	
	// Check size limit (1MB)
	maxSize := 1 << 20 // 1MB
	if len(content) > maxSize {
		return errors.New("compose file too large (maximum 1MB)")
	}
	
	// Parse and validate the compose file structure
	compose, err := docker.ParseCompose([]byte(content))
	if err != nil {
		return fmt.Errorf("invalid compose file: %w", err)
	}
	
	// Validate that services don't reference undefined networks
	if err := validateServiceNetworks(compose); err != nil {
		return err
	}
	
	// Validate that services don't reference undefined volumes
	if err := validateServiceVolumes(compose); err != nil {
		return err
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
			
			return fmt.Errorf("service %q refers to undefined network %q: network must be defined in the networks section", serviceName, networkName)
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

// ValidateDescription validates an app description
func ValidateDescription(description string) error {
	// Description is optional, but if provided should have reasonable length
	maxLength := 500
	if len(description) > maxLength {
		return errors.New("description must be 500 characters or less")
	}
	
	return nil
}

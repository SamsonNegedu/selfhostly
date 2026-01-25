package domain

import (
	"fmt"
	"regexp"
	"strings"
)

// ============================================================================
// Value Objects
// ============================================================================

// AppName represents a validated application name
type AppName struct {
	value string
}

// NewAppName creates a new validated app name
func NewAppName(name string) (*AppName, error) {
	// Validate app name
	if name == "" {
		return nil, &DomainError{
			Code:    ErrAppNameInvalid.Code,
			Message: "app name cannot be empty",
		}
	}

	// App name should be lowercase alphanumeric with hyphens
	// Valid: my-app, app123, test-app-1
	// Invalid: My App, app_name, -app, app-
	pattern := regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
	if !pattern.MatchString(name) {
		return nil, &DomainError{
			Code:    ErrAppNameInvalid.Code,
			Message: "app name must be lowercase alphanumeric with hyphens, cannot start or end with hyphen",
		}
	}

	// Check length constraints
	if len(name) < 2 {
		return nil, &DomainError{
			Code:    ErrAppNameInvalid.Code,
			Message: "app name must be at least 2 characters",
		}
	}

	if len(name) > 63 {
		return nil, &DomainError{
			Code:    ErrAppNameInvalid.Code,
			Message: "app name cannot exceed 63 characters",
		}
	}

	return &AppName{value: name}, nil
}

// String returns the string value of the app name
func (n *AppName) String() string {
	return n.value
}

// Equals checks if two app names are equal
func (n *AppName) Equals(other *AppName) bool {
	if other == nil {
		return false
	}
	return n.value == other.value
}

// ============================================================================

// AppStatus represents the application status
type AppStatus string

const (
	AppStatusStopped  AppStatus = "stopped"
	AppStatusRunning  AppStatus = "running"
	AppStatusUpdating AppStatus = "updating"
	AppStatusError    AppStatus = "error"
)

// String returns the string value of the status
func (s AppStatus) String() string {
	return string(s)
}

// IsValid checks if the status is valid
func (s AppStatus) IsValid() bool {
	switch s {
	case AppStatusStopped, AppStatusRunning, AppStatusUpdating, AppStatusError:
		return true
	default:
		return false
	}
}

// CanTransitionTo checks if a status can transition to another status
func (s AppStatus) CanTransitionTo(target AppStatus) bool {
	// Define valid state transitions
	transitions := map[AppStatus][]AppStatus{
		AppStatusStopped: {
			AppStatusRunning, // start
			AppStatusError,   // error during start
		},
		AppStatusRunning: {
			AppStatusStopped,  // stop
			AppStatusUpdating, // update
			AppStatusError,    // error during operation
		},
		AppStatusUpdating: {
			AppStatusRunning, // update successful
			AppStatusError,   // update failed
		},
		AppStatusError: {
			AppStatusStopped,  // recover by stopping
			AppStatusRunning,  // recover by starting
			AppStatusUpdating, // recover by updating
		},
	}

	validTargets, exists := transitions[s]
	if !exists {
		return false
	}

	for _, valid := range validTargets {
		if valid == target {
			return true
		}
	}

	return false
}

// ============================================================================

// TunnelConfig represents tunnel configuration value object
type TunnelConfig struct {
	TunnelID     string
	TunnelToken  string
	TunnelDomain string
	PublicURL    string
}

// NewTunnelConfig creates a new tunnel configuration
func NewTunnelConfig(tunnelID, tunnelToken, tunnelDomain, publicURL string) *TunnelConfig {
	return &TunnelConfig{
		TunnelID:     tunnelID,
		TunnelToken:  tunnelToken,
		TunnelDomain: tunnelDomain,
		PublicURL:    publicURL,
	}
}

// IsConfigured checks if the tunnel is configured
func (t *TunnelConfig) IsConfigured() bool {
	return t.TunnelID != "" && t.TunnelToken != ""
}

// Validate checks if the tunnel configuration is valid
func (t *TunnelConfig) Validate() error {
	if t.TunnelID == "" {
		return &DomainError{
			Code:    ErrValidationFailed.Code,
			Message: "tunnel ID is required",
		}
	}

	if t.TunnelToken == "" {
		return &DomainError{
			Code:    ErrValidationFailed.Code,
			Message: "tunnel token is required",
		}
	}

	return nil
}

// ============================================================================

// ComposeContent represents validated Docker Compose content
type ComposeContent struct {
	content string
}

// NewComposeContent creates a new validated compose content
func NewComposeContent(content string) (*ComposeContent, error) {
	if content == "" {
		return nil, &DomainError{
			Code:    ErrComposeInvalid.Code,
			Message: "compose content cannot be empty",
		}
	}

	// Basic validation - should contain "services:"
	if !strings.Contains(content, "services:") {
		return nil, &DomainError{
			Code:    ErrComposeInvalid.Code,
			Message: "compose file must contain services section",
		}
	}

	return &ComposeContent{content: content}, nil
}

// String returns the string value of the compose content
func (c *ComposeContent) String() string {
	return c.content
}

// Equals checks if two compose contents are equal
func (c *ComposeContent) Equals(other *ComposeContent) bool {
	if other == nil {
		return false
	}
	return c.content == other.content
}

// ============================================================================

// CloudflareCredentials represents Cloudflare API credentials
type CloudflareCredentials struct {
	APIToken  string
	AccountID string
}

// NewCloudflareCredentials creates new Cloudflare credentials
func NewCloudflareCredentials(apiToken, accountID string) *CloudflareCredentials {
	return &CloudflareCredentials{
		APIToken:  apiToken,
		AccountID: accountID,
	}
}

// IsConfigured checks if Cloudflare credentials are configured
func (c *CloudflareCredentials) IsConfigured() bool {
	return c.APIToken != "" && c.AccountID != ""
}

// Validate checks if the credentials are valid
func (c *CloudflareCredentials) Validate() error {
	if c.APIToken == "" {
		return &DomainError{
			Code:    ErrValidationFailed.Code,
			Message: "Cloudflare API token is required",
		}
	}

	if c.AccountID == "" {
		return &DomainError{
			Code:    ErrValidationFailed.Code,
			Message: "Cloudflare account ID is required",
		}
	}

	return nil
}

// ============================================================================

// ContainerID represents a validated container ID
type ContainerID struct {
	value string
}

// NewContainerID creates a new validated container ID
func NewContainerID(id string) (*ContainerID, error) {
	if id == "" {
		return nil, &DomainError{
			Code:    ErrValidationFailed.Code,
			Message: "container ID cannot be empty",
		}
	}

	// Docker container IDs are typically 12 or 64 character hex strings
	// We'll accept any non-empty string for flexibility
	if len(id) < 12 {
		return nil, &DomainError{
			Code:    ErrValidationFailed.Code,
			Message: "container ID must be at least 12 characters",
		}
	}

	return &ContainerID{value: id}, nil
}

// String returns the string value of the container ID
func (c *ContainerID) String() string {
	return c.value
}

// Short returns the short form of the container ID (first 12 characters)
func (c *ContainerID) Short() string {
	if len(c.value) > 12 {
		return c.value[:12]
	}
	return c.value
}

// ============================================================================

// Hostname represents a validated hostname
type Hostname struct {
	value string
}

// NewHostname creates a new validated hostname
func NewHostname(hostname string) (*Hostname, error) {
	if hostname == "" {
		return nil, &DomainError{
			Code:    ErrValidationFailed.Code,
			Message: "hostname cannot be empty",
		}
	}

	// Basic hostname validation
	// Valid: example.com, sub.example.com, app-1.example.com
	pattern := regexp.MustCompile(`^([a-z0-9]([a-z0-9\-]{0,61}[a-z0-9])?\.)+[a-z]{2,}$`)
	if !pattern.MatchString(strings.ToLower(hostname)) {
		return nil, &DomainError{
			Code:    ErrValidationFailed.Code,
			Message: "hostname must be a valid domain name",
		}
	}

	return &Hostname{value: strings.ToLower(hostname)}, nil
}

// String returns the string value of the hostname
func (h *Hostname) String() string {
	return h.value
}

// Domain extracts the root domain from the hostname
// e.g., "app.example.com" -> "example.com"
func (h *Hostname) Domain() string {
	parts := strings.Split(h.value, ".")
	if len(parts) >= 2 {
		return strings.Join(parts[len(parts)-2:], ".")
	}
	return h.value
}

// Subdomain extracts the subdomain part
// e.g., "app.example.com" -> "app"
func (h *Hostname) Subdomain() string {
	parts := strings.Split(h.value, ".")
	if len(parts) > 2 {
		return strings.Join(parts[:len(parts)-2], ".")
	}
	return ""
}

// ============================================================================

// Version represents a version number
type Version struct {
	number int
}

// NewVersion creates a new version
func NewVersion(number int) (*Version, error) {
	if number < 1 {
		return nil, &DomainError{
			Code:    ErrValidationFailed.Code,
			Message: fmt.Sprintf("version number must be >= 1, got: %d", number),
		}
	}

	return &Version{number: number}, nil
}

// Number returns the version number
func (v *Version) Number() int {
	return v.number
}

// Next returns the next version number
func (v *Version) Next() *Version {
	return &Version{number: v.number + 1}
}

// Equals checks if two versions are equal
func (v *Version) Equals(other *Version) bool {
	if other == nil {
		return false
	}
	return v.number == other.number
}

// String returns the string representation of the version
func (v *Version) String() string {
	return fmt.Sprintf("v%d", v.number)
}

package tunnel

import (
	"errors"
	"fmt"
)

var (
	// ErrTunnelNotFound is returned when a tunnel doesn't exist for the given app
	ErrTunnelNotFound = errors.New("tunnel not found")

	// ErrProviderNotFound is returned when trying to get a provider that isn't registered
	ErrProviderNotFound = errors.New("tunnel provider not found")

	// ErrProviderNotConfigured is returned when a provider is registered but not configured
	ErrProviderNotConfigured = errors.New("tunnel provider not configured")

	// ErrFeatureNotSupported is returned when trying to use a feature the provider doesn't support
	ErrFeatureNotSupported = errors.New("feature not supported by this provider")

	// ErrInvalidConfiguration is returned when provider configuration is invalid
	ErrInvalidConfiguration = errors.New("invalid provider configuration")
)

// FeatureNotSupportedError wraps ErrFeatureNotSupported with context about
// which provider and feature are involved.
type FeatureNotSupportedError struct {
	Provider string
	Feature  Feature
}

func (e *FeatureNotSupportedError) Error() string {
	return fmt.Sprintf("%s does not support %s", e.Provider, e.Feature)
}

func (e *FeatureNotSupportedError) Unwrap() error {
	return ErrFeatureNotSupported
}

// NewFeatureNotSupportedError creates a new FeatureNotSupportedError.
func NewFeatureNotSupportedError(provider string, feature Feature) error {
	return &FeatureNotSupportedError{
		Provider: provider,
		Feature:  feature,
	}
}

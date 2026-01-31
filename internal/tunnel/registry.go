package tunnel

import (
	"fmt"
	"sync"
)

// ProviderFactory is a function that creates a Provider instance given configuration.
// Each registered provider must provide a factory function.
//
// The config parameter contains provider-specific configuration (API keys, regions, etc.)
// that has been unmarshaled from the settings.
type ProviderFactory func(config map[string]interface{}) (Provider, error)

// Registry manages the available tunnel providers and creates provider instances.
// It implements the Factory pattern and provides thread-safe provider registration.
//
// Usage:
//
//	registry := tunnel.NewRegistry()
//	registry.Register("cloudflare", cloudflareProviderFactory)
//	registry.Register("ngrok", ngrokProviderFactory)
//
//	provider, err := registry.GetProvider("cloudflare", config)
type Registry struct {
	providers map[string]ProviderFactory
	mu        sync.RWMutex // Protects providers map
}

// NewRegistry creates a new provider registry.
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]ProviderFactory),
	}
}

// Register adds a provider factory to the registry.
// The name should be lowercase and match the provider's Name() return value.
//
// This is typically called during application initialization:
//
//	func init() {
//	    registry.Register("cloudflare", NewCloudflareProvider)
//	}
func (r *Registry) Register(name string, factory ProviderFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.providers[name] = factory
}

// GetProvider creates a provider instance using the registered factory.
// Returns ErrProviderNotFound if no provider with the given name is registered.
//
// The config parameter is passed directly to the provider's factory function
// and should contain all necessary configuration (API keys, credentials, etc.).
func (r *Registry) GetProvider(name string, config map[string]interface{}) (Provider, error) {
	r.mu.RLock()
	factory, exists := r.providers[name]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrProviderNotFound, name)
	}

	provider, err := factory(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider %s: %w", name, err)
	}

	return provider, nil
}

// IsRegistered checks if a provider with the given name is registered.
func (r *Registry) IsRegistered(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.providers[name]
	return exists
}

// ListProviders returns the names of all registered providers.
// This is useful for API endpoints that need to show available providers.
func (r *Registry) ListProviders() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}

	return names
}

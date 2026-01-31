package tunnel

// CreateOptions contains parameters for creating a new tunnel.
type CreateOptions struct {
	// AppID is the ID of the application to create a tunnel for
	AppID string

	// Name is the tunnel name (often the same as the app name)
	Name string

	// AdditionalConfig contains provider-specific configuration
	// that doesn't fit the common options
	AdditionalConfig map[string]interface{}
}

// DNSOptions contains parameters for creating DNS records.
type DNSOptions struct {
	// Hostname is the subdomain or full hostname (e.g., "myapp" or "myapp.example.com")
	Hostname string

	// Domain is the apex domain (e.g., "example.com")
	Domain string

	// TTL is the DNS record time-to-live (optional, provider decides default)
	TTL int

	// Proxied indicates whether the record should be proxied (Cloudflare-specific)
	Proxied bool
}

package tunnel

// Feature represents a tunnel provider capability.
// Use these constants with SupportsFeature to check if a provider
// implements a specific feature.
type Feature string

const (
	// FeatureIngress indicates the provider supports configurable ingress/routing rules
	FeatureIngress Feature = "ingress"

	// FeatureDNS indicates the provider can manage DNS records
	FeatureDNS Feature = "dns"

	// FeatureStatusSync indicates the provider can sync tunnel status from its API
	FeatureStatusSync Feature = "status_sync"

	// FeatureContainer indicates the provider requires a Docker container sidecar
	FeatureContainer Feature = "container"

	// FeatureList indicates the provider can list all tunnels
	FeatureList Feature = "list"

	// FeatureQuickTunnel indicates the provider supports Quick Tunnels
	// (temporary tunnels without API registration, e.g., Cloudflare's trycloudflare.com)
	FeatureQuickTunnel Feature = "quick_tunnel"
)

// SupportsFeature checks if a provider implements a specific feature
// by performing interface assertions.
//
// This follows Go's "accept interfaces, return structs" principle and
// allows runtime feature detection without reflection.
//
// Example usage:
//
//	if SupportsFeature(provider, tunnel.FeatureIngress) {
//	    ingressProvider := provider.(tunnel.IngressProvider)
//	    err := ingressProvider.UpdateIngress(ctx, appID, rules)
//	}
func SupportsFeature(p Provider, feature Feature) bool {
	if p == nil {
		return false
	}

	switch feature {
	case FeatureIngress:
		_, ok := p.(IngressProvider)
		return ok

	case FeatureDNS:
		_, ok := p.(DNSProvider)
		return ok

	case FeatureStatusSync:
		_, ok := p.(StatusSyncProvider)
		return ok

	case FeatureContainer:
		_, ok := p.(ContainerProvider)
		return ok

	case FeatureList:
		_, ok := p.(ListProvider)
		return ok

	default:
		return false
	}
}

// GetSupportedFeatures returns a map of all features and whether the provider supports them.
// This is useful for API responses to inform clients about provider capabilities.
func GetSupportedFeatures(p Provider) map[Feature]bool {
	return map[Feature]bool{
		FeatureIngress:     SupportsFeature(p, FeatureIngress),
		FeatureDNS:         SupportsFeature(p, FeatureDNS),
		FeatureStatusSync:  SupportsFeature(p, FeatureStatusSync),
		FeatureContainer:   SupportsFeature(p, FeatureContainer),
		FeatureList:        SupportsFeature(p, FeatureList),
		FeatureQuickTunnel: SupportsFeature(p, FeatureQuickTunnel),
	}
}

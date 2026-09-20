package node

import (
	"net/http"

	"github.com/selfhostly/internal/constants"
	"sync"
	"time"

	"github.com/selfhostly/internal/netguard"
)

var (
	policyMu      sync.RWMutex
	networkPolicy *netguard.Policy
	linkTransport http.RoundTripper
)

// SetLinkTransport installs the transport for tunnel://<node-id> endpoints (nodes that dialled out to
// this primary). Call it once at startup, before node clients are created.
func SetLinkTransport(rt http.RoundTripper) {
	policyMu.Lock()
	defer policyMu.Unlock()
	linkTransport = rt
}

// SetNetworkPolicy installs the destination policy used by every inter-node HTTP client created
// afterwards, and by ValidateEndpoint. Call it once at startup, before services are constructed.
func SetNetworkPolicy(p *netguard.Policy) {
	policyMu.Lock()
	defer policyMu.Unlock()
	networkPolicy = p
}

func currentPolicy() *netguard.Policy {
	policyMu.RLock()
	defer policyMu.RUnlock()
	return networkPolicy
}

// ValidateEndpoint checks a node API endpoint against the installed policy. Without a policy only
// the scheme and shape are checked.
func ValidateEndpoint(raw string) error {
	p := currentPolicy()
	if p == nil {
		p, _ = netguard.NewPolicy(true, nil)
	}
	return p.ValidateEndpoint(raw)
}

// nodeHTTPClient builds the client used to call other nodes. With a policy installed it refuses
// link-local and other forbidden destinations at connect time; redirects are never followed
// because they would carry node credentials to a host that was not vetted.
func nodeHTTPClient(timeout time.Duration) *http.Client {
	var client *http.Client
	if p := currentPolicy(); p != nil {
		client = p.NewHTTPClient(timeout)
	} else {
		fallback, _ := netguard.NewPolicy(true, nil)
		client = fallback.NewHTTPClient(timeout)
	}
	policyMu.RLock()
	rt := linkTransport
	policyMu.RUnlock()
	if tr, ok := client.Transport.(*http.Transport); ok && rt != nil {
		tr.RegisterProtocol(constants.LinkEndpointScheme, rt)
	}
	return client
}

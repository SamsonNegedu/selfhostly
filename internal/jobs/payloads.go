package jobs

// AppCreatePayload contains data for app_create jobs
type AppCreatePayload struct {
	Name               string        `json:"name"`
	Description        string        `json:"description"`
	ComposeContent     string        `json:"compose_content"`
	IngressRules       []IngressRule `json:"ingress_rules,omitempty"`
	TunnelMode         string        `json:"tunnel_mode,omitempty"`
	QuickTunnelService string        `json:"quick_tunnel_service,omitempty"`
	QuickTunnelPort    int           `json:"quick_tunnel_port,omitempty"`
	AutoStart          bool          `json:"auto_start"`
	NodeID             string        `json:"node_id,omitempty"`
}

// AppUpdatePayload contains data for app_update jobs
type AppUpdatePayload struct {
	// Currently empty, but reserved for future options like:
	// ForceRecreate bool `json:"force_recreate,omitempty"`
	// NoCache       bool `json:"no_cache,omitempty"`
}

// TunnelCreatePayload contains data for tunnel_create jobs
type TunnelCreatePayload struct {
	IngressRules []IngressRule `json:"ingress_rules,omitempty"`
}

// QuickTunnelPayload contains data for quick_tunnel jobs
type QuickTunnelPayload struct {
	Service string `json:"service"`
	Port    int    `json:"port"`
}

// IngressRule represents a tunnel ingress rule
type IngressRule struct {
	Hostname      *string                `json:"hostname,omitempty"`
	Service       string                 `json:"service"`
	Path          *string                `json:"path,omitempty"`
	OriginRequest map[string]interface{} `json:"originRequest,omitempty"`
}
